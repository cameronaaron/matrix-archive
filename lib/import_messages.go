package archive

import (
	"context"
	"fmt"
	"log"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

var messageEventTypes = []event.Type{
	event.EventMessage,
	event.EventReaction, // m.room.message.feedback equivalent
}

// Rate limiting configuration
const (
	DefaultRateLimit = 10          // requests per second
	RateLimitWindow  = time.Second // rate limit window
	MaxBurstSize     = 20          // maximum burst size
)

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

// importMessages imports messages from Matrix rooms into the database
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
		// Fallback to original implementation if enhanced fails
		log.Printf("Warning: Could not create enhanced client, using fallback: %v", err)
		return importMessagesLegacy(limit)
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

// importMessagesLegacy provides fallback using the original implementation with DuckDB
func importMessagesLegacy(limit int) error {
	// Get Matrix client
	client, err := GetMatrixClient()
	if err != nil {
		return fmt.Errorf("failed to get Matrix client: %w", err)
	}

	// Get configured room IDs
	roomIDs, err := GetMatrixRoomIDs()
	if err != nil {
		return fmt.Errorf("failed to get room IDs: %w", err)
	}

	// Create rate limiter (10 requests per second) - fallback to custom implementation
	rateLimiter := NewRateLimiter(DefaultRateLimit)

	totalImported := 0

	// Import from each room
	for i, roomID := range roomIDs {
		fmt.Printf("\n[%d/%d] Processing room: %s\n", i+1, len(roomIDs), roomID)

		count, err := importEventsFromRoomLegacy(client, roomID, limit, rateLimiter)
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

// importEventsFromRoomLegacy imports events from a specific room (legacy implementation with DuckDB)
func importEventsFromRoomLegacy(client *mautrix.Client, roomID string, limit int, rateLimiter *RateLimiter) (int, error) {
	// Get room display name for logging
	displayName, err := GetRoomDisplayName(client, roomID)
	if err != nil {
		displayName = roomID
	}

	fmt.Printf("Reading events from room %q...\n", displayName)

	importCount := 0
	batchSize := 1000
	totalEventsRead := 0

	// Rate limit the initial request
	rateLimiter.Wait()

	// Get initial events from the room
	resp, err := client.Messages(context.Background(), id.RoomID(roomID), "", "", mautrix.DirectionBackward, nil, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to get messages from room %s: %w", roomID, err)
	}

	for {
		totalEventsRead += len(resp.Chunk)

		// Process events from this batch using DuckDB
		count, err := processEventBatchLegacy(resp.Chunk, roomID, limit-importCount)
		if err != nil {
			return importCount, err
		}
		importCount += count

		// Show progress with more detail
		if limit > 0 {
			percentage := float64(importCount) / float64(limit) * 100
			fmt.Printf("Progress: %d/%d messages imported (%.1f%%), %d events read\n",
				importCount, limit, percentage, totalEventsRead)
		} else {
			fmt.Printf("Progress: %d messages imported, %d events read\n", importCount, totalEventsRead)
		}

		// Check if we've reached the limit
		if limit > 0 && importCount >= limit {
			fmt.Printf("✓ Reached import limit of %d messages\n", limit)
			break
		}

		// Check if there are more events to fetch
		if resp.End == "" || len(resp.Chunk) == 0 {
			break
		}

		fmt.Printf("Read %d events...\n", len(resp.Chunk))

		// Rate limit before next API call
		rateLimiter.Wait()

		// Get next batch
		resp, err = client.Messages(context.Background(), id.RoomID(roomID), resp.End, "", mautrix.DirectionBackward, nil, batchSize)
		if err != nil {
			return importCount, fmt.Errorf("failed to get more messages: %w", err)
		}
	}

	return importCount, nil
}

// processEventBatchLegacy processes a batch of events and saves them to DuckDB (legacy implementation)
func processEventBatchLegacy(events []*event.Event, roomID string, remainingLimit int) (int, error) {
	ctx := context.Background()
	importCount := 0

	// Use smaller batch sizes for database operations to manage memory
	const dbBatchSize = 100
	var messageBatch []*Message

	for _, evt := range events {
		// Check limit
		if remainingLimit > 0 && importCount >= remainingLimit {
			break
		}

		// Filter for message events
		if !IsMessageEvent(evt.Type) {
			continue
		}

		// Skip redacted messages
		if evt.Unsigned.RedactedBecause != nil {
			continue
		}

		// Convert event to Message struct
		message, err := convertEventToMessage(evt, roomID)
		if err != nil {
			log.Printf("Failed to convert event %s: %v", evt.ID, err)
			continue
		}

		// Validate message
		if err := message.Validate(); err != nil {
			log.Printf("Invalid message %s: %v", evt.ID, err)
			continue
		}

		// Add to batch
		messageBatch = append(messageBatch, message)

		// Process batch when it reaches the limit or this is the last message
		if len(messageBatch) >= dbBatchSize || (remainingLimit > 0 && importCount+len(messageBatch) >= remainingLimit) {
			insertedCount, err := GetDatabase().InsertMessageBatch(ctx, messageBatch)
			if err != nil {
				log.Printf("Failed to insert batch: %v", err)
				// Continue with remaining messages despite batch failure
			} else {
				importCount += insertedCount
			}
			// Clear batch to free memory
			messageBatch = messageBatch[:0]
		}
	}

	// Process any remaining messages in the batch
	if len(messageBatch) > 0 {
		insertedCount, err := GetDatabase().InsertMessageBatch(ctx, messageBatch)
		if err != nil {
			log.Printf("Failed to insert final batch: %v", err)
		} else {
			importCount += insertedCount
		}
	}

	return importCount, nil
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
	// Use the raw content directly - DuckDB JSON storage doesn't have MongoDB's dot limitation
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


