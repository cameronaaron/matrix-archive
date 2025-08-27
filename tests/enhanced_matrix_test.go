package tests

import (
	"context"
	"testing"
	"time"

	archive "github.com/osteele/matrix-archive/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// TestEnhancedMatrixClient tests the enhanced Matrix client functionality
func TestEnhancedMatrixClient(t *testing.T) {
	// Test creating enhanced client
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	err := archive.InitDatabase(config)
	require.NoError(t, err)
	defer archive.CloseDatabase()

	// Mock homeserver URL, user ID, and access token
	homeserverURL := "https://matrix.example.com"
	userID := id.UserID("@testuser:example.com")
	accessToken := "test_token"

	enhanced, err := archive.NewEnhancedMatrixClient(homeserverURL, userID, accessToken, archive.GetDatabase())
	assert.NoError(t, err)
	assert.NotNil(t, enhanced)

	// Test client configuration
	assert.Equal(t, 3, enhanced.DefaultHTTPRetries)
	assert.Equal(t, 2*time.Second, enhanced.DefaultHTTPBackoff)
	assert.False(t, enhanced.IgnoreRateLimit)
	assert.NotNil(t, enhanced.StateStore)
}

// TestEventConversion tests the enhanced event conversion functionality
func TestEventConversion(t *testing.T) {
	// Create test event
	evt := &event.Event{
		ID:        id.EventID("$test:example.com"),
		Type:      event.EventMessage,
		Sender:    id.UserID("@sender:example.com"),
		Timestamp: time.Now().UnixMilli(),
		Content: event.Content{
			Raw: map[string]interface{}{
				"msgtype": "m.text",
				"body":    "Test message",
			},
		},
	}

	// Initialize database
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	err := archive.InitDatabase(config)
	require.NoError(t, err)
	defer archive.CloseDatabase()

	// Create enhanced client
	enhanced, err := archive.NewEnhancedMatrixClient("https://matrix.example.com", "@test:example.com", "token", archive.GetDatabase())
	require.NoError(t, err)

	// Test event conversion
	message, err := enhanced.ConvertEventToMessageEnhanced(evt, "!testroom:example.com")
	assert.NoError(t, err)
	assert.NotNil(t, message)

	assert.Equal(t, "!testroom:example.com", message.RoomID)
	assert.Equal(t, "$test:example.com", message.EventID)
	assert.Equal(t, "@sender:example.com", message.Sender)
	assert.Equal(t, "m.room.message", message.MessageType)
	assert.Equal(t, "m.text", message.Content["msgtype"])
	assert.Equal(t, "Test message", message.Content["body"])
}

// TestMessageEventTypeChecking tests the enhanced message event type checking
func TestEnhancedMessageEventTypeChecking(t *testing.T) {
	// Initialize database for creating enhanced client
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	err := archive.InitDatabase(config)
	require.NoError(t, err)
	defer archive.CloseDatabase()

	enhanced, err := archive.NewEnhancedMatrixClient("https://matrix.example.com", "@test:example.com", "token", archive.GetDatabase())
	require.NoError(t, err)

	// Test supported event types
	assert.True(t, enhanced.IsMessageEvent(event.EventMessage))
	assert.True(t, enhanced.IsMessageEvent(event.EventReaction))
	assert.True(t, enhanced.IsMessageEvent(event.EventEncrypted))

	// Test unsupported event types
	assert.False(t, enhanced.IsMessageEvent(event.StateRoomName))
	assert.False(t, enhanced.IsMessageEvent(event.StateMember))
	assert.False(t, enhanced.IsMessageEvent(event.StateRoomAvatar))
}

// TestMediaDownloadURL tests the media download URL parsing
func TestMediaDownloadURL(t *testing.T) {
	// Initialize database for creating enhanced client
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	err := archive.InitDatabase(config)
	require.NoError(t, err)
	defer archive.CloseDatabase()

	enhanced, err := archive.NewEnhancedMatrixClient("https://matrix.example.com", "@test:example.com", "token", archive.GetDatabase())
	require.NoError(t, err)

	// Test valid mxc URL
	ctx := context.Background()
	mxcURL := "mxc://example.com/test"

	// This will fail because we're not connected to a real server, but we can test URL parsing
	_, err = enhanced.DownloadMedia(ctx, mxcURL)
	// We expect an error since this is a test environment, but the URL should be parsed correctly
	assert.Error(t, err)

	// Test invalid URL
	_, err = enhanced.DownloadMedia(ctx, "invalid://url")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mxc URL")
}
