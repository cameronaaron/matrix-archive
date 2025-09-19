package archive

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/util/dbutil"
	_ "go.mau.fi/util/dbutil/litestream"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type CryptoManager struct {
	olmMachine  *crypto.OlmMachine
	cryptoStore *crypto.SQLCryptoStore
	client      *mautrix.Client
	userID      id.UserID
	deviceID    id.DeviceID
}

type SimpleStateStore struct{}

func (s *SimpleStateStore) GetMembership(ctx context.Context, roomID id.RoomID, userID id.UserID) (event.Membership, error) {
	return event.MembershipJoin, nil
}

func (s *SimpleStateStore) IsEncrypted(ctx context.Context, roomID id.RoomID) (bool, error) {
	return true, nil
}

func (s *SimpleStateStore) GetEncryptionEvent(ctx context.Context, roomID id.RoomID) (*event.EncryptionEventContent, error) {
	return &event.EncryptionEventContent{
		Algorithm: id.AlgorithmMegolmV1,
	}, nil
}

func (s *SimpleStateStore) FindSharedRooms(ctx context.Context, userID id.UserID) ([]id.RoomID, error) {
	return []id.RoomID{}, nil
}

func NewCryptoManager(client *mautrix.Client, dbPath string) (*CryptoManager, error) {
	// Override device ID to be deterministic before creating crypto helper
	deterministic_device_id := "MATRIXARCH"
	client.DeviceID = id.DeviceID(deterministic_device_id)

	// Create SQL crypto store like gomuks
	cryptoDBPath := dbPath + "_crypto.db"
	cryptoDB, err := dbutil.NewWithDialect(cryptoDBPath, "sqlite3")
	if err != nil {
		return nil, fmt.Errorf("failed to open crypto database: %w", err)
	}
	cryptoDB.Owner = "matrix-archive"
	cryptoDB.Log = dbutil.ZeroLogger(zerolog.New(log.Writer()))

	// Upgrade the database schema
	err = cryptoDB.Upgrade(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade crypto database: %w", err)
	}

	cryptoStore := crypto.NewSQLCryptoStore(cryptoDB, dbutil.ZeroLogger(zerolog.New(log.Writer())), "", "", []byte("matrix-archive-crypto"))

	// Set the account info on the crypto store
	cryptoStore.AccountID = client.UserID.String()
	cryptoStore.DeviceID = client.DeviceID

	// Upgrade the crypto store schema
	err = cryptoStore.DB.Upgrade(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade crypto store schema: %w", err)
	}

	// Create simple state store for crypto
	stateStore := &SimpleStateStore{}

	// Create OlmMachine directly like gomuks
	olmMachine := crypto.NewOlmMachine(client, nil, cryptoStore, stateStore)
	olmMachine.DisableRatchetTracking = true
	olmMachine.DisableDecryptKeyFetching = true
	olmMachine.IgnorePostDecryptionParseErrors = true

	return &CryptoManager{
		olmMachine:  olmMachine,
		cryptoStore: cryptoStore,
		client:      client,
		userID:      client.UserID,
		deviceID:    client.DeviceID,
	}, nil
}

func (cm *CryptoManager) Start(ctx context.Context) error {
	// Initialize using the CryptoHelper interface
	err := cm.Init(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize crypto machine: %w", err)
	}

	log.Println("Crypto machine initialized successfully")
	return nil
}

func (cm *CryptoManager) DecryptEvent(ctx context.Context, evt *event.Event) (*event.Event, error) {
	if cm.olmMachine == nil {
		return nil, fmt.Errorf("crypto machine not initialized")
	}

	decrypted, err := cm.olmMachine.DecryptMegolmEvent(ctx, evt)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt event: %w", err)
	}

	return decrypted, nil
}

func (cm *CryptoManager) GetOlmMachine() *crypto.OlmMachine {
	return cm.olmMachine
}

// Implement the mautrix.CryptoHelper interface to wrap the OlmMachine
func (cm *CryptoManager) Init(ctx context.Context) error {
	// Load the crypto account and keys
	return cm.olmMachine.Load(ctx)
}

func (cm *CryptoManager) Encrypt(ctx context.Context, roomID id.RoomID, evtType event.Type, content any) (*event.EncryptedEventContent, error) {
	return cm.olmMachine.EncryptMegolmEvent(ctx, roomID, evtType, content)
}

func (cm *CryptoManager) Decrypt(ctx context.Context, evt *event.Event) (*event.Event, error) {
	return cm.olmMachine.DecryptMegolmEvent(ctx, evt)
}

func (cm *CryptoManager) WaitForSession(ctx context.Context, roomID id.RoomID, senderKey id.SenderKey, sessionID id.SessionID, timeout time.Duration) bool {
	return cm.olmMachine.WaitForSession(ctx, roomID, senderKey, sessionID, timeout)
}

func (cm *CryptoManager) RequestSession(ctx context.Context, roomID id.RoomID, senderKey id.SenderKey, sessionID id.SessionID, userID id.UserID, deviceID id.DeviceID) {
	err := cm.olmMachine.SendRoomKeyRequest(ctx, roomID, senderKey, sessionID, "", map[id.UserID][]id.DeviceID{
		userID: {deviceID},
	})
	if err != nil {
		log.Printf("Failed to send room key request: %v", err)
	}
}
