package tests

import (
	archive "github.com/osteele/matrix-archive/lib"

	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
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

func TestReplaceDots(t *testing.T) {
	// Test simple map with dots
	input := map[string]interface{}{
		"key.with.dots": "value",
		"normal_key":    "value2",
	}
	result := archive.ReplaceDots(input)
	resultMap, ok := result.(bson.M)
	assert.True(t, ok, "ReplaceDots should return bson.M for map input")
	assert.Equal(t, "value", resultMap["key•with•dots"])
	assert.Equal(t, "value2", resultMap["normal_key"])

	// Test nil input
	result = archive.ReplaceDots(nil)
	assert.Nil(t, result)

	// Test non-map input (should return as-is)
	result = archive.ReplaceDots("simple string")
	assert.Equal(t, "simple string", result)
	
	// Test nested map with dots
	nestedInput := map[string]interface{}{
		"outer.key": map[string]interface{}{
			"inner.key": "nested_value",
		},
	}
	result = archive.ReplaceDots(nestedInput)
	resultMap, ok = result.(bson.M)
	assert.True(t, ok)
	innerMap, ok := resultMap["outer•key"].(bson.M)
	assert.True(t, ok)
	assert.Equal(t, "nested_value", innerMap["inner•key"])
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
	assert.Contains(t, err.Error(), "failed to get room IDs")
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

	// Save original MongoDB URI
	originalURI := os.Getenv("MONGODB_URI")
	defer func() {
		if originalURI != "" {
			os.Setenv("MONGODB_URI", originalURI)
		} else {
			os.Unsetenv("MONGODB_URI")
		}
	}()

	// Set invalid MongoDB URI
	os.Setenv("MONGODB_URI", "mongodb://invalid:99999")

	err := archive.ImportMessages(10)
	assert.Error(t, err)
	// Should fail at database initialization step
	assert.Contains(t, err.Error(), "failed to initialize database")
}
