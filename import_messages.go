package main

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

// importMessages imports messages from Matrix rooms into MongoDB
func importMessages(limit int) error {
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

	totalImported := 0

	// Import from each room
	for _, roomID := range roomIDs {
		count, err := importEventsFromRoom(client, roomID, limit)
		if err != nil {
			log.Printf("Error importing from room %s: %v", roomID, err)
			continue
		}
		totalImported += count
		fmt.Printf("Imported %d messages from room %s\n", count, roomID)
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
func importEventsFromRoom(client *mautrix.Client, roomID string, limit int) (int, error) {
	// Get room display name for logging
	displayName, err := getRoomDisplayName(client, roomID)
	if err != nil {
		displayName = roomID
	}

	fmt.Printf("Reading events from room %q...\n", displayName)

	collection := GetMessagesCollection()
	importCount := 0
	batchSize := 1000

	// Get initial events from the room
	resp, err := client.Messages(context.Background(), id.RoomID(roomID), "", "", mautrix.DirectionBackward, nil, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to get messages from room %s: %w", roomID, err)
	}

	for {
		// Process events from this batch
		count, err := processEventBatch(collection, resp.Chunk, roomID, limit-importCount)
		if err != nil {
			return importCount, err
		}
		importCount += count

		// Check if we've reached the limit
		if limit > 0 && importCount >= limit {
			break
		}

		// Check if there are more events to fetch
		if resp.End == "" || len(resp.Chunk) == 0 {
			break
		}

		fmt.Printf("Read %d events...\n", len(resp.Chunk))

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

	for _, evt := range events {
		// Check limit
		if remainingLimit > 0 && importCount >= remainingLimit {
			break
		}

		// Filter for message events
		if !isMessageEvent(evt.Type) {
			continue
		}

		// Skip redacted messages
		if evt.Unsigned.RedactedBecause != nil {
			continue
		}

		// Check if message already exists
		filter := bson.M{
			"event_id": evt.ID.String(),
			"room_id":  roomID,
		}
		count, err := collection.CountDocuments(ctx, filter)
		if err != nil {
			return importCount, fmt.Errorf("failed to check existing message: %w", err)
		}
		if count > 0 {
			continue // Message already exists
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

		// Save message to database
		_, err = collection.InsertOne(ctx, message)
		if err != nil {
			log.Printf("Failed to save message %s: %v", evt.ID, err)
			continue
		}

		importCount++
	}

	return importCount, nil
}

// isMessageEvent checks if an event type is a message event
func isMessageEvent(eventType event.Type) bool {
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
	content := replaceDots(evt.Content.Raw)

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

// replaceDots recursively replaces '.' with '•' in map keys to avoid MongoDB issues
func replaceDots(obj interface{}) bson.M {
	switch v := obj.(type) {
	case map[string]interface{}:
		result := bson.M{}
		for key, value := range v {
			newKey := strings.ReplaceAll(key, ".", "•")
			result[newKey] = replaceDots(value)
		}
		return result
	case bson.M:
		result := bson.M{}
		for key, value := range v {
			newKey := strings.ReplaceAll(key, ".", "•")
			result[newKey] = replaceDots(value)
		}
		return result
	default:
		// For non-map types, just return as bson.M with the value
		if v == nil {
			return bson.M{}
		}
		return bson.M{"data": v}
	}
}
