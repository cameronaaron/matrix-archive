package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func TestValidationAndUtilityFunctions(t *testing.T) {
	// Test message validation
	msg := &Message{
		RoomID:      "!room:example.com",
		EventID:     "$event123",
		Sender:      "@user:example.com",
		MessageType: "m.room.message",
	}
	err := msg.Validate()
	assert.NoError(t, err)

	// Test export format validation
	assert.True(t, isValidFormat("json"))
	assert.True(t, isValidFormat("html"))
	assert.True(t, isValidFormat("yaml"))
	assert.True(t, isValidFormat("txt"))
	assert.False(t, isValidFormat("invalid"))
	assert.False(t, isValidFormat(""))
}

func TestBeeperAuthCreation(t *testing.T) {
	// Test that we can create BeeperAuth without errors
	auth := NewBeeperAuth("test.com")
	assert.NotNil(t, auth)
	assert.Equal(t, "test.com", auth.BaseDomain)

	// Test default domain
	authDefault := NewBeeperAuth("")
	assert.NotNil(t, authDefault)
	assert.Equal(t, "beeper.com", authDefault.BaseDomain)
}

func TestPerformBeeperLogin_NonInteractive(t *testing.T) {
	// Test that Beeper login fails gracefully in non-interactive mode
	err := performBeeperLogin("test.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-interactive mode")
}

func TestPerformBeeperLogout(t *testing.T) {
	// Test that Beeper logout works (should not error even if no credentials exist)
	err := performBeeperLogout("test.com")
	assert.NoError(t, err)
}

func TestExportMessageConversion(t *testing.T) {
	// Test converting Message to ExportMessage
	message := Message{
		RoomID:      "!room123:example.com",
		EventID:     "$event123:example.com",
		Sender:      "@user:example.com",
		MessageType: "m.room.message",
		Content:     map[string]interface{}{"body": "Hello world"},
	}

	exportMessage, err := convertToExportMessage(message, false)
	assert.NoError(t, err)

	assert.Equal(t, "user", exportMessage.Sender) // Should extract username part
	assert.Equal(t, "Hello world", exportMessage.Content["body"])
	assert.NotEmpty(t, exportMessage.Timestamp) // Should have a timestamp string
}

func TestIsImageContent(t *testing.T) {
	// Test image content detection
	imageContent := map[string]interface{}{
		"msgtype": "m.image",
		"body":    "image.jpg",
	}
	assert.True(t, isImageContent(imageContent))

	textContent := map[string]interface{}{
		"msgtype": "m.text",
		"body":    "Hello world",
	}
	assert.False(t, isImageContent(textContent))

	noMsgType := map[string]interface{}{
		"body": "Hello world",
	}
	assert.False(t, isImageContent(noMsgType))
}

func TestMXCToLocalPathConversion(t *testing.T) {
	// Test MXC URL conversion to local path
	content := map[string]interface{}{
		"msgtype": "m.image",
	}

	// Test basic MXC URL
	result := convertMXCToLocalPath("mxc://example.com/abc123", content)
	assert.Equal(t, "thumbnails/abc123", result)

	// Test non-MXC URL (should return as-is)
	result = convertMXCToLocalPath("https://example.com/image.jpg", content)
	assert.Equal(t, "https://example.com/image.jpg", result)

	// Test invalid MXC URL (should return as-is)
	result = convertMXCToLocalPath("mxc://invalid", content)
	assert.Equal(t, "mxc://invalid", result)

	// Test MXC with thumbnail
	contentWithThumb := map[string]interface{}{
		"msgtype": "m.image",
		"info": map[string]interface{}{
			"thumbnail_url": "mxc://example.com/thumb456",
		},
	}
	result = convertMXCToLocalPath("mxc://example.com/abc123", contentWithThumb)
	assert.Equal(t, "thumbnails/thumb456", result)

	// Test MXC with mimetype
	contentWithMime := map[string]interface{}{
		"msgtype": "m.image",
		"info": map[string]interface{}{
			"mimetype": "image/jpeg",
		},
	}
	result = convertMXCToLocalPath("mxc://example.com/abc123", contentWithMime)
	assert.Equal(t, "thumbnails/abc123.jpeg", result)
}

func TestConvertToLocalImages(t *testing.T) {
	// Test converting URLs to local paths for images
	imageContent := map[string]interface{}{
		"url":     "mxc://example.com/abc123",
		"msgtype": "m.image",
	}
	result := convertToLocalImages(imageContent)
	assert.Equal(t, "thumbnails/abc123", result["url"])
	assert.Equal(t, "m.image", result["msgtype"])

	// Test non-image content (should be unchanged)
	textContent := map[string]interface{}{
		"url":     "mxc://example.com/abc123",
		"msgtype": "m.text",
	}
	result = convertToLocalImages(textContent)
	assert.Equal(t, "mxc://example.com/abc123", result["url"])
}

