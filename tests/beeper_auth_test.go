package tests

import (
	archive "github.com/osteele/matrix-archive/lib"

	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osteele/matrix-archive/internal/beeperapi"
)

func TestNewBeeperAuth(t *testing.T) {
	tests := []struct {
		name           string
		baseDomain     string
		expectedDomain string
	}{
		{
			name:           "Default domain",
			baseDomain:     "",
			expectedDomain: "beeper.com",
		},
		{
			name:           "Custom domain",
			baseDomain:     "custom.beeper.com",
			expectedDomain: "custom.beeper.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := archive.NewBeeperAuth(tt.baseDomain)
			assert.Equal(t, tt.expectedDomain, auth.BaseDomain)
		})
	}
}

func TestIsTerminalInteractive(t *testing.T) {
	// This test is environment dependent, so we'll just ensure it doesn't panic
	result := archive.IsTerminalInteractive()
	assert.IsType(t, false, result) // Just check it returns a boolean
}

func TestBeeperAuth_GetCredentialsFilePath(t *testing.T) {
	auth := archive.NewBeeperAuth("test.com")

	path, err := auth.GetCredentialsFilePath()
	assert.NoError(t, err)
	assert.Contains(t, path, ".matrix-archive")
	assert.Contains(t, path, "beeper-credentials-test.com.json")
}

func TestBeeperAuth_SaveAndLoadCredentials(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "matrix-archive-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Mock the home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	auth := archive.NewBeeperAuth("test.com")
	auth.Email = "test@example.com"
	auth.Token = "test-token"
	auth.MatrixToken = "matrix-token"
	auth.MatrixUserID = "@test:test.com"
	auth.Whoami = &beeperapi.RespWhoami{
		UserInfo: beeperapi.WhoamiUserInfo{
			Username: "testuser",
			Email:    "test@example.com",
		},
	}

	// Test saving credentials
	err = auth.SaveCredentialsToFile()
	assert.NoError(t, err)

	// Verify file exists
	filePath, err := auth.GetCredentialsFilePath()
	assert.NoError(t, err)
	assert.FileExists(t, filePath)

	// Test loading credentials
	newAuth := archive.NewBeeperAuth("test.com")
	loaded := newAuth.LoadCredentialsFromFile()
	assert.True(t, loaded)
	assert.Equal(t, auth.Email, newAuth.Email)
	assert.Equal(t, auth.Token, newAuth.Token)
	assert.Equal(t, auth.MatrixToken, newAuth.MatrixToken)
	assert.Equal(t, auth.MatrixUserID, newAuth.MatrixUserID)
}

func TestBeeperAuth_LoadCredentials_EnvironmentVariables(t *testing.T) {
	// Save original env vars
	originalToken := os.Getenv("BEEPER_TOKEN")
	originalEmail := os.Getenv("BEEPER_EMAIL")
	defer func() {
		if originalToken != "" {
			os.Setenv("BEEPER_TOKEN", originalToken)
		} else {
			os.Unsetenv("BEEPER_TOKEN")
		}
		if originalEmail != "" {
			os.Setenv("BEEPER_EMAIL", originalEmail)
		} else {
			os.Unsetenv("BEEPER_EMAIL")
		}
	}()

	// Set test env vars
	os.Setenv("BEEPER_TOKEN", "env-token")
	os.Setenv("BEEPER_EMAIL", "env@example.com")

	auth := archive.NewBeeperAuth("test.com")
	loaded := auth.LoadCredentials()

	assert.True(t, loaded)
	assert.Equal(t, "env-token", auth.Token)
	assert.Equal(t, "env@example.com", auth.Email)
}

func TestBeeperAuth_ClearCredentials(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "matrix-archive-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Mock the home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	auth := archive.NewBeeperAuth("test.com")
	auth.Email = "test@example.com"
	auth.Token = "test-token"

	// Save credentials first
	err = auth.SaveCredentialsToFile()
	assert.NoError(t, err)

	// Set env vars
	os.Setenv("BEEPER_TOKEN", "test-token")
	os.Setenv("BEEPER_EMAIL", "test@example.com")

	// Clear credentials
	err = auth.ClearCredentials()
	assert.NoError(t, err)

	// Verify everything is cleared
	assert.Empty(t, auth.Email)
	assert.Empty(t, auth.Token)
	assert.Empty(t, os.Getenv("BEEPER_TOKEN"))
	assert.Empty(t, os.Getenv("BEEPER_EMAIL"))

	// Verify file is removed
	filePath, err := auth.GetCredentialsFilePath()
	assert.NoError(t, err)
	assert.NoFileExists(t, filePath)
}

func TestBeeperAuth_SaveCredentials_EnvironmentVariables(t *testing.T) {
	// Save original env vars
	originalToken := os.Getenv("BEEPER_TOKEN")
	originalEmail := os.Getenv("BEEPER_EMAIL")
	originalUsername := os.Getenv("BEEPER_USERNAME")

	defer func() {
		if originalToken != "" {
			os.Setenv("BEEPER_TOKEN", originalToken)
		} else {
			os.Unsetenv("BEEPER_TOKEN")
		}
		if originalEmail != "" {
			os.Setenv("BEEPER_EMAIL", originalEmail)
		} else {
			os.Unsetenv("BEEPER_EMAIL")
		}
		if originalUsername != "" {
			os.Setenv("BEEPER_USERNAME", originalUsername)
		} else {
			os.Unsetenv("BEEPER_USERNAME")
		}
	}()

	auth := archive.NewBeeperAuth("test.com")
	auth.Email = "test@example.com"
	auth.Token = "test-token"
	auth.Whoami = &beeperapi.RespWhoami{
		UserInfo: beeperapi.WhoamiUserInfo{
			Username: "testuser",
			Email:    "test@example.com",
		},
	}

	auth.SaveCredentials()

	// Verify env vars are set
	assert.Equal(t, "test-token", os.Getenv("BEEPER_TOKEN"))
	assert.Equal(t, "test@example.com", os.Getenv("BEEPER_EMAIL"))
	assert.Equal(t, "testuser", os.Getenv("BEEPER_USERNAME"))
}

func TestBeeperAuth_GetMatrixClient_NotAuthenticated(t *testing.T) {
	auth := archive.NewBeeperAuth("test.com")

	client, err := auth.GetMatrixClient()
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestBeeperCredentials_JSON(t *testing.T) {
	creds := archive.BeeperCredentials{
		BaseDomain:   "test.com",
		Email:        "test@example.com",
		Token:        "test-token",
		Username:     "testuser",
		MatrixToken:  "matrix-token",
		MatrixUserID: "@test:test.com",
		Whoami: &beeperapi.RespWhoami{
			UserInfo: beeperapi.WhoamiUserInfo{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(creds)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "test@example.com")

	// Test unmarshaling
	var newCreds archive.BeeperCredentials
	err = json.Unmarshal(data, &newCreds)
	assert.NoError(t, err)
	assert.Equal(t, creds.Email, newCreds.Email)
	assert.Equal(t, creds.Token, newCreds.Token)
}
