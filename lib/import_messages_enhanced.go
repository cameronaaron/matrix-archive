package archive

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

var supportedMessageEventTypes = []event.Type{
	event.EventMessage,
	event.EventReaction,
	event.EventEncrypted, // We'll handle encrypted events if possible
	event.EventRedaction,
}

// EnhancedMatrixClient wraps mautrix.Client with enhanced functionality
type EnhancedMatrixClient struct {
	*mautrix.Client
	db            DatabaseInterface
	stateStore    mautrix.StateStore
	enableRetries bool
	maxRetries    int
	backoffTime   time.Duration
}

// NewEnhancedMatrixClient creates a new enhanced Matrix client
func NewEnhancedMatrixClient(homeserverURL string, userID id.UserID, accessToken string, db DatabaseInterface) (*EnhancedMatrixClient, error) {
	// Create base mautrix client
	client, err := mautrix.NewClient(homeserverURL, userID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create mautrix client: %w", err)
	}

	// Configure built-in rate limiting and retries
	client.DefaultHTTPRetries = 3
	client.DefaultHTTPBackoff = 2 * time.Second
	client.IgnoreRateLimit = false // Use mautrix built-in rate limiting

	// Create state store for room metadata caching
	stateStore := mautrix.NewMemoryStateStore()
	client.StateStore = stateStore

	enhanced := &EnhancedMatrixClient{
		Client:        client,
		db:            db,
		stateStore:    stateStore,
		enableRetries: true,
		maxRetries:    3,
		backoffTime:   2 * time.Second,
	}

	return enhanced, nil
}

// ImportMessages imports messages from Matrix rooms using enhanced mautrix-go features
func ImportMessagesEnhanced(db DatabaseInterface, limit int) error {
	// Initialize database connection
	if err := InitDuckDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer CloseDatabase()

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

	// Create enhanced client
	enhanced, err := NewEnhancedMatrixClient(
		client.HomeserverURL.String(),
		client.UserID,
		client.AccessToken,
		db,
	)
	if err != nil {
		return fmt.Errorf("failed to create enhanced client: %w", err)
	}

	totalImported := 0

	// Import from each room
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
	totalCount, err := db.GetMessageCount(context.Background(), nil)
	if err != nil {
		log.Printf("Failed to count total messages: %v", err)
	} else {
		fmt.Printf("The database now has %d total messages\n", totalCount)
	}

	return nil
}

// importEventsFromRoom imports events from a specific room using enhanced features
func (e *EnhancedMatrixClient) importEventsFromRoom(roomID string, limit int) (int, error) {
	ctx := context.Background()
	roomIDTyped := id.RoomID(roomID)

	// Get room state information using mautrix state store
	_, err := e.StateStore.GetRoomJoinedOrInvitedMembers(ctx, roomIDTyped)
	if err != nil {
		log.Printf("Warning: Could not get room members for %s: %v", roomID, err)
	}

	// Use mautrix built-in pagination for message history
	importCount := 0
	var nextBatch string

	for {
		// Check if we've reached the limit
		if limit > 0 && importCount >= limit {
			break
		}

		// Calculate how many messages to fetch in this batch
		batchLimit := 100 // Default batch size
		if limit > 0 && limit-importCount < batchLimit {
			batchLimit = limit - importCount
		}

		// Get messages using mautrix built-in pagination
		messages, err := e.Messages(ctx, roomIDTyped, nextBatch, "", mautrix.DirectionBackward, nil, batchLimit)
		if err != nil {
			return importCount, fmt.Errorf("failed to fetch messages: %w", err)
		}

		if len(messages.Chunk) == 0 {
			break
		}

		// Process the batch using enhanced event processing
		batchCount, err := e.processEventBatchEnhanced(messages.Chunk, roomID, limit-importCount)
		if err != nil {
			log.Printf("Error processing event batch: %v", err)
		} else {
			importCount += batchCount
		}

		// Update next batch token
		nextBatch = messages.End
		if nextBatch == "" {
			break
		}

		// Log progress
		fmt.Printf("  Processed batch of %d events, total imported: %d\n", len(messages.Chunk), importCount)
	}

	return importCount, nil
}

// processEventBatchEnhanced processes events using mautrix built-in parsers
func (e *EnhancedMatrixClient) processEventBatchEnhanced(events []*event.Event, roomID string, remainingLimit int) (int, error) {
	ctx := context.Background()
	importCount := 0

	// Use smaller batch sizes for database operations
	const dbBatchSize = 100
	var messageBatch []*Message

	for _, evt := range events {
		// Check limit
		if remainingLimit > 0 && importCount >= remainingLimit {
			break
		}

		// Filter for supported message events using mautrix built-in type checking
		if !e.isMessageEvent(evt.Type) {
			continue
		}

		// Skip redacted messages using mautrix built-in redaction handling
		if evt.Unsigned.RedactedBecause != nil {
			continue
		}

		// Convert event to Message struct using enhanced parsing
		message, err := e.convertEventToMessageEnhanced(evt, roomID)
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

		// Process batch when it reaches the limit
		if len(messageBatch) >= dbBatchSize || (remainingLimit > 0 && importCount+len(messageBatch) >= remainingLimit) {
			insertedCount, err := e.db.InsertMessageBatch(ctx, messageBatch)
			if err != nil {
				log.Printf("Failed to insert batch: %v", err)
			} else {
				importCount += insertedCount
			}
			// Clear batch to free memory
			messageBatch = messageBatch[:0]
		}
	}

	// Process any remaining messages in the batch
	if len(messageBatch) > 0 {
		insertedCount, err := e.db.InsertMessageBatch(ctx, messageBatch)
		if err != nil {
			log.Printf("Failed to insert final batch: %v", err)
		} else {
			importCount += insertedCount
		}
	}

	return importCount, nil
}