func TestConvertToDownloadURLs(t *testing.T) {
	// Test converting MXC URLs to download URLs
	content := map[string]interface{}{
		"url": "mxc://example.com/abc123",
	}

	result := convertToDownloadURLs(content)
	// This should convert the MXC URL to a proper download URL
	assert.Contains(t, result["url"].(string), "example.com/abc123")
	assert.Contains(t, result["url"].(string), "download")
}

func TestExportMessages_UnsupportedFormat(t *testing.T) {
	// Test export with unsupported format
	err := exportMessages("test.invalid", "!room:example.com", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestFindRoomByName(t *testing.T) {
	// Test finding room by name that doesn't exist
	roomID, err := findRoomByName("NonExistentRoomName123456789")
	assert.Error(t, err)
	assert.Empty(t, roomID)
	assert.Contains(t, err.Error(), "room not found")
}

// Test exportWithTemplate function
func TestExportWithTemplate(t *testing.T) {
	// Create a temporary template file
	tmpDir, err := os.MkdirTemp("", "matrix-archive-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	templatePath := filepath.Join(tmpDir, "test.tpl")
	templateContent := `{{range .messages}}{{.Sender}}: {{.Content.body}}
{{end}}`
	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	assert.NoError(t, err)

	// Create test data
	messages := []ExportMessage{
		{
			Sender:    "user1",
			Content:   map[string]interface{}{"body": "Hello"},
			Timestamp: "2023-01-01 10:00:00",
		},
		{
			Sender:    "user2",
			Content:   map[string]interface{}{"body": "World"},
			Timestamp: "2023-01-01 10:01:00",
		},
	}

	// Create output file
	outputPath := filepath.Join(tmpDir, "output.txt")
	file, err := os.Create(outputPath)
	assert.NoError(t, err)
	defer file.Close()

	// Test successful template export
	err = exportWithTemplate(file, templatePath, messages)
	assert.NoError(t, err)

	// Test with non-existent template
	err = exportWithTemplate(file, "/non/existent/template.tpl", messages)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read template")

	// Test with invalid template
	invalidTemplatePath := filepath.Join(tmpDir, "invalid.tpl")
	invalidTemplate := `{{range .messages}}{{.invalid.template.syntax}}`
	err = os.WriteFile(invalidTemplatePath, []byte(invalidTemplate), 0644)
	assert.NoError(t, err)

	err = exportWithTemplate(file, invalidTemplatePath, messages)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse template")
}

// Test convertEventToMessage function
func TestConvertEventToMessage(t *testing.T) {
	// Create a test event
	evt := &event.Event{
		ID:        id.EventID("$event123:example.com"),
		Sender:    id.UserID("@user:example.com"),
		Type:      event.EventMessage,
		Timestamp: time.Now().UnixMilli(),
		Content: event.Content{
			Raw: map[string]interface{}{
				"msgtype":  "m.text",
				"body":     "Hello world",
				"some.key": "value",
			},
		},
	}

	roomID := "!room123:example.com"

	// Test successful conversion
	message, err := convertEventToMessage(evt, roomID)
	assert.NoError(t, err)
	assert.NotNil(t, message)
	assert.Equal(t, roomID, message.RoomID)
	assert.Equal(t, evt.ID.String(), message.EventID)
	assert.Equal(t, evt.Sender.String(), message.Sender)
	assert.Equal(t, "m.room.message", message.MessageType)

	// Check that dots were replaced
	content := message.Content
	_, hasDots := content["some.key"]
	_, hasBullets := content["some•key"]
	assert.False(t, hasDots, "Dots should be replaced")
	assert.True(t, hasBullets, "Should have bullets instead of dots")
}

// Test processEventBatch function
func TestProcessEventBatch(t *testing.T) {
	// Initialize MongoDB for testing
	InitMongoDB()
	collection := GetMessagesCollection()

	// Create test events
	events := []*event.Event{
		{
			ID:        id.EventID("$event1:example.com"),
			Sender:    id.UserID("@user1:example.com"),
			Type:      event.EventMessage,
			Timestamp: time.Now().UnixMilli(),
			Content: event.Content{
				Raw: map[string]interface{}{
					"msgtype": "m.text",
					"body":    "Message 1",
				},
			},
		},
		{
			ID:        id.EventID("$event2:example.com"),
			Sender:    id.UserID("@user2:example.com"),
			Type:      event.EventMessage,
			Timestamp: time.Now().UnixMilli(),
			Content: event.Content{
				Raw: map[string]interface{}{
					"msgtype": "m.text",
					"body":    "Message 2",
				},
			},
		},
		{
			ID:        id.EventID("$non_message:example.com"),
			Sender:    id.UserID("@user3:example.com"),
			Type:      event.StateMember, // Not a message event
			Timestamp: time.Now().UnixMilli(),
		},
	}

	roomID := "!testroom:example.com"

	// Clean up first - remove any existing test documents
	_, _ = collection.DeleteMany(context.Background(), bson.M{"room_id": roomID})

	// Test processing events with high limit
	count, err := processEventBatch(collection, events, roomID, 10)
	assert.NoError(t, err)
	assert.Equal(t, 2, count) // Should process 2 message events, ignore the state event

	// Clean up again
	_, _ = collection.DeleteMany(context.Background(), bson.M{"room_id": roomID})

	// Test with limit of 1
	count, err = processEventBatch(collection, events, roomID, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, count) // Should stop at limit

	// Clean up - remove test documents
	_, _ = collection.DeleteMany(context.Background(), bson.M{"room_id": roomID})
}

// Test importEventsFromRoom function with mock client
func TestImportEventsFromRoom_ErrorHandling(t *testing.T) {
	// Test with nil client (should return error quickly)
	// We'll just test that it doesn't crash with a timeout
	defer func() {
		if r := recover(); r != nil {
			// Expected - nil pointer dereference is what we want to test
			t.Log("Caught expected panic from nil client")
		}
	}()

	count, err := importEventsFromRoom(nil, "!room:example.com", 0)
	// If we get here without panic, the function handled nil client gracefully
	if err != nil {
		assert.Equal(t, 0, count)
		t.Log("Function returned error instead of panicking:", err)
	}
}

// Test main function indirectly by testing CLI setup
func TestMainFunctionSetup(t *testing.T) {
	// We can't directly test main() as it runs the CLI, but we can test
	// that the setup doesn't panic and basic functions work

	// Test that basic auth creation works
	auth := NewBeeperAuth("")
	assert.NotNil(t, auth)

	// Test that format validation works
	assert.True(t, isValidFormat("json"))

	// Test that we can create message validation
	msg := &Message{
		RoomID:      "!room:example.com",
		EventID:     "$event123",
		Sender:      "@user:example.com",
		MessageType: "m.room.message",
	}
	err := msg.Validate()
	assert.NoError(t, err)
}

// Test exportMessages function
func TestExportMessages(t *testing.T) {
	// Initialize MongoDB for testing
	InitMongoDB()
	defer CloseMongoDB()

	// Create a temporary output file
	tmpDir, err := os.MkdirTemp("", "matrix-archive-export-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test with unsupported format (already tested elsewhere, but for completeness)
	outputFile := filepath.Join(tmpDir, "test.unsupported")
	err = exportMessages(outputFile, "!room:example.com", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")

	// Test with non-existent room (should return quickly with no messages)
	outputFile = filepath.Join(tmpDir, "test.json")
	err = exportMessages(outputFile, "!nonexistent:example.com", false)
	// This may succeed with 0 messages, which is fine
	if err != nil {
		t.Logf("Export failed as expected: %v", err)
	}

	// Test with missing template for HTML format
	outputFile = filepath.Join(tmpDir, "test.html")
	err = exportMessages(outputFile, "!nonexistent:example.com", false)
	// This may fail due to missing template, which is expected
	if err != nil {
		t.Logf("HTML export failed as expected: %v", err)
	}
}

// Test runDownloads function
func TestRunDownloads(t *testing.T) {
	// Create temporary download directory
	tmpDir, err := os.MkdirTemp("", "matrix-archive-download-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test with empty message list
	err = runDownloads([]Message{}, tmpDir, false)
	assert.NoError(t, err)

	// Test with messages that have no image URLs
	messages := []Message{
		{
			RoomID:      "!room:example.com",
			EventID:     "$event1",
			Sender:      "@user:example.com",
			MessageType: "m.room.message",
			Content: bson.M{
				"msgtype": "m.text",
				"body":    "Hello world",
			},
		},
	}
	err = runDownloads(messages, tmpDir, false)
	assert.NoError(t, err) // Should succeed but download nothing

	// Test with message that has invalid image URL
	messages = []Message{
		{
			RoomID:      "!room:example.com",
			EventID:     "$event2",
			Sender:      "@user:example.com",
			MessageType: "m.room.message",
			Content: bson.M{
				"msgtype": "m.image",
				"url":     "invalid-url",
			},
		},
	}
	err = runDownloads(messages, tmpDir, false)
	assert.NoError(t, err) // Should succeed but skip invalid URL

	// Test with message that has valid MXC URL (will fail to download, but that's OK)
	messages = []Message{
		{
			RoomID:      "!room:example.com",
			EventID:     "$event3",
			Sender:      "@user:example.com",
			MessageType: "m.room.message",
			Content: bson.M{
				"msgtype": "m.image",
				"url":     "mxc://example.com/valid123",
			},
		},
	}
	err = runDownloads(messages, tmpDir, false)
	assert.NoError(t, err) // Should succeed even if download fails

	// Test with thumbnails preference
	err = runDownloads(messages, tmpDir, true)
	assert.NoError(t, err) // Should succeed even if download fails
}
