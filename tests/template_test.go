package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	archive "github.com/osteele/matrix-archive/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplateWithStringTimestamps tests that both templates work correctly with string timestamps
func TestTemplateWithStringTimestamps(t *testing.T) {
	// Create test messages with string timestamps
	testTimestamp := time.Now().Format(time.RFC3339)
	messages := []archive.ExportMessage{
		{
			Sender:    "testuser",
			Timestamp: testTimestamp,
			Content: map[string]interface{}{
				"msgtype": "m.text",
				"body":    "Test message 1",
			},
		},
		{
			Sender:    "anotheruser",
			Timestamp: testTimestamp,
			Content: map[string]interface{}{
				"msgtype": "m.image",
				"body":    "Test image caption",
				"url":     "https://example.com/image.jpg",
			},
		},
	}

	// Test HTML template
	t.Run("HTML Template", func(t *testing.T) {
		// Create temporary output file
		tmpDir, err := os.MkdirTemp("", "template_test_html")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		outputFile := filepath.Join(tmpDir, "test_output.html")
		file, err := os.Create(outputFile)
		require.NoError(t, err)
		defer file.Close()

		// Test the HTML template
		templatePath := filepath.Join("..", "templates", "default.html.tpl")
		err = archive.ExportWithTemplate(file, templatePath, messages)
		assert.NoError(t, err)

		// Close the file so we can read it
		file.Close()

		// Read the output and verify it contains formatted timestamps
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		contentStr := string(content)

		// Verify the content contains the expected elements
		assert.Contains(t, contentStr, "testuser")
		assert.Contains(t, contentStr, "anotheruser")
		assert.Contains(t, contentStr, "Test message 1")
		assert.Contains(t, contentStr, "Test image caption")
		
		// Verify that the timestamp is formatted correctly (not the raw RFC3339)
		// Should contain a human-readable date format, not RFC3339
		parsedTime, _ := time.Parse(time.RFC3339, testTimestamp)
		expectedFormat := parsedTime.Format("2006-01-02 15:04:05")
		assert.Contains(t, contentStr, expectedFormat)
		
		// Should NOT contain the raw RFC3339 format in the displayed time
		// RFC3339 format looks like "2025-08-27T10:00:18Z" with a "T" separator
		// Check that the timestamp in the output doesn't contain "T" between date and time
		assert.NotContains(t, contentStr, testTimestamp, "Should not contain raw RFC3339 timestamp in the display")
	})

	// Test TXT template
	t.Run("TXT Template", func(t *testing.T) {
		// Create temporary output file
		tmpDir, err := os.MkdirTemp("", "template_test_txt")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		outputFile := filepath.Join(tmpDir, "test_output.txt")
		file, err := os.Create(outputFile)
		require.NoError(t, err)
		defer file.Close()

		// Test the TXT template
		templatePath := filepath.Join("..", "templates", "default.txt.tpl")
		err = archive.ExportWithTemplate(file, templatePath, messages)
		assert.NoError(t, err)

		// Close the file so we can read it
		file.Close()

		// Read the output and verify it contains formatted timestamps
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		contentStr := string(content)

		// Verify the content contains the expected elements
		assert.Contains(t, contentStr, "From: testuser")
		assert.Contains(t, contentStr, "From: anotheruser")
		assert.Contains(t, contentStr, "Test message 1")
		assert.Contains(t, contentStr, "Caption: Test image caption")
		
		// Verify that the timestamp is formatted correctly
		parsedTime, _ := time.Parse(time.RFC3339, testTimestamp)
		expectedFormat := parsedTime.Format("2006-01-02 15:04:05")
		assert.Contains(t, contentStr, "Date: " + expectedFormat)
		
		// Should NOT contain the raw RFC3339 format in the displayed time
		// RFC3339 format looks like "2025-08-27T10:00:18Z" with a "T" separator
		// Check that the timestamp in the output doesn't contain "T" between date and time
		assert.NotContains(t, contentStr, testTimestamp, "Should not contain raw RFC3339 timestamp in the display")
	})
}

// TestTemplateFormatTimeFunction tests the formatTime function with edge cases
func TestTemplateFormatTimeFunction(t *testing.T) {
	messages := []archive.ExportMessage{
		{
			Sender:    "testuser",
			Timestamp: "", // Empty timestamp
			Content: map[string]interface{}{
				"msgtype": "m.text",
				"body":    "Test with empty timestamp",
			},
		},
		{
			Sender:    "testuser2",
			Timestamp: "invalid-timestamp", // Invalid timestamp
			Content: map[string]interface{}{
				"msgtype": "m.text", 
				"body":    "Test with invalid timestamp",
			},
		},
	}

	// Test with TXT template (simpler to parse)
	tmpDir, err := os.MkdirTemp("", "template_edge_cases")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	outputFile := filepath.Join(tmpDir, "test_edge_cases.txt")
	file, err := os.Create(outputFile)
	require.NoError(t, err)
	defer file.Close()

	templatePath := filepath.Join("..", "templates", "default.txt.tpl")
	err = archive.ExportWithTemplate(file, templatePath, messages)
	assert.NoError(t, err)

	file.Close()

	// Read the output and verify edge cases are handled
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	contentStr := string(content)

	// Should contain the test messages
	assert.Contains(t, contentStr, "Test with empty timestamp")
	assert.Contains(t, contentStr, "Test with invalid timestamp")
	
	// Should handle edge cases gracefully - empty timestamp should result in empty date
	// Invalid timestamp should be displayed as-is
	lines := strings.Split(contentStr, "\n")
	var emptyTimestampLine, invalidTimestampLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "Date: ") {
			if emptyTimestampLine == "" {
				emptyTimestampLine = line
			} else {
				invalidTimestampLine = line
			}
		}
	}
	
	// The first message has empty timestamp, should result in "Date: "
	assert.Equal(t, "Date: ", emptyTimestampLine)
	// The second message has invalid timestamp, should show the invalid string
	assert.Equal(t, "Date: invalid-timestamp", invalidTimestampLine)
}