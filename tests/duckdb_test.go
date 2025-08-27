package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	archive "github.com/osteele/matrix-archive/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDuckDBDatabase tests the DuckDB database implementation
func TestDuckDBDatabase(t *testing.T) {
	// Use in-memory database for testing
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       true,
	}

	db := archive.NewDuckDBDatabase(config)
	require.NotNil(t, db)

	ctx := context.Background()

	// Test connection
	err := db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Test ping
	err = db.Ping(ctx)
	assert.NoError(t, err)

	// Test creating tables (should be done during Connect)
	err = db.CreateTables(ctx)
	assert.NoError(t, err)

	// Test migration
	err = db.Migrate(ctx)
	assert.NoError(t, err)
}

// TestDuckDBMessageOperations tests message CRUD operations
func TestDuckDBMessageOperations(t *testing.T) {
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	db := archive.NewDuckDBDatabase(config)
	require.NotNil(t, db)

	ctx := context.Background()
	err := db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test message
	testMessage := &archive.Message{
		RoomID:      "!testroom:example.com",
		EventID:     "$testevent:example.com",
		Sender:      "@testuser:example.com",
		UserID:      "@testuser:example.com",
		MessageType: "m.room.message",
		Timestamp:   time.Now(),
		Content: map[string]interface{}{
			"msgtype": "m.text",
			"body":    "Test message",
		},
	}

	// Test validation
	err = testMessage.Validate()
	assert.NoError(t, err)

	// Test insert message
	err = db.InsertMessage(ctx, testMessage)
	assert.NoError(t, err)

	// Test get message
	retrievedMessage, err := db.GetMessage(ctx, testMessage.EventID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedMessage)
	assert.Equal(t, testMessage.RoomID, retrievedMessage.RoomID)
	assert.Equal(t, testMessage.EventID, retrievedMessage.EventID)
	assert.Equal(t, testMessage.Sender, retrievedMessage.Sender)
	assert.Equal(t, testMessage.MessageType, retrievedMessage.MessageType)

	// Test content serialization/deserialization
	assert.Equal(t, "m.text", retrievedMessage.Content["msgtype"])
	assert.Equal(t, "Test message", retrievedMessage.Content["body"])

	// Test message not found
	_, err = db.GetMessage(ctx, "$nonexistent:example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message not found")
}

// TestDuckDBBatchOperations tests batch insert operations
func TestDuckDBBatchOperations(t *testing.T) {
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	db := archive.NewDuckDBDatabase(config)
	require.NotNil(t, db)

	ctx := context.Background()
	err := db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test messages
	messages := []*archive.Message{
		{
			RoomID:      "!testroom:example.com",
			EventID:     "$event1:example.com",
			Sender:      "@user1:example.com",
			MessageType: "m.room.message",
			Timestamp:   time.Now(),
			Content: map[string]interface{}{
				"msgtype": "m.text",
				"body":    "Message 1",
			},
		},
		{
			RoomID:      "!testroom:example.com",
			EventID:     "$event2:example.com",
			Sender:      "@user2:example.com",
			MessageType: "m.room.message",
			Timestamp:   time.Now().Add(time.Minute),
			Content: map[string]interface{}{
				"msgtype": "m.text",
				"body":    "Message 2",
			},
		},
		{
			RoomID:      "!testroom2:example.com",
			EventID:     "$event3:example.com",
			Sender:      "@user1:example.com",
			MessageType: "m.room.message",
			Timestamp:   time.Now().Add(2 * time.Minute),
			Content: map[string]interface{}{
				"msgtype": "m.image",
				"body":    "Image message",
				"url":     "mxc://example.com/image",
			},
		},
	}

	// Test batch insert
	insertedCount, err := db.InsertMessageBatch(ctx, messages)
	assert.NoError(t, err)
	assert.Equal(t, 3, insertedCount)

	// Test get messages count
	count, err := db.GetMessageCount(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Test get messages with filter
	filter := &archive.MessageFilter{
		RoomID: "!testroom:example.com",
	}

	filteredMessages, err := db.GetMessages(ctx, filter, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, filteredMessages, 2)

	// Test get message count with filter
	filteredCount, err := db.GetMessageCount(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), filteredCount)
}

