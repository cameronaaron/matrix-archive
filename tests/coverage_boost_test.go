package tests

import (
	"os"
	"testing"

	archive "github.com/osteele/matrix-archive/lib"
	"github.com/stretchr/testify/assert"
)

// setupFullTestEnvironment sets up comprehensive test environment
func setupFullTestEnvironment(t *testing.T) {
	os.Setenv("USE_BEEPER_AUTH", "true")
	os.Setenv("BEEPER_TOKEN", testBeeperToken)
	os.Setenv("BEEPER_EMAIL", testEmail)
	os.Setenv("BEEPER_USERNAME", "testuser")
	os.Setenv("MATRIX_USER", testUserID)
	os.Setenv("MATRIX_HOST", testMatrixHost)
	os.Setenv("MATRIX_ROOM_IDS", testRoomID)
	os.Setenv("BEEPER_DOMAIN", "example.com")
	os.Setenv("MATRIX_PASSWORD", "fake-password")
}

// cleanupFullTestEnvironment cleans up test environment
func cleanupFullTestEnvironment(t *testing.T) {
	envVars := []string{
		"USE_BEEPER_AUTH", "BEEPER_TOKEN", "BEEPER_EMAIL", "BEEPER_USERNAME",
		"MATRIX_USER", "MATRIX_HOST", "MATRIX_ROOM_IDS", "BEEPER_DOMAIN", "MATRIX_PASSWORD",
	}
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

// TestCoverageBoost aims to increase coverage by testing uncovered functions
func TestCoverageBoostBeeperAuth(t *testing.T) {
	setupFullTestEnvironment(t)
	defer cleanupFullTestEnvironment(t)

	t.Run("PerformBeeperLogin_NonInteractive", func(t *testing.T) {
		// Test PerformBeeperLogin in non-interactive mode
		err := archive.PerformBeeperLogin("example.com", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot perform interactive login in non-interactive mode")
	})

	t.Run("PerformBeeperLogout", func(t *testing.T) {
		// Test PerformBeeperLogout
		err := archive.PerformBeeperLogout("example.com")
		assert.NoError(t, err)
	})

	t.Run("GetMatrixClient_StandardAuth_MissingPassword", func(t *testing.T) {
		// Clear Beeper auth and test standard auth path
		os.Unsetenv("USE_BEEPER_AUTH")
		os.Unsetenv("BEEPER_TOKEN")
		os.Unsetenv("BEEPER_EMAIL")
		os.Unsetenv("MATRIX_PASSWORD")

		defer func() {
			setupFullTestEnvironment(t)
		}()

		_, err := archive.GetMatrixClient()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MATRIX_PASSWORD environment variable is required")
	})

	t.Run("GetMatrixClient_StandardAuth_FullPath", func(t *testing.T) {
		// Clear Beeper auth and test standard auth path with all variables
		os.Unsetenv("USE_BEEPER_AUTH")
		os.Unsetenv("BEEPER_TOKEN")
		os.Unsetenv("BEEPER_EMAIL")
		os.Setenv("MATRIX_PASSWORD", "fake-password")
		os.Setenv("MATRIX_HOST", "https://fake.matrix.server")

		defer func() {
			setupFullTestEnvironment(t)
		}()

		_, err := archive.GetMatrixClient()
		assert.Error(t, err)
		// Will fail due to network issues but tests the code path
		assert.Contains(t, err.Error(), "failed to login to Matrix")
	})

	t.Run("BeeperAuth_Login_NonInteractive", func(t *testing.T) {
		auth := archive.NewBeeperAuth("beeper.com")

		err := auth.Login()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot perform interactive login in non-interactive mode")
	})
}

// TestCoverageBoostDatabase tests database functions to increase coverage
