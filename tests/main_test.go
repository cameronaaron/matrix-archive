package tests

import (
	archive "github.com/osteele/matrix-archive/lib"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidationAndUtilityFunctions(t *testing.T) {
	// Test message validation
	msg := &archive.Message{
		RoomID:      "!room:example.com",
		EventID:     "$event123",
		Sender:      "@user:example.com",
		MessageType: "m.room.message",
	}
	err := msg.Validate()
	assert.NoError(t, err)

	// Test export format validation
	assert.True(t, archive.IsValidFormat("json"))
	assert.True(t, archive.IsValidFormat("html"))
	assert.True(t, archive.IsValidFormat("yaml"))
	assert.True(t, archive.IsValidFormat("txt"))
	assert.False(t, archive.IsValidFormat("invalid"))
	assert.False(t, archive.IsValidFormat(""))
}

func TestBeeperAuthCreation(t *testing.T) {
	// Test that we can create BeeperAuth without errors
	auth := archive.NewBeeperAuth("test.com")
	assert.NotNil(t, auth)

	// Test default domain
	authDefault := archive.NewBeeperAuth("")
	assert.NotNil(t, authDefault)
}

func TestPublicAPIAccessibility(t *testing.T) {
	// Test that all main public functions are accessible
	_ = archive.DownloadImages
	_ = archive.GetDownloadStem
	_ = archive.GetExistingFilesMap
	_ = archive.ImportMessages
	_ = archive.IsMessageEvent
	_ = archive.ListRooms
	_ = archive.GetRoomDisplayName
	_ = archive.IsValidFormat
	_ = archive.NewBeeperAuth
}
