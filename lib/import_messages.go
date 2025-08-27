package archive

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
	DefaultRateLimit = 10               // requests per second
	RateLimitWindow  = time.Second      // rate limit window
	MaxBurstSize     = 20               // maximum burst size
)

// RateLimiter provides simple rate limiting for API calls
type RateLimiter struct {
	rate     time.Duration
	lastCall time.Time
}

// NewRateLimiter creates a new rate limiter with the specified rate (requests per second)
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
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

// importMessages imports messages from Matrix rooms into MongoDB
func ImportMessages(limit int) error {
	// Initialize database connection
	if err := InitMongoDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer CloseMongoDB()

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

	// Create rate limiter (10 requests per second)
	rateLimiter := NewRateLimiter(DefaultRateLimit)

	totalImported := 0

	// Import from each room
	for i, roomID := range roomIDs {
		fmt.Printf("\n[%d/%d] Processing room: %s\n", i+1, len(roomIDs), roomID)
		
		count, err := importEventsFromRoom(client, roomID, limit, rateLimiter)
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
	collection := GetMessagesCollection()
	totalCount, err := collection.CountDocuments(context.Background(), bson.M{})
	if err != nil {
		log.Printf("Failed to count total messages: %v", err)
	} else {
		fmt.Printf("The database now has %d total messages\n", totalCount)
	}

	return nil
}

// importEventsFromRoom imports events from a specific room
func importEventsFromRoom(client *mautrix.Client, roomID string, limit int, rateLimiter *RateLimiter) (int, error) {
	// Get room display name for logging
	displayName, err := GetRoomDisplayName(client, roomID)
	if err != nil {
		displayName = roomID
	}

	fmt.Printf("Reading events from room %q...\n", displayName)

	collection := GetMessagesCollection()
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
		
		// Process events from this batch
		count, err := processEventBatch(collection, resp.Chunk, roomID, limit-importCount)
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

// processEventBatch processes a batch of events and saves them to MongoDB
func processEventBatch(collection *mongo.Collection, events []*event.Event, roomID string, remainingLimit int) (int, error) {
	ctx := context.Background()
	importCount := 0
	
	// Use smaller batch sizes for database operations to manage memory
	const dbBatchSize = 100
	var messageBatch []interface{}

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
			insertedCount, err := insertMessageBatch(collection, ctx, messageBatch)
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
		insertedCount, err := insertMessageBatch(collection, ctx, messageBatch)
		if err != nil {
			log.Printf("Failed to insert final batch: %v", err)
		} else {
			importCount += insertedCount
		}
	}

	return importCount, nil
}

// insertMessageBatch inserts a batch of messages efficiently
func insertMessageBatch(collection *mongo.Collection, ctx context.Context, messages []interface{}) (int, error) {
	if len(messages) == 0 {
		return 0, nil
	}
	
	result, err := collection.InsertMany(ctx, messages)
	if err != nil {
		return 0, err
	}
	
	return len(result.InsertedIDs), nil
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
	// Replace dots with bullets in content to avoid MongoDB issues
	processedContent := ReplaceDots(evt.Content.Raw)
	
	// Convert to bson.M, handling the case where ReplaceDots returns interface{}
	var content bson.M
	if contentMap, ok := processedContent.(bson.M); ok {
		content = contentMap
	} else if contentMap, ok := processedContent.(map[string]interface{}); ok {
		content = bson.M(contentMap)
	} else {
		// Fallback for unexpected types
		content = bson.M{"data": processedContent}
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

// ReplaceDots recursively replaces '.' with '•' in map keys to avoid MongoDB issues
func ReplaceDots(obj interface{}) interface{} {
	switch v := obj.(type) {
	case map[string]interface{}:
		result := bson.M{}
		for key, value := range v {
			newKey := strings.ReplaceAll(key, ".", "•")
			result[newKey] = ReplaceDots(value)
		}
		return result
	case bson.M:
		result := bson.M{}
		for key, value := range v {
			newKey := strings.ReplaceAll(key, ".", "•")
			result[newKey] = ReplaceDots(value)
		}
		return result
	case []interface{}:
		// Handle arrays by processing each element
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = ReplaceDots(item)
		}
		return result
	default:
		// For primitive types (strings, numbers, booleans), return as-is
		return v
	}
}
