package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	archive "github.com/osteele/matrix-archive/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplateTimestampHandlingEndToEnd tests the complete template functionality with string timestamps
func TestTemplateTimestampHandlingEndToEnd(t *testing.T) {
	// Create realistic test messages with different message types
	testTimestamp := "2025-08-27T15:30:45Z"
	messages := []archive.ExportMessage{
		{
			Sender:    "alice",
			Timestamp: testTimestamp,
			Content: map[string]interface{}{
				"msgtype": "m.text",
				"body":    "Hello, this is a text message!",
			},
		},
		{
			Sender:    "bob",
			Timestamp: testTimestamp,
			Content: map[string]interface{}{
				"msgtype": "m.image",
				"body":    "Beautiful sunset photo",
				"url":     "https://example.com/sunset.jpg",
			},
		},
		{
			Sender:    "charlie",
			Timestamp: testTimestamp,
			Content: map[string]interface{}{
				"msgtype": "m.notice",
				"body":    "System notice: Room settings updated",
			},
		},
		{
			Sender:    "dave",
			Timestamp: "", // Test empty timestamp
			Content: map[string]interface{}{
				"msgtype": "m.text",
				"body":    "Message with empty timestamp",
			},
		},
		{
			Sender:    "eve",
			Timestamp: "invalid-timestamp", // Test invalid timestamp
			Content: map[string]interface{}{
				"msgtype": "m.text",
				"body":    "Message with invalid timestamp",
			},
		},
	}

	t.Run("HTMLTemplateEndToEnd", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "html_e2e_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		outputFile := filepath.Join(tmpDir, "output.html")
		file, err := os.Create(outputFile)
		require.NoError(t, err)
		defer file.Close()

		templatePath := filepath.Join("..", "templates", "default.html.tpl")
		err = archive.ExportWithTemplate(file, templatePath, messages)
		require.NoError(t, err)

		file.Close()

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		contentStr := string(content)

		// Verify HTML structure
		assert.Contains(t, contentStr, "<!DOCTYPE html>")
		assert.Contains(t, contentStr, "<html")
		assert.Contains(t, contentStr, "</html>")
		assert.Contains(t, contentStr, "Matrix Room Archive")

		// Verify all users appear
		assert.Contains(t, contentStr, "alice")
		assert.Contains(t, contentStr, "bob") 
		assert.Contains(t, contentStr, "charlie")
		assert.Contains(t, contentStr, "dave")
		assert.Contains(t, contentStr, "eve")

		// Verify message content appears
		assert.Contains(t, contentStr, "Hello, this is a text message!")
		assert.Contains(t, contentStr, "Beautiful sunset photo")
		assert.Contains(t, contentStr, "System notice: Room settings updated")
		assert.Contains(t, contentStr, "Message with empty timestamp")
		assert.Contains(t, contentStr, "Message with invalid timestamp")

		// Verify timestamps are properly formatted (not RFC3339)
		expectedTime, _ := time.Parse(time.RFC3339, testTimestamp)
		expectedFormat := expectedTime.Format("2006-01-02 15:04:05")
		assert.Contains(t, contentStr, expectedFormat)
		
		// Verify raw RFC3339 timestamp doesn't appear in output
		assert.NotContains(t, contentStr, testTimestamp)

		// Verify edge cases handled properly
		assert.Contains(t, contentStr, "<span class=\"timestamp\"></span>") // Empty timestamp should result in empty span
		assert.Contains(t, contentStr, "<span class=\"timestamp\">invalid-timestamp</span>") // Invalid timestamp should appear as-is
	})

	t.Run("TXTTemplateEndToEnd", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "txt_e2e_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		outputFile := filepath.Join(tmpDir, "output.txt")
		file, err := os.Create(outputFile)
		require.NoError(t, err)
		defer file.Close()

		templatePath := filepath.Join("..", "templates", "default.txt.tpl")
		err = archive.ExportWithTemplate(file, templatePath, messages)
		require.NoError(t, err)

		file.Close()

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		contentStr := string(content)

		// Verify text structure
		assert.Contains(t, contentStr, "================================================================================")
		assert.Contains(t, contentStr, "From: alice")
		assert.Contains(t, contentStr, "From: bob")
		assert.Contains(t, contentStr, "From: charlie")
		assert.Contains(t, contentStr, "From: dave")
		assert.Contains(t, contentStr, "From: eve")

		// Verify message type indicators
		assert.Contains(t, contentStr, "Type: m.text")
		assert.Contains(t, contentStr, "Type: m.image")
		assert.Contains(t, contentStr, "Type: m.notice")

		// Verify timestamps are properly formatted
		expectedTime, _ := time.Parse(time.RFC3339, testTimestamp)
		expectedFormat := expectedTime.Format("2006-01-02 15:04:05")
		lines := strings.Split(contentStr, "\n")
		
		dateLines := []string{}
		for _, line := range lines {
			if strings.HasPrefix(line, "Date: ") {
				dateLines = append(dateLines, line)
			}
		}
		
		// Should have 5 Date lines (one for each message)
		assert.Len(t, dateLines, 5)
		
		// Check specific date formatting
		expectedDateLine := "Date: " + expectedFormat
		count := 0
		for _, line := range dateLines {
			if line == expectedDateLine {
				count++
			}
		}
		assert.Equal(t, 3, count) // 3 messages have the valid timestamp
		
		// Check edge cases
		assert.Contains(t, contentStr, "Date: ") // Empty timestamp case
		assert.Contains(t, contentStr, "Date: invalid-timestamp") // Invalid timestamp case
	})

	t.Run("JSONExportForComparison", func(t *testing.T) {
		// Test that the same messages can be exported to JSON correctly
		// This validates that the ExportMessage structure is sound
		tmpDir, err := os.MkdirTemp("", "json_comparison_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		outputFile := filepath.Join(tmpDir, "output.json")
		file, err := os.Create(outputFile)
		require.NoError(t, err)
		defer file.Close()

		// Manually create JSON to verify structure
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(messages)
		require.NoError(t, err)

		file.Close()

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		// Verify JSON structure is correct
		var decoded []archive.ExportMessage
		err = json.Unmarshal(content, &decoded)
		require.NoError(t, err)

		assert.Len(t, decoded, 5)
		assert.Equal(t, "alice", decoded[0].Sender)
		assert.Equal(t, testTimestamp, decoded[0].Timestamp) // Should be raw RFC3339 in JSON
		assert.Equal(t, "m.text", decoded[0].Content["msgtype"])
	})
}