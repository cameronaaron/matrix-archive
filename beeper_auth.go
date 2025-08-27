package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"

	"matrix-archive/beeperapi"
)

// BeeperAuth handles Beeper authentication
type BeeperAuth struct {
	BaseDomain   string
	Email        string
	Token        string
	Whoami       *beeperapi.RespWhoami
	MatrixToken  string // Cache the Matrix access token
	MatrixUserID string // Cache the Matrix user ID
}

// BeeperCredentials represents saved credentials
type BeeperCredentials struct {
	BaseDomain   string                `json:"base_domain"`
	Email        string                `json:"email"`
	Token        string                `json:"token"`
	Username     string                `json:"username"`
	Whoami       *beeperapi.RespWhoami `json:"whoami,omitempty"`
	MatrixToken  string                `json:"matrix_token,omitempty"`
	MatrixUserID string                `json:"matrix_user_id,omitempty"`
}

// NewBeeperAuth creates a new BeeperAuth instance
func NewBeeperAuth(baseDomain string) *BeeperAuth {
	if baseDomain == "" {
		baseDomain = "beeper.com"
	}
	return &BeeperAuth{
		BaseDomain: baseDomain,
	}
}

// Login performs the Beeper authentication flow
func (b *BeeperAuth) Login() error {
	// Check if we're in an interactive terminal early
	if !isTerminalInteractive() {
		return fmt.Errorf("cannot perform interactive login in non-interactive mode - please run 'matrix-archive beeper-login' in a terminal")
	}

	fmt.Printf("Starting Beeper login for %s...\n", b.BaseDomain)

	// Start login
	loginResp, err := beeperapi.StartLogin(b.BaseDomain)
	if err != nil {
		return fmt.Errorf("failed to start login: %w", err)
	}

	fmt.Printf("Login session started. Request ID: %s\n", loginResp.RequestID)

	// Get email from user
	if b.Email == "" {
		b.Email, err = b.promptEmail()
		if err != nil {
			return fmt.Errorf("failed to get email: %w", err)
		}
	}

	// Send login email
	fmt.Printf("Sending login email to %s...\n", b.Email)
	err = beeperapi.SendLoginEmail(b.BaseDomain, loginResp.RequestID, b.Email)
	if err != nil {
		return fmt.Errorf("failed to send login email: %w", err)
	}

	fmt.Println("Check your email for the login code.")

	// Get login code from user
	code, err := b.promptLoginCode()
	if err != nil {
		return fmt.Errorf("failed to get login code: %w", err)
	}

	// Send login code
	fmt.Println("Verifying login code...")
	codeResp, err := beeperapi.SendLoginCode(b.BaseDomain, loginResp.RequestID, code)
	if err != nil {
		return fmt.Errorf("failed to verify login code: %w", err)
	}

	b.Token = codeResp.LoginToken
	b.Whoami = codeResp.Whoami

	fmt.Printf("Successfully logged in as %s (%s)\n",
		b.Whoami.UserInfo.Username,
		b.Whoami.UserInfo.Email)

	return nil
}

// GetMatrixClient creates a Matrix client using Beeper authentication
func (b *BeeperAuth) GetMatrixClient() (*mautrix.Client, error) {
	if b.Token == "" || b.Whoami == nil {
		return nil, fmt.Errorf("not authenticated - call Login() first")
	}

	// The correct Matrix server for Beeper
	matrixHost := "https://matrix.beeper.com"

	// Try to use cached Matrix token first
	if b.MatrixToken != "" && b.MatrixUserID != "" {
		userID := id.UserID(b.MatrixUserID)
		client, err := mautrix.NewClient(matrixHost, userID, b.MatrixToken)
		if err == nil {
			// Test the connection
			_, err = client.Whoami(context.Background())
			if err == nil {
				fmt.Printf("Matrix connection verified using cached token. User ID: %s\n", b.MatrixUserID)
				return client, nil
			}
		}
		// Cached token failed, clear it and get a new one
		fmt.Printf("Cached Matrix token invalid, getting new one...\n")
		b.MatrixToken = ""
		b.MatrixUserID = ""
	}

	// Use the Beeper JWT token to get a proper Matrix access token
	matrixLogin, err := beeperapi.GetMatrixTokenFromJWT(b.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to get Matrix access token from Beeper JWT: %w", err)
	}

	// Cache the Matrix token for future use
	b.MatrixToken = matrixLogin.AccessToken
	b.MatrixUserID = matrixLogin.UserID

	// Create Matrix client with the proper access token
	userID := id.UserID(matrixLogin.UserID)
	client, err := mautrix.NewClient(matrixHost, userID, matrixLogin.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Matrix client: %w", err)
	}

	// Test the connection
	whoamiResp, err := client.Whoami(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to verify Matrix connection: %w", err)
	}

	fmt.Printf("Matrix connection verified. User ID: %s\n", whoamiResp.UserID)
	return client, nil
}

