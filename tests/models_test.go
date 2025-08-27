package tests

import (
	archive "github.com/osteele/matrix-archive/lib"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_IsImage(t *testing.T) {
	// Test image message
	imageMsg := archive.Message{
		Content: map[string]interface{}{
			"msgtype": "m.image",
			"body":    "image.jpg",
		},
	}
	assert.True(t, imageMsg.IsImage())

	// Test text message
	textMsg := archive.Message{
		Content: map[string]interface{}{
			"msgtype": "m.text",
			"body":    "Hello world",
		},
	}
	assert.False(t, textMsg.IsImage())
}

func TestMessage_ImageURL(t *testing.T) {
	// Test image message with URL
	imageMsg := archive.Message{
		Content: map[string]interface{}{
			"msgtype": "m.image",
			"url":     "mxc://example.com/abc123",
		},
	}
	assert.Equal(t, "mxc://example.com/abc123", imageMsg.ImageURL())

	// Test non-image message
	textMsg := archive.Message{
		Content: map[string]interface{}{
			"msgtype": "m.text",
			"body":    "Hello world",
		},
	}
	assert.Equal(t, "", textMsg.ImageURL())
}

func TestMessage_Validate(t *testing.T) {
	// Test valid message
	validMsg := archive.Message{
		RoomID:      "!room123:example.com",
		EventID:     "$event123:example.com",
		Sender:      "@user:example.com",
		MessageType: "m.room.message",
	}
	assert.NoError(t, validMsg.Validate())

	// Test invalid room ID
	invalidRoomID := archive.Message{
		RoomID:      "invalid",
		EventID:     "$event123:example.com",
		Sender:      "@user:example.com",
		MessageType: "m.room.message",
	}
	assert.Error(t, invalidRoomID.Validate())
}

func TestMessageFilter_ToSQL(t *testing.T) {
	// Test empty filter
	emptyFilter := archive.MessageFilter{}
	sql, args := emptyFilter.ToSQL()
	assert.Empty(t, sql)
	assert.Empty(t, args)

	// Test filter with room ID
	roomFilter := archive.MessageFilter{
		RoomID: "!room123:example.com",
	}
	sql, args = roomFilter.ToSQL()
	assert.Equal(t, "room_id = ?", sql)
	assert.Equal(t, []interface{}{"!room123:example.com"}, args)

	// Test filter with sender
	senderFilter := archive.MessageFilter{
		Sender: "@user:example.com",
	}
	sql, args = senderFilter.ToSQL()
	assert.Equal(t, "sender = ?", sql)
	assert.Equal(t, []interface{}{"@user:example.com"}, args)

	// Test filter with event ID
	eventFilter := archive.MessageFilter{
		EventID: "$event123:example.com",
	}
	sql, args = eventFilter.ToSQL()
	assert.Equal(t, "event_id = ?", sql)
	assert.Equal(t, []interface{}{"$event123:example.com"}, args)

	// Test complete filter
	completeFilter := archive.MessageFilter{
		RoomID:  "!room123:example.com",
		Sender:  "@user:example.com",
		EventID: "$event123:example.com",
	}
	sql, args = completeFilter.ToSQL()
	assert.Equal(t, "room_id = ? AND event_id = ? AND sender = ?", sql)
	assert.Equal(t, []interface{}{"!room123:example.com", "$event123:example.com", "@user:example.com"}, args)
}

func TestMessage_ThumbnailURL(t *testing.T) {
	// Test image message with thumbnail URL
	imageMsg := archive.Message{
		Content: map[string]interface{}{
			"msgtype": "m.image",
			"info": map[string]interface{}{
				"thumbnail_url": "mxc://example.com/thumb123",
			},
		},
	}
	assert.Equal(t, "mxc://example.com/thumb123", imageMsg.ThumbnailURL())

	// Test image message without thumbnail
	imageMsgNoThumb := archive.Message{
		Content: map[string]interface{}{
			"msgtype": "m.image",
			"url":     "mxc://example.com/abc123",
		},
	}
	assert.Equal(t, "", imageMsgNoThumb.ThumbnailURL())

	// Test non-image message
	textMsg := archive.Message{
		Content: map[string]interface{}{
			"msgtype": "m.text",
			"body":    "Hello world",
		},
	}
	assert.Equal(t, "", textMsg.ThumbnailURL())

	// Test image message with info but no thumbnail URL
	imageMsgNoThumbURL := archive.Message{
		Content: map[string]interface{}{
			"msgtype": "m.image",
			"info": map[string]interface{}{
				"mimetype": "image/jpeg",
			},
		},
	}
	assert.Equal(t, "", imageMsgNoThumbURL.ThumbnailURL())
}

func TestValidationError(t *testing.T) {
	err := &archive.ValidationError{
		Field:   "room_id",
		Message: "Invalid format",
	}
	assert.Equal(t, "room_id: Invalid format", err.Error())
}
