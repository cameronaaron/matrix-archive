package archive

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// Message represents a Matrix message stored in the database
type Message struct {
	ID          int64                  `json:"id,omitempty"`
	RoomID      string                 `json:"room_id"`
	EventID     string                 `json:"event_id"`
	Sender      string                 `json:"sender"`
	UserID      string                 `json:"user_id,omitempty"`
	MessageType string                 `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	Content     map[string]interface{} `json:"content"`
}

// ContentJSON returns the content as a JSON string for database storage
func (m *Message) ContentJSON() (string, error) {
	if m.Content == nil {
		return "{}", nil
	}
	data, err := json.Marshal(m.Content)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SetContentFromJSON sets the content from a JSON string
func (m *Message) SetContentFromJSON(jsonStr string) error {
	if jsonStr == "" || jsonStr == "{}" {
		m.Content = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), &m.Content)
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
	if info, ok := m.Content["info"].(map[string]interface{}); ok {
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

// ToSQL converts the filter to SQL WHERE conditions and arguments
func (f *MessageFilter) ToSQL() (string, []interface{}) {
	if f == nil {
		return "", nil
	}

	var conditions []string
	var args []interface{}

	if f.RoomID != "" {
		conditions = append(conditions, "room_id = ?")
		args = append(args, f.RoomID)
	}

	if f.EventID != "" {
		conditions = append(conditions, "event_id = ?")
		args = append(args, f.EventID)
	}

	if f.Sender != "" {
		conditions = append(conditions, "sender = ?")
		args = append(args, f.Sender)
	}

	if f.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *f.StartTime)
	}

	if f.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *f.EndTime)
	}

	if len(conditions) == 0 {
		return "", args
	}

	return strings.Join(conditions, " AND "), args
}


