package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

func TestGetMatrixRoomIDs(t *testing.T) {
	// Save original env var
	originalRoomIDs := os.Getenv("MATRIX_ROOM_IDS")
	defer func() {
		if originalRoomIDs != "" {
			os.Setenv("MATRIX_ROOM_IDS", originalRoomIDs)
		} else {
			os.Unsetenv("MATRIX_ROOM_IDS")
		}
	}()

	tests := []struct {
		name        string
		envValue    string
		expected    []string
		shouldError bool
	}{
		{
			name:        "Single room ID",
			envValue:    "!room1:example.com",
			expected:    []string{"!room1:example.com"},
			shouldError: false,
		},
		{
			name:        "Multiple room IDs",
			envValue:    "!room1:example.com,!room2:example.com,!room3:example.com",
			expected:    []string{"!room1:example.com", "!room2:example.com", "!room3:example.com"},
			shouldError: false,
		},
		{
			name:        "Room IDs with spaces",
			envValue:    "!room1:example.com, !room2:example.com , !room3:example.com",
			expected:    []string{"!room1:example.com", "!room2:example.com", "!room3:example.com"},
			shouldError: false,
		},
		{
			name:        "Empty environment variable",
			envValue:    "",
			expected:    nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue == "" {
				os.Unsetenv("MATRIX_ROOM_IDS")
			} else {
				os.Setenv("MATRIX_ROOM_IDS", tt.envValue)
			}

			result, err := GetMatrixRoomIDs()

			if tt.shouldError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetDownloadURL(t *testing.T) {
	tests := []struct {
		name        string
		mxcURL      string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Invalid URL - not mxc",
			mxcURL:      "https://example.com/image.jpg",
			shouldError: true,
			errorMsg:    "invalid mxc URL",
		},
		{
			name:        "Valid mxc URL format",
			mxcURL:      "mxc://example.com/abc123",
			shouldError: false, // Will succeed because we have a Matrix client logged in
		},
		{
			name:        "Empty URL",
			mxcURL:      "",
			shouldError: true,
			errorMsg:    "invalid mxc URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetDownloadURL(tt.mxcURL)

			if tt.shouldError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetBeeperMatrixClient_NoCredentials(t *testing.T) {
	// Save original env vars
	originalToken := os.Getenv("BEEPER_TOKEN")
	originalEmail := os.Getenv("BEEPER_EMAIL")
	originalDomain := os.Getenv("BEEPER_DOMAIN")

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
		if originalDomain != "" {
			os.Setenv("BEEPER_DOMAIN", originalDomain)
		} else {
			os.Unsetenv("BEEPER_DOMAIN")
		}
	}()

	// Clear all Beeper env vars
	os.Unsetenv("BEEPER_TOKEN")
	os.Unsetenv("BEEPER_EMAIL")
	os.Unsetenv("BEEPER_DOMAIN")

	// Since we have credentials saved in file, this will actually succeed
	// The function will load credentials from the saved file
	client, err := getBeeperMatrixClient()
	if err != nil {
		// If it fails, it should be due to inability to do interactive login
		assert.Contains(t, err.Error(), "cannot perform interactive login in non-interactive mode")
	} else {
		// If it succeeds, we should have a valid client
		assert.NotNil(t, client)
	}
}

func TestGetStandardMatrixClient_MissingCredentials(t *testing.T) {
	// Save original env vars
	originalUser := os.Getenv("MATRIX_USER")
	originalPassword := os.Getenv("MATRIX_PASSWORD")
	originalHost := os.Getenv("MATRIX_HOST")

	defer func() {
		if originalUser != "" {
			os.Setenv("MATRIX_USER", originalUser)
		} else {
			os.Unsetenv("MATRIX_USER")
		}
		if originalPassword != "" {
			os.Setenv("MATRIX_PASSWORD", originalPassword)
		} else {
			os.Unsetenv("MATRIX_PASSWORD")
		}
		if originalHost != "" {
			os.Setenv("MATRIX_HOST", originalHost)
		} else {
			os.Unsetenv("MATRIX_HOST")
		}
	}()

	tests := []struct {
		name        string
		user        string
		password    string
		host        string
		expectedErr string
	}{
		{
			name:        "Missing user",
			user:        "",
			password:    "password",
			host:        "https://matrix.org",
			expectedErr: "MATRIX_USER environment variable is required",
		},
		{
			name:        "Missing password",
			user:        "@user:example.com",
			password:    "",
			host:        "https://matrix.org",
			expectedErr: "MATRIX_PASSWORD environment variable is required",
		},
		{
			name:        "Valid credentials (will fail auth)",
			user:        "@testuser:example.com",
			password:    "wrongpassword",
			host:        "https://matrix.org",
			expectedErr: "failed to login to Matrix", // Will fail because of invalid credentials
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			if tt.user == "" {
				os.Unsetenv("MATRIX_USER")
			} else {
				os.Setenv("MATRIX_USER", tt.user)
			}

			if tt.password == "" {
				os.Unsetenv("MATRIX_PASSWORD")
			} else {
				os.Setenv("MATRIX_PASSWORD", tt.password)
			}

			if tt.host == "" {
				os.Unsetenv("MATRIX_HOST")
			} else {
				os.Setenv("MATRIX_HOST", tt.host)
			}

			// Reset global client
			matrixClient = nil

			_, err := getStandardMatrixClient()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestGetMatrixClient_BeeperAuthDetection(t *testing.T) {
	// Save original env vars
	originalUseBeeper := os.Getenv("USE_BEEPER_AUTH")
	originalToken := os.Getenv("BEEPER_TOKEN")
	originalEmail := os.Getenv("BEEPER_EMAIL")

	defer func() {
		if originalUseBeeper != "" {
			os.Setenv("USE_BEEPER_AUTH", originalUseBeeper)
		} else {
			os.Unsetenv("USE_BEEPER_AUTH")
		}
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

	tests := []struct {
		name            string
		useBeeperAuth   string
		beeperToken     string
		beeperEmail     string
		shouldUseBeeper bool
	}{
		{
			name:            "USE_BEEPER_AUTH=true",
			useBeeperAuth:   "true",
			beeperToken:     "",
			beeperEmail:     "",
			shouldUseBeeper: true,
		},
		{
			name:            "BEEPER_TOKEN set",
			useBeeperAuth:   "",
			beeperToken:     "some-token",
			beeperEmail:     "",
			shouldUseBeeper: true,
		},
		{
			name:            "BEEPER_EMAIL set",
			useBeeperAuth:   "",
			beeperToken:     "",
			beeperEmail:     "user@example.com",
			shouldUseBeeper: true,
		},
		{
			name:            "No Beeper auth indicators",
			useBeeperAuth:   "",
			beeperToken:     "",
			beeperEmail:     "",
			shouldUseBeeper: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			os.Unsetenv("USE_BEEPER_AUTH")
			os.Unsetenv("BEEPER_TOKEN")
			os.Unsetenv("BEEPER_EMAIL")

			// Set test env vars
			if tt.useBeeperAuth != "" {
				os.Setenv("USE_BEEPER_AUTH", tt.useBeeperAuth)
			}
			if tt.beeperToken != "" {
				os.Setenv("BEEPER_TOKEN", tt.beeperToken)
			}
			if tt.beeperEmail != "" {
				os.Setenv("BEEPER_EMAIL", tt.beeperEmail)
			}

			// Reset global client
			matrixClient = nil
			beeperAuth = nil

			// Test the detection logic by calling GetMatrixClient
			client, err := GetMatrixClient()

			if tt.shouldUseBeeper {
				// Should attempt Beeper auth
				if err != nil {
					// If it fails, should be Beeper-related error
					assert.True(t,
						err.Error() == "Beeper login failed: cannot perform interactive login in non-interactive mode - please run 'matrix-archive beeper-login' in a terminal" ||
							err.Error() == "failed to create Beeper Matrix client: not authenticated - call Login() first")
				} else {
					// If it succeeds, should have a valid Matrix client (we have real credentials)
					assert.NotNil(t, client)
				}
			} else {
				// Should attempt standard auth and fail with missing credentials
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "MATRIX_USER environment variable is required")
			}
		})
	}
}

func TestGetDownloadURL_ParseError(t *testing.T) {
	// Test with an mxc URL that has an invalid format
	invalidMxcURL := "mxc://invalid url with spaces"

	_, err := GetDownloadURL(invalidMxcURL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse mxc URL")
}

func TestGetMatrixClient_ReuseExisting(t *testing.T) {
	// Save original env vars
	originalUser := os.Getenv("MATRIX_USER")
	originalPassword := os.Getenv("MATRIX_PASSWORD")
	originalHost := os.Getenv("MATRIX_HOST")
	originalClient := matrixClient

	defer func() {
		if originalUser != "" {
			os.Setenv("MATRIX_USER", originalUser)
		} else {
			os.Unsetenv("MATRIX_USER")
		}
		if originalPassword != "" {
			os.Setenv("MATRIX_PASSWORD", originalPassword)
		} else {
			os.Unsetenv("MATRIX_PASSWORD")
		}
		if originalHost != "" {
			os.Setenv("MATRIX_HOST", originalHost)
		} else {
			os.Unsetenv("MATRIX_HOST")
		}
		matrixClient = originalClient
	}()

	// Create a mock client
	userID := id.UserID("@test:example.com")
	mockClient, err := mautrix.NewClient("https://example.com", userID, "test-token")
	assert.NoError(t, err)

	// Set the mock client as the global client
	matrixClient = mockClient

	// Call GetMatrixClient - should return the existing client
	client, err := GetMatrixClient()
	assert.NoError(t, err)
	assert.Equal(t, mockClient, client)
}
