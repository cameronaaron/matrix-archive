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

// NewEnhancedMatrixClient creates a new enhanced Matrix client from an existing client
func NewEnhancedMatrixClient(client *mautrix.Client, db DatabaseInterface) (*EnhancedMatrixClient, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	// Configure built-in rate limiting and retries
	client.DefaultHTTPRetries = 3
	client.DefaultHTTPBackoff = 2 * time.Second
	client.IgnoreRateLimit = false // Use mautrix built-in rate limiting

	// Create state store for room metadata caching if not already set
	if client.StateStore == nil {
		stateStore := mautrix.NewMemoryStateStore()
		client.StateStore = stateStore
	}

	enhanced := &EnhancedMatrixClient{
		Client:        client,
		db:            db,
		stateStore:    client.StateStore,
		enableRetries: true,
		maxRetries:    3,
		backoffTime:   2 * time.Second,
	}

	// Check if the client has crypto enabled
	if client.Crypto != nil {
		log.Printf("Enhanced client using crypto-enabled Matrix client")
	} else {
		log.Printf("Enhanced client using non-crypto Matrix client")
	}

	return enhanced, nil
}

// importEventsFromRoom imports events from a specific room using enhanced mautrix-go features

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

// convertEventToMessageEnhanced converts a Matrix event using mautrix built-in parsers
func (e *EnhancedMatrixClient) convertEventToMessageEnhanced(evt *event.Event, roomID string) (*Message, error) {
	// Use mautrix built-in content parsing
	var content map[string]interface{}

	log.Printf("DEBUG: Processing event %s of type %s", evt.ID, evt.Type)
	log.Printf("DEBUG: Raw content: %+v", evt.Content.Raw)

	// Parse event content using mautrix built-in parsers
	switch evt.Type {
	case event.EventMessage:
		// Parse message content using mautrix built-in message content parser
		if msgContent, ok := evt.Content.Parsed.(*event.MessageEventContent); ok {
			content = map[string]interface{}{
				"msgtype": msgContent.MsgType,
				"body":    msgContent.Body,
			}

			log.Printf("DEBUG: Parsed message content - msgtype: %s, body: %s", msgContent.MsgType, msgContent.Body)

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
			log.Printf("DEBUG: Failed to parse message content, using raw: %+v", evt.Content.Raw)
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
		// For encrypted events, we need to explicitly decrypt them
		// The cryptohelper doesn't automatically decrypt historical events
		if e.Client.Crypto != nil {
			log.Printf("DEBUG: Attempting to decrypt encrypted event %s", evt.ID)

			// Parse the event content manually first
			err := evt.Content.ParseRaw(evt.Type)
			if err != nil {
				log.Printf("DEBUG: Failed to parse encrypted event content: %v", err)
			} else {
				log.Printf("DEBUG: Successfully parsed event content")
			}

			// Try to decrypt the event using the crypto helper
			decryptedEvt, err := e.Client.Crypto.Decrypt(context.Background(), evt)
			if err != nil {
				log.Printf("DEBUG: Failed to decrypt event %s: %v", evt.ID, err)
			} else if decryptedEvt != nil {
				log.Printf("DEBUG: Successfully decrypted event %s", evt.ID)
				// Use the decrypted event content
				if msgContent, ok := decryptedEvt.Content.Parsed.(*event.MessageEventContent); ok {
					content = map[string]interface{}{
						"msgtype": msgContent.MsgType,
						"body":    msgContent.Body,
					}
					if msgContent.FormattedBody != "" {
						content["formatted_body"] = msgContent.FormattedBody
						content["format"] = msgContent.Format
					}
					log.Printf("DEBUG: Decrypted message content - msgtype: %s, body: %s", msgContent.MsgType, msgContent.Body)
				} else {
					// Try to parse the decrypted content directly
					if body, ok := decryptedEvt.Content.Raw["body"].(string); ok {
						content = map[string]interface{}{
							"msgtype": decryptedEvt.Content.Raw["msgtype"],
							"body":    body,
						}
						if formattedBody, exists := decryptedEvt.Content.Raw["formatted_body"]; exists {
							content["formatted_body"] = formattedBody
							content["format"] = decryptedEvt.Content.Raw["format"]
						}
						log.Printf("DEBUG: Decrypted message from raw content - body: %s", body)
					} else {
						// Still couldn't parse decrypted content, fall back to encrypted placeholder
						content = map[string]interface{}{
							"msgtype":    "m.text",
							"body":       "[Encrypted message - decryption not available]",
							"algorithm":  evt.Content.Raw["algorithm"],
							"session_id": evt.Content.Raw["session_id"],
						}
						log.Printf("DEBUG: Decrypted event but couldn't parse content")
					}
				}
			} else {
				// Decryption failed, use encrypted placeholder
				content = map[string]interface{}{
					"msgtype":    "m.text",
					"body":       "[Encrypted message - decryption not available]",
					"algorithm":  evt.Content.Raw["algorithm"],
					"session_id": evt.Content.Raw["session_id"],
				}
				log.Printf("DEBUG: Event decryption returned nil")
			}
		} else {
			// No crypto helper available, use encrypted placeholder
			content = map[string]interface{}{
				"msgtype":    "m.text",
				"body":       "[Encrypted message - decryption not available]",
				"algorithm":  evt.Content.Raw["algorithm"],
				"session_id": evt.Content.Raw["session_id"],
			}
			log.Printf("DEBUG: No crypto helper available for decryption")
		}

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