// promptEmail prompts the user for their email address
func (b *BeeperAuth) promptEmail() (string, error) {
	// Check if we're in an interactive terminal
	if !isTerminalInteractive() {
		return "", fmt.Errorf("cannot prompt for email in non-interactive mode - please run 'matrix-archive beeper-login' directly")
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your Beeper email address: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(email), nil
}

// promptLoginCode prompts the user for their login code
func (b *BeeperAuth) promptLoginCode() (string, error) {
	// Check if we're in an interactive terminal
	if !isTerminalInteractive() {
		return "", fmt.Errorf("cannot prompt for login code in non-interactive mode - please run 'matrix-archive beeper-login' directly")
	}

	fmt.Print("Enter the login code from your email: ")

	// Use regular visible input instead of hidden input
	reader := bufio.NewReader(os.Stdin)
	code, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

// isTerminalInteractive checks if we're running in an interactive terminal
func isTerminalInteractive() bool {
	// If stdout is being piped/redirected, we're likely not interactive
	stdoutStat, stdoutErr := os.Stdout.Stat()
	if stdoutErr != nil {
		return false
	}

	// If stdout is not a character device, we're being piped
	stdoutIsTerminal := (stdoutStat.Mode() & os.ModeCharDevice) != 0

	return stdoutIsTerminal
}

// SaveCredentials saves the authentication credentials to both environment variables and file
func (b *BeeperAuth) SaveCredentials() {
	// Save to environment variables (for current session)
	if b.Token != "" {
		os.Setenv("BEEPER_TOKEN", b.Token)
	}
	if b.Email != "" {
		os.Setenv("BEEPER_EMAIL", b.Email)
	}
	if b.Whoami != nil {
		os.Setenv("BEEPER_USERNAME", b.Whoami.UserInfo.Username)
		os.Setenv("MATRIX_USER", fmt.Sprintf("@%s:%s", b.Whoami.UserInfo.Username, b.BaseDomain))
	}

	// Save to file (for persistent storage)
	if err := b.saveCredentialsToFile(); err != nil {
		fmt.Printf("Warning: Failed to save credentials to file: %v\n", err)
	} else {
		fmt.Println("Credentials saved to file for future use.")
	}
}

// LoadCredentials loads authentication credentials from file and environment variables
func (b *BeeperAuth) LoadCredentials() bool {
	// First try to load from environment variables
	envToken := os.Getenv("BEEPER_TOKEN")
	envEmail := os.Getenv("BEEPER_EMAIL")

	// Then try to load from file
	fileLoaded := b.loadCredentialsFromFile()

	// Use environment variables if they exist and are different from file
	if envToken != "" && envToken != b.Token {
		b.Token = envToken
		b.Email = envEmail
		fileLoaded = false // Don't trust file data if env vars are different
	}

	if b.Token != "" {
		// Just check if we have the basic token info, don't validate with API call
		// The Matrix client creation will validate if the token actually works
		if b.Whoami == nil {
			// Try to get whoami info if we don't have it, but don't fail if it doesn't work
			whoami, err := beeperapi.Whoami(b.BaseDomain, b.Token)
			if err == nil {
				b.Whoami = whoami
			}
			// Don't clear credentials if this fails - the token might still work for Matrix
		}
		return true
	}

	return fileLoaded && b.Token != ""
}

// getCredentialsFilePath returns the path to the credentials file
func (b *BeeperAuth) getCredentialsFilePath() (string, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create .matrix-archive directory if it doesn't exist
	configDir := filepath.Join(homeDir, ".matrix-archive")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	// Return path to credentials file
	filename := fmt.Sprintf("beeper-credentials-%s.json", b.BaseDomain)
	return filepath.Join(configDir, filename), nil
}

// saveCredentialsToFile saves credentials to a JSON file
func (b *BeeperAuth) saveCredentialsToFile() error {
	filePath, err := b.getCredentialsFilePath()
	if err != nil {
		return err
	}

	creds := BeeperCredentials{
		BaseDomain:   b.BaseDomain,
		Email:        b.Email,
		Token:        b.Token,
		Whoami:       b.Whoami,
		MatrixToken:  b.MatrixToken,
		MatrixUserID: b.MatrixUserID,
	}

	if b.Whoami != nil {
		creds.Username = b.Whoami.UserInfo.Username
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write with secure permissions (only readable by owner)
	err = os.WriteFile(filePath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	fmt.Printf("Credentials saved to: %s\n", filePath)
	return nil
}

// loadCredentialsFromFile loads credentials from a JSON file
func (b *BeeperAuth) loadCredentialsFromFile() bool {
	filePath, err := b.getCredentialsFilePath()
	if err != nil {
		return false
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Warning: Failed to read credentials file: %v\n", err)
		return false
	}

	var creds BeeperCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		fmt.Printf("Warning: Failed to parse credentials file: %v\n", err)
		return false
	}

	// Only load if the domain matches
	if creds.BaseDomain != b.BaseDomain {
		return false
	}

	b.Email = creds.Email
	b.Token = creds.Token
	b.Whoami = creds.Whoami
	b.MatrixToken = creds.MatrixToken
	b.MatrixUserID = creds.MatrixUserID

	fmt.Printf("Loaded credentials for %s from file\n", creds.Email)
	return true
}

// clearInvalidCredentials removes invalid credentials from storage
func (b *BeeperAuth) clearInvalidCredentials() {
	// Clear in-memory credentials
	b.Token = ""
	b.Whoami = nil
	b.MatrixToken = ""
	b.MatrixUserID = ""

	// Clear environment variables
	os.Unsetenv("BEEPER_TOKEN")
	os.Unsetenv("BEEPER_USERNAME")

	// Remove credentials file
	if filePath, err := b.getCredentialsFilePath(); err == nil {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: Failed to remove invalid credentials file: %v\n", err)
		} else {
			fmt.Println("Removed invalid credentials file")
		}
	}
}

// ClearCredentials removes all stored credentials (for logout)
func (b *BeeperAuth) ClearCredentials() error {
	b.clearInvalidCredentials()

	// Also clear email since we're doing a full logout
	b.Email = ""
	os.Unsetenv("BEEPER_EMAIL")

	fmt.Println("All Beeper credentials cleared")
	return nil
}
