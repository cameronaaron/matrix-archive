package archive

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"maunium.net/go/mautrix"
)

var (
	matrixClient *mautrix.Client
	beeperAuth   *BeeperAuth
)

// GetMatrixClient returns a connected Matrix client using Beeper authentication
func GetMatrixClient() (*mautrix.Client, error) {
	if matrixClient != nil {
		return matrixClient, nil
	}

	return GetBeeperMatrixClient()
}

// GetBeeperMatrixClient creates a Matrix client using Beeper authentication with crypto
func GetBeeperMatrixClient() (*mautrix.Client, error) {
	baseDomain := os.Getenv("BEEPER_DOMAIN")
	if baseDomain == "" {
		baseDomain = "beeper.com"
	}

	if beeperAuth == nil {
		beeperAuth = NewBeeperAuth(baseDomain)
	}

	// Try to load existing credentials
	if !beeperAuth.LoadCredentials() {
		// If no valid credentials, perform login flow
		fmt.Println("No valid Beeper credentials found. Starting login process...")
		if err := beeperAuth.Login(); err != nil {
			return nil, fmt.Errorf("Beeper login failed: %w", err)
		}
		// Save credentials for future use
		beeperAuth.SaveCredentials()
	}

	// Get Matrix client with crypto using the helper approach
	client, err := beeperAuth.GetMatrixClientWithCrypto()
	if err != nil {
		// If this fails due to expired token, clear credentials and try login again
		if strings.Contains(err.Error(), "expired_token") || strings.Contains(err.Error(), "M_FORBIDDEN") {
			fmt.Println("Beeper credentials expired. Re-authenticating...")
			beeperAuth.ClearCredentials()

			// Perform fresh login
			if err := beeperAuth.Login(); err != nil {
				return nil, fmt.Errorf("Beeper re-authentication failed: %w", err)
			}
			beeperAuth.SaveCredentials()

			// Try again with fresh credentials
			client, err = beeperAuth.GetMatrixClientWithCrypto()
			if err != nil {
				return nil, fmt.Errorf("failed to create Beeper Matrix client after re-auth: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to create Beeper Matrix client: %w", err)
		}
	}

	// Create crypto manager with database path
	// Use a default crypto database path
	cryptoDbPath := "./crypto_store"
	cryptoManager, err := NewCryptoManager(client, cryptoDbPath)
	if err != nil {
		log.Printf("Warning: Failed to initialize crypto: %v", err)
		// Continue without crypto rather than failing completely
	} else {
		// Start the crypto manager
		ctx := context.Background()
		if err := cryptoManager.Start(ctx); err != nil {
			log.Printf("Warning: Failed to start crypto manager: %v", err)
		} else {
			// Assign the crypto manager (which implements CryptoHelper) to the client
			client.Crypto = cryptoManager
			log.Printf("Crypto-enabled Matrix client initialized successfully")
		}
	}

	// Always save credentials after successfully getting a Matrix client
	beeperAuth.SaveCredentials()

	matrixClient = client
	log.Printf("Logged in via Beeper as %s", beeperAuth.Whoami.UserInfo.Username)
	return matrixClient, nil
}

// GetDownloadURL converts an mxc:// URL to an HTTP download URL
func GetDownloadURL(mxcURL string) (string, error) {
	if !strings.HasPrefix(mxcURL, "mxc://") {
		return "", fmt.Errorf("invalid mxc URL: %s", mxcURL)
	}

	client, err := GetMatrixClient()
	if err != nil {
		return "", err
	}

	// Parse the mxc URL
	u, err := url.Parse(mxcURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse mxc URL: %w", err)
	}

	// Convert to download URL using the client's homeserver
	downloadURL := client.BuildURL(mautrix.ClientURLPath{"_matrix", "media", "r0", "download", u.Host, u.Path[1:]})
	return downloadURL, nil
}

// GetMatrixDeviceID returns the device ID from the current beeper auth
func GetMatrixDeviceID() string {
	if beeperAuth != nil {
		return beeperAuth.GetMatrixDeviceID()
	}
	return ""
}