// TestDuckDBFilterOperations tests various filtering scenarios
func TestDuckDBFilterOperations(t *testing.T) {
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	db := archive.NewDuckDBDatabase(config)
	require.NotNil(t, db)

	ctx := context.Background()
	err := db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test data with different timestamps
	baseTime := time.Now()
	messages := []*archive.Message{
		{
			RoomID:      "!room1:example.com",
			EventID:     "$event1:example.com",
			Sender:      "@user1:example.com",
			MessageType: "m.room.message",
			Timestamp:   baseTime,
			Content:     map[string]interface{}{"msgtype": "m.text", "body": "Message 1"},
		},
		{
			RoomID:      "!room1:example.com",
			EventID:     "$event2:example.com",
			Sender:      "@user2:example.com",
			MessageType: "m.room.message",
			Timestamp:   baseTime.Add(time.Hour),
			Content:     map[string]interface{}{"msgtype": "m.text", "body": "Message 2"},
		},
		{
			RoomID:      "!room2:example.com",
			EventID:     "$event3:example.com",
			Sender:      "@user1:example.com",
			MessageType: "m.room.message",
			Timestamp:   baseTime.Add(2 * time.Hour),
			Content:     map[string]interface{}{"msgtype": "m.text", "body": "Message 3"},
		},
	}

	// Insert test data
	insertedCount, err := db.InsertMessageBatch(ctx, messages)
	require.NoError(t, err)
	require.Equal(t, 3, insertedCount)

	// Test filter by sender
	senderFilter := &archive.MessageFilter{
		Sender: "@user1:example.com",
	}

	senderMessages, err := db.GetMessages(ctx, senderFilter, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, senderMessages, 2)

	// Test filter by time range
	startTime := baseTime.Add(30 * time.Minute)
	endTime := baseTime.Add(90 * time.Minute)
	timeFilter := &archive.MessageFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	timeMessages, err := db.GetMessages(ctx, timeFilter, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, timeMessages, 1)

	// Test filter by event ID
	eventFilter := &archive.MessageFilter{
		EventID: "$event2:example.com",
	}

	eventMessages, err := db.GetMessages(ctx, eventFilter, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, eventMessages, 1)
	assert.Equal(t, "$event2:example.com", eventMessages[0].EventID)

	// Test combined filters
	combinedFilter := &archive.MessageFilter{
		RoomID: "!room1:example.com",
		Sender: "@user1:example.com",
	}

	combinedMessages, err := db.GetMessages(ctx, combinedFilter, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, combinedMessages, 1)
}

// TestDuckDBRoomOperations tests room-related operations
func TestDuckDBRoomOperations(t *testing.T) {
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	db := archive.NewDuckDBDatabase(config)
	require.NotNil(t, db)

	ctx := context.Background()
	err := db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Insert messages in different rooms
	messages := []*archive.Message{
		{
			RoomID:      "!room1:example.com",
			EventID:     "$event1:example.com",
			Sender:      "@user1:example.com",
			MessageType: "m.room.message",
			Timestamp:   time.Now(),
			Content:     map[string]interface{}{"msgtype": "m.text", "body": "Message 1"},
		},
		{
			RoomID:      "!room1:example.com",
			EventID:     "$event2:example.com",
			Sender:      "@user2:example.com",
			MessageType: "m.room.message",
			Timestamp:   time.Now(),
			Content:     map[string]interface{}{"msgtype": "m.text", "body": "Message 2"},
		},
		{
			RoomID:      "!room2:example.com",
			EventID:     "$event3:example.com",
			Sender:      "@user1:example.com",
			MessageType: "m.room.message",
			Timestamp:   time.Now(),
			Content:     map[string]interface{}{"msgtype": "m.text", "body": "Message 3"},
		},
	}

	insertedCount, err := db.InsertMessageBatch(ctx, messages)
	require.NoError(t, err)
	require.Equal(t, 3, insertedCount)

	// Test get rooms
	rooms, err := db.GetRooms(ctx)
	assert.NoError(t, err)
	assert.Len(t, rooms, 2)
	assert.Contains(t, rooms, "!room1:example.com")
	assert.Contains(t, rooms, "!room2:example.com")

	// Test room message count
	room1Count, err := db.GetRoomMessageCount(ctx, "!room1:example.com")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), room1Count)

	room2Count, err := db.GetRoomMessageCount(ctx, "!room2:example.com")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), room2Count)
}

