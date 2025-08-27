package main

import (
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message represents a Matrix message stored in MongoDB
type Message struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	RoomID      string             `bson:"room_id" json:"room_id"`
	EventID     string             `bson:"event_id" json:"event_id"`
	Sender      string             `bson:"sender" json:"sender"`
	UserID      string             `bson:"user_id,omitempty" json:"user_id,omitempty"`
	MessageType string             `bson:"type" json:"type"`
	Timestamp   time.Time          `bson:"timestamp" json:"timestamp"`
	Content     bson.M             `bson:"content" json:"content"`
}

// IsImage returns true if the message is an image message
func (m *Message) IsImage() bool {
	msgtype, ok := m.Content["msgtype"].(string)
	return ok && msgtype == "m.image"
}

// ImageURL returns the image URL if this is an image message
func (m *Message) ImageURL() string {
	if !m.IsImage() {
		return ""
	}
	if url, ok := m.Content["url"].(string); ok {
		return url
	}
	return ""
}

// ThumbnailURL returns the thumbnail URL if this is an image message
func (m *Message) ThumbnailURL() string {
	if !m.IsImage() {
		return ""
	}
	if info, ok := m.Content["info"].(bson.M); ok {
		if thumbURL, ok := info["thumbnail_url"].(string); ok {
			return thumbURL
		}
	}
	return ""
}

// ValidateMessage validates a message according to Matrix patterns
func (m *Message) Validate() error {
	// Room ID pattern: !.+:.+
	roomIDPattern := regexp.MustCompile(`^!.+:.+$`)
	if !roomIDPattern.MatchString(m.RoomID) {
		return &ValidationError{Field: "room_id", Message: "Invalid room ID format"}
	}

	// Event ID pattern: $.+
	eventIDPattern := regexp.MustCompile(`^\$.+$`)
	if !eventIDPattern.MatchString(m.EventID) {
		return &ValidationError{Field: "event_id", Message: "Invalid event ID format"}
	}

	// Sender pattern: @.+:.+
	senderPattern := regexp.MustCompile(`^@.+:.+$`)
	if !senderPattern.MatchString(m.Sender) {
		return &ValidationError{Field: "sender", Message: "Invalid sender format"}
	}

	// UserID pattern (optional): @.+:.+
	if m.UserID != "" {
		if !senderPattern.MatchString(m.UserID) {
			return &ValidationError{Field: "user_id", Message: "Invalid user ID format"}
		}
	}

	// Message type should be m.room.message
	if m.MessageType != "m.room.message" {
		return &ValidationError{Field: "type", Message: "Invalid message type"}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// MessageFilter represents filters for querying messages
type MessageFilter struct {
	RoomID    string
	EventID   string
	Sender    string
	StartTime *time.Time
	EndTime   *time.Time
}

// ToBSON converts the filter to a BSON query
func (f *MessageFilter) ToBSON() bson.M {
	filter := bson.M{}

	if f.RoomID != "" {
		filter["room_id"] = f.RoomID
	}

	if f.EventID != "" {
		filter["event_id"] = f.EventID
	}

	if f.Sender != "" {
		filter["sender"] = f.Sender
	}

	if f.StartTime != nil || f.EndTime != nil {
		timeFilter := bson.M{}
		if f.StartTime != nil {
			timeFilter["$gte"] = *f.StartTime
		}
		if f.EndTime != nil {
			timeFilter["$lte"] = *f.EndTime
		}
		filter["timestamp"] = timeFilter
	}

	return filter
}