// isMessageEvent checks if an event type is a supported message event using mautrix constants (exported for testing)
func (e *EnhancedMatrixClient) IsMessageEvent(eventType event.Type) bool {
	return e.isMessageEvent(eventType)
}

// isMessageEvent checks if an event type is a supported message event using mautrix constants
func (e *EnhancedMatrixClient) isMessageEvent(eventType event.Type) bool {
	// Use mautrix built-in event type constants
	switch eventType {
	case event.EventMessage,
		event.EventReaction,
		event.EventEncrypted: // Handle encrypted events if possible
		return true
	default:
		return false
	}
}

// convertEventToMessageEnhanced converts a Matrix event using mautrix built-in parsers (exported for testing)
func (e *EnhancedMatrixClient) ConvertEventToMessageEnhanced(evt *event.Event, roomID string) (*Message, error) {
	return e.convertEventToMessageEnhanced(evt, roomID)
}

// convertEventToMessageEnhanced converts a Matrix event using mautrix built-in parsers
func (e *EnhancedMatrixClient) convertEventToMessageEnhanced(evt *event.Event, roomID string) (*Message, error) {
	// Use mautrix built-in content parsing
	var content map[string]interface{}

	// Parse event content using mautrix built-in parsers
	switch evt.Type {
	case event.EventMessage:
		// Parse message content using mautrix built-in message content parser
		if msgContent, ok := evt.Content.Parsed.(*event.MessageEventContent); ok {
			content = map[string]interface{}{
				"msgtype": msgContent.MsgType,
				"body":    msgContent.Body,
			}

			// Add formatted body if present
			if msgContent.FormattedBody != "" {
				content["formatted_body"] = msgContent.FormattedBody
				content["format"] = msgContent.Format
			}

			// Add file info for media messages
			if msgContent.File != nil {
				content["file"] = msgContent.File
			}
			if msgContent.Info != nil {
				content["info"] = msgContent.Info
			}
			if msgContent.URL != "" {
				content["url"] = msgContent.URL
			}
		} else {
			// Fallback to raw content
			content = evt.Content.Raw
		}

	case event.EventReaction:
		// Parse reaction content using mautrix built-in reaction parser
		if reactionContent, ok := evt.Content.Parsed.(*event.ReactionEventContent); ok {
			content = map[string]interface{}{
				"m.relates_to": map[string]interface{}{
					"rel_type": reactionContent.RelatesTo.Type,
					"event_id": reactionContent.RelatesTo.EventID,
					"key":      reactionContent.RelatesTo.Key,
				},
			}
		} else {
			content = evt.Content.Raw
		}

	case event.EventEncrypted:
		// For encrypted events, store the raw encrypted content
		// In a full implementation, we'd decrypt using mautrix crypto
		content = evt.Content.Raw

	default:
		// For other events, use raw content
		content = evt.Content.Raw
	}

	// Use content directly - DuckDB JSON storage doesn't need dot replacement
	processedContent := content

	message := &Message{
		RoomID:      roomID,
		EventID:     evt.ID.String(),
		Sender:      evt.Sender.String(),
		MessageType: "m.room.message",
		Timestamp:   time.Unix(evt.Timestamp/1000, (evt.Timestamp%1000)*1000000),
		Content:     processedContent,
	}

	return message, nil
}



// DownloadMedia downloads media using mautrix built-in media functionality
func (e *EnhancedMatrixClient) DownloadMedia(ctx context.Context, mxcURL string) ([]byte, error) {
	if !strings.HasPrefix(mxcURL, "mxc://") {
		return nil, fmt.Errorf("invalid mxc URL: %s", mxcURL)
	}

	// Parse mxc URL and convert to ContentURI
	contentURI, err := id.ParseContentURI(mxcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mxc URL: %w", err)
	}

	// Use mautrix built-in media download
	return e.DownloadBytes(ctx, contentURI)
}

// GetRoomDisplayName gets room display name using mautrix state store
func (e *EnhancedMatrixClient) GetRoomDisplayName(ctx context.Context, roomID id.RoomID) string {
	// Try to get room name from state store using our own helper since StateStore doesn't have GetStateEvent
	// For now, fallback to room ID
	return roomID.String()
}

// GetRoomMembers gets room members using mautrix state store
func (e *EnhancedMatrixClient) GetRoomMembers(ctx context.Context, roomID id.RoomID) ([]id.UserID, error) {
	return e.StateStore.GetRoomJoinedOrInvitedMembers(ctx, roomID)
}
