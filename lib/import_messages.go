package archive

import (
	"context"
	"fmt"
	"log"
	"time"

	"maunium.net/go/mautrix/event"
)

var messageEventTypes = []event.Type{
	event.EventMessage,
	event.EventReaction, // m.room.message.feedback equivalent
}

// RateLimiter provides simple rate limiting for API calls
type RateLimiter struct {
	rate     time.Duration
	lastCall time.Time
}

// NewRateLimiter creates a new rate limiter with the specified rate (requests per second)
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	if requestsPerSecond <= 0 {
		// Handle edge cases: use a very slow rate (1 request per hour) for zero/negative values
		requestsPerSecond = 1
		return &RateLimiter{
			rate: time.Hour,
		}
	}
	return &RateLimiter{
		rate: time.Second / time.Duration(requestsPerSecond),
	}
}

// Wait blocks until it's safe to make the next request
func (rl *RateLimiter) Wait() {
	now := time.Now()
	if elapsed := now.Sub(rl.lastCall); elapsed < rl.rate {
		time.Sleep(rl.rate - elapsed)
	}
	rl.lastCall = time.Now()
}

// ImportMessages imports messages from Matrix rooms into the database
func ImportMessages(limit int) error {
	// Initialize database connection with DuckDB
	if err := InitDuckDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer CloseDatabase()

	// Get Matrix client
	client, err := GetMatrixClient()
	if err != nil {
		return fmt.Errorf("failed to get Matrix client: %w", err)
	}

	// Use enhanced client for better mautrix-go integration
	enhanced, err := NewEnhancedMatrixClient(
		client.HomeserverURL.String(),
		client.UserID,
		client.AccessToken,
		GetDatabase(),
	)
	if err != nil {
		return fmt.Errorf("failed to create enhanced client: %w", err)
	}

	// Get configured room IDs
	roomIDs, err := GetMatrixRoomIDs()
	if err != nil {
		return fmt.Errorf("failed to get room IDs: %w", err)
	}

	totalImported := 0

	// Import from each room using enhanced client
	for i, roomID := range roomIDs {
		fmt.Printf("\n[%d/%d] Processing room: %s\n", i+1, len(roomIDs), roomID)

		count, err := enhanced.importEventsFromRoom(roomID, limit)
		if err != nil {
			log.Printf("Error importing from room %s: %v", roomID, err)
			continue
		}
		totalImported += count
		fmt.Printf("✓ Imported %d messages from room %s\n", count, roomID)

		// Show progress
		if len(roomIDs) > 1 {
			fmt.Printf("Progress: %d/%d rooms completed\n", i+1, len(roomIDs))
		}
	}

	// Get total message count
	totalCount, err := GetDatabase().GetMessageCount(context.Background(), nil)
	if err != nil {
		log.Printf("Failed to count total messages: %v", err)
	} else {
		fmt.Printf("The database now has %d total messages\n", totalCount)
	}

	return nil
}

// IsMessageEvent checks if an event type is a message event
func IsMessageEvent(eventType event.Type) bool {
	for _, msgType := range messageEventTypes {
		if eventType == msgType {
			return true
		}
	}
	return false
}

// convertEventToMessage converts a Matrix event to our Message struct
func convertEventToMessage(evt *event.Event, roomID string) (*Message, error) {
	// Use the raw content directly - DuckDB JSON storage allows flexible content structure
	var content map[string]interface{}
	if evt.Content.Raw != nil {
		content = evt.Content.Raw
	} else {
		content = make(map[string]interface{})
	}

	message := &Message{
		RoomID:      roomID,
		EventID:     evt.ID.String(),
		Sender:      evt.Sender.String(),
		MessageType: "m.room.message",
		Timestamp:   time.Unix(evt.Timestamp/1000, (evt.Timestamp%1000)*1000000),
		Content:     content,
	}

	return message, nil
}


