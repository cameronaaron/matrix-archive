package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

func TestMessage_IsImage(t *testing.T) {
	// Test image message
	imageMsg := Message{
		Content: map[string]interface{}{
			"msgtype": "m.image",
			"body":    "image.jpg",
		},
	}
	assert.True(t, imageMsg.IsImage())

	// Test text message
	textMsg := Message{
		Content: map[string]interface{}{
			"msgtype": "m.text",
			"body":    "Hello world",
		},
	}
	assert.False(t, textMsg.IsImage())
}

func TestMessage_ImageURL(t *testing.T) {
	// Test image message with URL
	imageMsg := Message{
		Content: map[string]interface{}{
			"msgtype": "m.image",
			"url":     "mxc://example.com/abc123",
		},
	}
	assert.Equal(t, "mxc://example.com/abc123", imageMsg.ImageURL())

	// Test non-image message
	textMsg := Message{
		Content: map[string]interface{}{
			"msgtype": "m.text",
			"body":    "Hello world",
		},
	}
	assert.Equal(t, "", textMsg.ImageURL())
}

func TestMessage_Validate(t *testing.T) {
	// Test valid message
	validMsg := Message{
		RoomID:      "!room123:example.com",
		EventID:     "$event123:example.com",
		Sender:      "@user:example.com",
		MessageType: "m.room.message",
	}
	assert.NoError(t, validMsg.Validate())

	// Test invalid room ID
	invalidRoomID := Message{
		RoomID:      "invalid",
		EventID:     "$event123:example.com",
		Sender:      "@user:example.com",
		MessageType: "m.room.message",
	}
	assert.Error(t, invalidRoomID.Validate())
}

func TestMessageFilter_ToBSON(t *testing.T) {
	// Test empty filter
	emptyFilter := MessageFilter{}
	bson := emptyFilter.ToBSON()
	assert.Empty(t, bson)

	// Test filter with room ID
	roomFilter := MessageFilter{
		RoomID: "!room123:example.com",
	}
	bson = roomFilter.ToBSON()
	assert.Equal(t, "!room123:example.com", bson["room_id"])

	// Test filter with sender
	senderFilter := MessageFilter{
		Sender: "@user:example.com",
	}
	bson = senderFilter.ToBSON()
	assert.Equal(t, "@user:example.com", bson["sender"])

	// Test filter with event ID
	eventFilter := MessageFilter{
		EventID: "$event123:example.com",
	}
	bson = eventFilter.ToBSON()
	assert.Equal(t, "$event123:example.com", bson["event_id"])

	// Test complete filter
	completeFilter := MessageFilter{
		RoomID:  "!room123:example.com",
		Sender:  "@user:example.com",
		EventID: "$event123:example.com",
	}
	bson = completeFilter.ToBSON()
	assert.Equal(t, "!room123:example.com", bson["room_id"])
	assert.Equal(t, "@user:example.com", bson["sender"])
	assert.Equal(t, "$event123:example.com", bson["event_id"])
}

func TestMessage_ThumbnailURL(t *testing.T) {
	// Test image message with thumbnail URL
	imageMsg := Message{
		Content: bson.M{
			"msgtype": "m.image",
			"info": bson.M{
				"thumbnail_url": "mxc://example.com/thumb123",
			},
		},
	}
	assert.Equal(t, "mxc://example.com/thumb123", imageMsg.ThumbnailURL())

	// Test image message without thumbnail
	imageMsgNoThumb := Message{
		Content: bson.M{
			"msgtype": "m.image",
			"url":     "mxc://example.com/abc123",
		},
	}
	assert.Equal(t, "", imageMsgNoThumb.ThumbnailURL())

	// Test non-image message
	textMsg := Message{
		Content: bson.M{
			"msgtype": "m.text",
			"body":    "Hello world",
		},
	}
	assert.Equal(t, "", textMsg.ThumbnailURL())

	// Test image message with info but no thumbnail URL
	imageMsgNoThumbURL := Message{
		Content: bson.M{
			"msgtype": "m.image",
			"info": bson.M{
				"mimetype": "image/jpeg",
			},
		},
	}
	assert.Equal(t, "", imageMsgNoThumbURL.ThumbnailURL())
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "room_id",
		Message: "Invalid format",
	}
	assert.Equal(t, "room_id: Invalid format", err.Error())
}