// TestDuckDBDeleteOperations tests delete operations
func TestDuckDBDeleteOperations(t *testing.T) {
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	db := archive.NewDuckDBDatabase(config)
	require.NotNil(t, db)

	ctx := context.Background()
	err := db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Insert test message
	testMessage := &archive.Message{
		RoomID:      "!testroom:example.com",
		EventID:     "$testevent:example.com",
		Sender:      "@testuser:example.com",
		MessageType: "m.room.message",
		Timestamp:   time.Now(),
		Content:     map[string]interface{}{"msgtype": "m.text", "body": "Test message"},
	}

	err = db.InsertMessage(ctx, testMessage)
	require.NoError(t, err)

	// Verify message exists
	_, err = db.GetMessage(ctx, testMessage.EventID)
	assert.NoError(t, err)

	// Test delete message
	err = db.DeleteMessage(ctx, testMessage.EventID)
	assert.NoError(t, err)

	// Verify message is deleted
	_, err = db.GetMessage(ctx, testMessage.EventID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message not found")

	// Test delete non-existent message
	err = db.DeleteMessage(ctx, "$nonexistent:example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message not found")
}

// TestDuckDBPaginationAndLimits tests pagination and limit functionality
func TestDuckDBPaginationAndLimits(t *testing.T) {
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	db := archive.NewDuckDBDatabase(config)
	require.NotNil(t, db)

	ctx := context.Background()
	err := db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Insert multiple messages with incremental timestamps
	baseTime := time.Now()
	var messages []*archive.Message
	for i := 0; i < 10; i++ {
		message := &archive.Message{
			RoomID:      "!testroom:example.com",
			EventID:     fmt.Sprintf("$event%d:example.com", i),
			Sender:      "@testuser:example.com",
			MessageType: "m.room.message",
			Timestamp:   baseTime.Add(time.Duration(i) * time.Minute),
			Content: map[string]interface{}{
				"msgtype": "m.text",
				"body":    fmt.Sprintf("Message %d", i),
			},
		}
		messages = append(messages, message)
	}

	insertedCount, err := db.InsertMessageBatch(ctx, messages)
	require.NoError(t, err)
	require.Equal(t, 10, insertedCount)

	// Test limit
	limitedMessages, err := db.GetMessages(ctx, nil, 5, 0)
	assert.NoError(t, err)
	assert.Len(t, limitedMessages, 5)

	// Test offset
	offsetMessages, err := db.GetMessages(ctx, nil, 5, 5)
	assert.NoError(t, err)
	assert.Len(t, offsetMessages, 5)

	// Verify no overlap between pages
	for _, msg1 := range limitedMessages {
		for _, msg2 := range offsetMessages {
			assert.NotEqual(t, msg1.EventID, msg2.EventID)
		}
	}

	// Test beyond available data
	beyondMessages, err := db.GetMessages(ctx, nil, 5, 20)
	assert.NoError(t, err)
	assert.Len(t, beyondMessages, 0)
}

// TestMessageHelperFunctions tests the helper functions on Message struct
func TestMessageHelperFunctions(t *testing.T) {
	// Test text message
	textMessage := &archive.Message{
		Content: map[string]interface{}{
			"msgtype": "m.text",
			"body":    "Hello world",
		},
	}

	assert.False(t, textMessage.IsImage())
	assert.Empty(t, textMessage.ImageURL())
	assert.Empty(t, textMessage.ThumbnailURL())

	// Test image message
	imageMessage := &archive.Message{
		Content: map[string]interface{}{
			"msgtype": "m.image",
			"body":    "image.png",
			"url":     "mxc://example.com/image",
			"info": map[string]interface{}{
				"thumbnail_url": "mxc://example.com/thumb",
			},
		},
	}

	assert.True(t, imageMessage.IsImage())
	assert.Equal(t, "mxc://example.com/image", imageMessage.ImageURL())
	assert.Equal(t, "mxc://example.com/thumb", imageMessage.ThumbnailURL())

	// Test JSON serialization/deserialization
	jsonStr, err := textMessage.ContentJSON()
	assert.NoError(t, err)
	assert.Contains(t, jsonStr, "m.text")
	assert.Contains(t, jsonStr, "Hello world")

	newMessage := &archive.Message{}
	err = newMessage.SetContentFromJSON(jsonStr)
	assert.NoError(t, err)
	assert.Equal(t, "m.text", newMessage.Content["msgtype"])
	assert.Equal(t, "Hello world", newMessage.Content["body"])
}

// TestMessageFilter tests the MessageFilter functionality
func TestMessageFilter(t *testing.T) {
	baseTime := time.Now()
	startTime := baseTime.Add(-time.Hour)
	endTime := baseTime.Add(time.Hour)

	filter := &archive.MessageFilter{
		RoomID:    "!testroom:example.com",
		Sender:    "@testuser:example.com",
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	whereClause, args := filter.ToSQL()

	assert.Contains(t, whereClause, "room_id = ?")
	assert.Contains(t, whereClause, "sender = ?")
	assert.Contains(t, whereClause, "timestamp >= ?")
	assert.Contains(t, whereClause, "timestamp <= ?")

	assert.Len(t, args, 4)
	assert.Equal(t, "!testroom:example.com", args[0])
	assert.Equal(t, "@testuser:example.com", args[1])
	assert.Equal(t, startTime, args[2])
	assert.Equal(t, endTime, args[3])

	// Test empty filter
	emptyFilter := &archive.MessageFilter{}
	emptyWhereClause, emptyArgs := emptyFilter.ToSQL()
	assert.Empty(t, emptyWhereClause)
	assert.Empty(t, emptyArgs)

	// Test nil filter
	nilWhereClause, nilArgs := (*archive.MessageFilter)(nil).ToSQL()
	assert.Empty(t, nilWhereClause)
	assert.Empty(t, nilArgs)
}

// TestInitDatabase tests the global database initialization functions
func TestInitDatabase(t *testing.T) {
	// Test with custom config
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       true,
	}

	err := archive.InitDatabase(config)
	assert.NoError(t, err)

	// Test getting database instance
	db := archive.GetDatabase()
	assert.NotNil(t, db)

	// Test database operations
	ctx := context.Background()
	err = db.Ping(ctx)
	assert.NoError(t, err)

	// Clean up
	err = archive.CloseDatabase()
	assert.NoError(t, err)
}
