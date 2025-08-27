package tests

import (
	archive "github.com/osteele/matrix-archive/lib"

	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"maunium.net/go/mautrix/event"
)

func TestIsMessageEvent(t *testing.T) {
	// Test message event
	assert.True(t, archive.IsMessageEvent(event.EventMessage))

	// Test reaction event
	assert.True(t, archive.IsMessageEvent(event.EventReaction))

	// Test non-message event
	assert.False(t, archive.IsMessageEvent(event.StateRoomName))
}

func TestImportMessages_MissingRoomIDs(t *testing.T) {
	// Save original env var
	originalRoomIDs := os.Getenv("MATRIX_ROOM_IDS")
	defer func() {
		if originalRoomIDs != "" {
			os.Setenv("MATRIX_ROOM_IDS", originalRoomIDs)
		} else {
			os.Unsetenv("MATRIX_ROOM_IDS")
		}
	}()

	// Clear room IDs env var
	os.Unsetenv("MATRIX_ROOM_IDS")

	err := archive.ImportMessages(10)
	assert.Error(t, err)
	// Should fail early due to missing environment variables
	assert.Contains(t, err.Error(), "Matrix client")
}

func TestImportMessages_InvalidLimit(t *testing.T) {
	// Set a room ID to get past the room ID check
	os.Setenv("MATRIX_ROOM_IDS", "!test:example.com")
	defer os.Unsetenv("MATRIX_ROOM_IDS")

	// Test with negative limit (should work - negative means no limit)
	err := archive.ImportMessages(-1)
	// This may fail due to auth issues, but it should get past the limit validation
	if err != nil {
		// The error should not be about the limit itself
		assert.NotContains(t, err.Error(), "invalid limit")
	}

	// Test with zero limit (should work - zero means no limit)
	err = archive.ImportMessages(0)
	// This may fail due to auth issues, but it should get past the limit validation
	if err != nil {
		// The error should not be about the limit itself
		assert.NotContains(t, err.Error(), "invalid limit")
	}
}

func TestImportMessages_DatabaseInitError(t *testing.T) {
	// Set a room ID to get past the room ID check
	os.Setenv("MATRIX_ROOM_IDS", "!test:example.com")
	defer os.Unsetenv("MATRIX_ROOM_IDS")

	err := archive.ImportMessages(10)
	assert.Error(t, err)
	// Should fail early due to missing environment variables
	assert.Contains(t, err.Error(), "Matrix client")
}
