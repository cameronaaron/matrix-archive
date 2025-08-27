package tests

import (
	archive "github.com/osteele/matrix-archive/lib"

	"testing"

	"github.com/stretchr/testify/assert"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

func TestListRooms_InvalidPattern(t *testing.T) {
	// Test with invalid regex pattern
	err := archive.ListRooms("[invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex pattern")
}

func TestListRooms_ValidPattern(t *testing.T) {
	// Test with a valid pattern that likely won't match many rooms
	// This will test the pattern filtering logic
	err := archive.ListRooms("NonExistentRoomPattern123456789")
	assert.NoError(t, err) // Should succeed even if no rooms match
}

func TestGetRoomDisplayName(t *testing.T) {
	// Create a mock client
	userID := id.UserID("@test:example.com")
	mockClient, err := mautrix.NewClient("https://example.com", userID, "test-token")
	assert.NoError(t, err)

	// Test with non-existent room (will timeout and return room ID)
	displayName, err := archive.GetRoomDisplayName(mockClient, "!nonexistent:example.com")
	// Should return the room ID itself when can't get the name
	assert.NoError(t, err)
	assert.Equal(t, "!nonexistent:example.com", displayName)
}

func TestGetRoomDisplayName_WithContext(t *testing.T) {
	// Create a mock client
	userID := id.UserID("@test:example.com")
	mockClient, err := mautrix.NewClient("https://example.com", userID, "test-token")
	assert.NoError(t, err)

	// Test that the function uses context with timeout
	// This will test the context cancellation path
	displayName, err := archive.GetRoomDisplayName(mockClient, "!timeout:example.com")
	// Should return the room ID when can't get name
	assert.NoError(t, err)
	assert.Equal(t, "!timeout:example.com", displayName)
}
