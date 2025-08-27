package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

var (
	matrixClient *mautrix.Client
	beeperAuth   *BeeperAuth
)

// GetMatrixClient returns a connected Matrix client
func GetMatrixClient() (*mautrix.Client, error) {
	if matrixClient != nil {
		return matrixClient, nil
	}

	// Check if we should use Beeper authentication
	useBeeperAuth := os.Getenv("USE_BEEPER_AUTH") == "true" ||
		os.Getenv("BEEPER_TOKEN") != "" ||
		os.Getenv("BEEPER_EMAIL") != ""

	if useBeeperAuth {
		return getBeeperMatrixClient()
	}

	return getStandardMatrixClient()
}

// getBeeperMatrixClient creates a Matrix client using Beeper authentication
func getBeeperMatrixClient() (*mautrix.Client, error) {
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

	client, err := beeperAuth.GetMatrixClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Beeper Matrix client: %w", err)
	}

	matrixClient = client
	log.Printf("Logged in via Beeper as %s", beeperAuth.Whoami.UserInfo.Username)
	return matrixClient, nil
}

// getStandardMatrixClient creates a Matrix client using traditional username/password
func getStandardMatrixClient() (*mautrix.Client, error) {
	matrixUser := os.Getenv("MATRIX_USER")
	matrixPassword := os.Getenv("MATRIX_PASSWORD")
	matrixHost := os.Getenv("MATRIX_HOST")

	if matrixUser == "" {
		return nil, fmt.Errorf("MATRIX_USER environment variable is required")
	}
	if matrixPassword == "" {
		return nil, fmt.Errorf("MATRIX_PASSWORD environment variable is required")
	}
	if matrixHost == "" {
		matrixHost = "https://matrix.org"
	}

	fmt.Printf("Signing into %s...\n", matrixHost)

	// Create Matrix client
	userID := id.UserID(matrixUser)
	client, err := mautrix.NewClient(matrixHost, userID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create Matrix client: %w", err)
	}

	// Login
	resp, err := client.Login(context.Background(), &mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: matrixUser,
		},
		Password: matrixPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to login to Matrix: %w", err)
	}

	client.AccessToken = resp.AccessToken
	client.UserID = resp.UserID
	matrixClient = client

	log.Printf("Logged in as %s", resp.UserID)
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

// GetMatrixRoomIDs returns the configured room IDs from environment
func GetMatrixRoomIDs() ([]string, error) {
	roomIDsEnv := os.Getenv("MATRIX_ROOM_IDS")
	if roomIDsEnv == "" {
		return nil, fmt.Errorf("MATRIX_ROOM_IDS environment variable is required")
	}

	roomIDs := strings.Split(roomIDsEnv, ",")
	for i, roomID := range roomIDs {
		roomIDs[i] = strings.TrimSpace(roomID)
	}

	return roomIDs, nil
}
