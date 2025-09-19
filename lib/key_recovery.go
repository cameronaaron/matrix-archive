package archive

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/backup"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func RecoverKeysFromBackup(recoveryKey string) error {
	// Get a Beeper Matrix client
	client, err := GetBeeperMatrixClient()
	if err != nil {
		return fmt.Errorf("failed to get Matrix client: %w", err)
	}

	// Initialize crypto manager with client using the same path as import
	cryptoDbPath := "./crypto_store"
	cryptoManager, err := NewCryptoManager(client, cryptoDbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize crypto manager: %w", err)
	}

	// Get the OlmMachine
	olmMachine := cryptoManager.GetOlmMachine()

	// Parse recovery key from Base58 encoding (Matrix standard format)
	// Remove spaces and use the raw key for verification
	cleanKey := strings.ReplaceAll(recoveryKey, " ", "")

	ctx := context.Background()

	// Get SSSS key data
	keyID, keyData, err := olmMachine.SSSS.GetDefaultKeyData(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SSSS key data: %w", err)
	}

	// Verify recovery key directly (no need to decode Base58 first)
	key, err := keyData.VerifyRecoveryKey(keyID, cleanKey)
	if err != nil {
		return fmt.Errorf("failed to verify recovery key: %w", err)
	}

	// Fetch cross-signing keys from SSSS (this is crucial!)
	err = olmMachine.FetchCrossSigningKeysFromSSSS(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to fetch cross-signing keys from SSSS: %w", err)
	}

	// Get latest backup version
	latestVersion, err := olmMachine.Client.GetKeyBackupLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest backup version: %w", err)
	}

	// Fetch and decrypt backup key from SSSS using the proper event type
	backupKeyData, err := olmMachine.SSSS.GetDecryptedAccountData(ctx, event.AccountDataMegolmBackupKey, key)
	if err != nil {
		return fmt.Errorf("failed to get megolm backup key from SSSS: %w", err)
	}

	backupKey, err := backup.MegolmBackupKeyFromBytes(backupKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse megolm backup key: %w", err)
	}

	// Store backup key in crypto store
	err = cryptoManager.cryptoStore.PutSecret(ctx, id.SecretMegolmBackupV1, base64.StdEncoding.EncodeToString(backupKey.Bytes()))
	if err != nil {
		return fmt.Errorf("failed to store backup key: %w", err)
	}

	log.Printf("Fetching and importing room keys from backup version %s", latestVersion.Version)

	// Get all rooms from backup and import keys
	roomKeys, err := olmMachine.Client.GetKeyBackup(ctx, latestVersion.Version)
	if err != nil {
		return fmt.Errorf("failed to get room keys from backup: %w", err)
	}

	imported := 0
	failed := 0

	// Collect sessions before saving them
	type sessionEntry struct {
		roomID    id.RoomID
		sessionID id.SessionID
		entry     *crypto.InboundGroupSession
	}
	var sessions []sessionEntry

	for roomID, roomData := range roomKeys.Rooms {
		// Get encryption event for this room (needed for session import)
		encEvent := &event.EncryptionEventContent{
			Algorithm: id.AlgorithmMegolmV1,
		}

		for sessionID, sessionData := range roomData.Sessions {
			// Decrypt session data using backup key
			decrypted, err := sessionData.SessionData.Decrypt(backupKey)
			if err != nil {
				log.Printf("Failed to decrypt session %s in room %s: %v", sessionID, roomID, err)
				failed++
				continue
			}

			// Import room key from backup (this returns the session entry but doesn't save it)
			importedSession, err := olmMachine.ImportRoomKeyFromBackupWithoutSaving(ctx, latestVersion.Version, id.RoomID(roomID), encEvent, id.SessionID(sessionID), decrypted)
			if err != nil {
				log.Printf("Failed to import session %s in room %s: %v", sessionID, roomID, err)
				failed++
				continue
			}

			// Collect the session for later saving
			sessions = append(sessions, sessionEntry{
				roomID:    id.RoomID(roomID),
				sessionID: id.SessionID(sessionID),
				entry:     importedSession,
			})
			imported++
		}
	}

	// Save all sessions to the crypto store in a transaction-like manner
	log.Printf("Saving %d imported sessions to database...", len(sessions))
	for _, session := range sessions {
		err := cryptoManager.cryptoStore.PutGroupSession(ctx, session.entry)
		if err != nil {
			log.Printf("Failed to save session %s in room %s: %v", session.sessionID, session.roomID, err)
			continue
		}
	}

	// Flush any remaining changes
	err = cryptoManager.cryptoStore.Flush(ctx)
	if err != nil {
		return fmt.Errorf("failed to save imported sessions: %w", err)
	}

	log.Printf("Successfully recovered keys from backup: %d imported, %d failed", imported, failed)
	return nil
}

// PerformKeyRecovery is the main entry point for key recovery
func PerformKeyRecovery(recoveryKey string, roomID string) error {
	log.Printf("Starting key recovery process...")
	if err := RecoverKeysFromBackup(recoveryKey); err != nil {
		return fmt.Errorf("failed to recover keys: %w", err)
	}

	return nil
}
