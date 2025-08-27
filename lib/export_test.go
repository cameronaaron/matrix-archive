package archive

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExportWithTemplateInLib tests the ExportWithTemplate function directly in the lib package
func TestExportWithTemplateInLib(t *testing.T) {
	// Create test messages
	testTimestamp := time.Now().Format(time.RFC3339)
	messages := []ExportMessage{
		{
			Sender:    "testuser",
			Timestamp: testTimestamp,
			Content: map[string]interface{}{
				"msgtype": "m.text",
				"body":    "Test message",
			},
		},
	}

	t.Run("ValidHTMLTemplate", func(t *testing.T) {
		// Create temp file for output
		tmpDir, err := os.MkdirTemp("", "export_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		outputFile := filepath.Join(tmpDir, "output.html")
		file, err := os.Create(outputFile)
		require.NoError(t, err)
		defer file.Close()

		// Use the real HTML template
		templatePath := filepath.Join("../templates", "default.html.tpl")
		err = ExportWithTemplate(file, templatePath, messages)
		assert.NoError(t, err)
	})

	t.Run("ValidTXTTemplate", func(t *testing.T) {
		// Create temp file for output
		tmpDir, err := os.MkdirTemp("", "export_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		outputFile := filepath.Join(tmpDir, "output.txt")
		file, err := os.Create(outputFile)
		require.NoError(t, err)
		defer file.Close()

		// Use the real TXT template
		templatePath := filepath.Join("../templates", "default.txt.tpl")
		err = ExportWithTemplate(file, templatePath, messages)
		assert.NoError(t, err)
	})

	t.Run("InvalidTemplatePath", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "export_test_invalid")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		outputFile := filepath.Join(tmpDir, "output.html")
		file, err := os.Create(outputFile)
		require.NoError(t, err)
		defer file.Close()

		// Test with non-existent template
		err = ExportWithTemplate(file, "/nonexistent/template.tpl", messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read template")
	})

	t.Run("InvalidTemplateContent", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "export_test_invalid_content")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create template with invalid syntax
		invalidTemplate := filepath.Join(tmpDir, "invalid.tpl")
		err = os.WriteFile(invalidTemplate, []byte("{{.InvalidSyntax}"), 0644)
		require.NoError(t, err)

		outputFile := filepath.Join(tmpDir, "output.html")
		file, err := os.Create(outputFile)
		require.NoError(t, err)
		defer file.Close()

		// Test with template that has invalid syntax
		err = ExportWithTemplate(file, invalidTemplate, messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse template")
	})
}

// TestIsValidFormatInLib tests the format validation function in lib package
func TestIsValidFormatInLib(t *testing.T) {
	// Test valid formats
	assert.True(t, IsValidFormat("json"))
	assert.True(t, IsValidFormat("html"))
	assert.True(t, IsValidFormat("txt"))
	assert.True(t, IsValidFormat("yaml"))

	// Test invalid formats
	assert.False(t, IsValidFormat("pdf"))
	assert.False(t, IsValidFormat("xml"))
	assert.False(t, IsValidFormat(""))
	assert.False(t, IsValidFormat("unknown"))
}

// TestIsImageContentInLib tests the IsImageContent function in lib package
func TestIsImageContentInLib(t *testing.T) {
	// Test image content
	imageContent := map[string]interface{}{
		"msgtype": "m.image",
		"body":    "test.jpg",
		"url":     "mxc://example.com/test123",
	}
	assert.True(t, IsImageContent(imageContent))

	// Test non-image content
	textContent := map[string]interface{}{
		"msgtype": "m.text",
		"body":    "Hello world",
	}
	assert.False(t, IsImageContent(textContent))

	// Test empty content
	emptyContent := map[string]interface{}{}
	assert.False(t, IsImageContent(emptyContent))

	// Test nil content
	assert.False(t, IsImageContent(nil))
}

// TestConvertToDownloadURLs tests the convertToDownloadURLs function
func TestConvertToDownloadURLs(t *testing.T) {
	t.Run("ConvertMXCURL", func(t *testing.T) {
		content := map[string]interface{}{
			"msgtype": "m.image",
			"body":    "test.jpg",
			"url":     "mxc://example.com/test123",
		}

		result := convertToDownloadURLs(content)
		
		// Check that the result contains the expected download URL
		assert.Contains(t, result, "url")
		// The URL should be converted from mxc:// to download URL format
		url, ok := result["url"].(string)
		assert.True(t, ok)
		assert.Contains(t, url, "test123")
	})

	t.Run("NoMXCURL", func(t *testing.T) {
		content := map[string]interface{}{
			"msgtype": "m.text",
			"body":    "Hello world",
		}

		result := convertToDownloadURLs(content)
		
		// Should be unchanged if no mxc URLs
		assert.Equal(t, content, result)
	})
}

// TestConvertToLocalImages tests the convertToLocalImages function
func TestConvertToLocalImages(t *testing.T) {
	t.Run("ConvertMXCToLocal", func(t *testing.T) {
		content := map[string]interface{}{
			"msgtype": "m.image",
			"body":    "test.jpg",
			"url":     "mxc://example.com/test123",
		}

		result := convertToLocalImages(content)
		
		// Check that the result contains a local path
		assert.Contains(t, result, "url")
		url, ok := result["url"].(string)
		assert.True(t, ok)
		// Should be a local path, not mxc://
		assert.NotContains(t, url, "mxc://")
	})

	t.Run("NoMXCURL", func(t *testing.T) {
		content := map[string]interface{}{
			"msgtype": "m.text",
			"body":    "Hello world",
		}

		result := convertToLocalImages(content)
		
		// Should be unchanged if no mxc URLs
		assert.Equal(t, content, result)
	})
}

// TestConvertMXCToLocalPath tests the convertMXCToLocalPath function
func TestConvertMXCToLocalPath(t *testing.T) {
	t.Run("StandardMXC", func(t *testing.T) {
		mxcURL := "mxc://example.com/test123"
		content := map[string]interface{}{
			"body": "test.jpg",
		}

		result := convertMXCToLocalPath(mxcURL, content)
		
		// Should return a local path
		assert.NotEmpty(t, result)
		assert.NotContains(t, result, "mxc://")
		assert.Contains(t, result, "test123")
	})

	t.Run("InvalidMXC", func(t *testing.T) {
		mxcURL := "not-an-mxc-url"
		content := map[string]interface{}{
			"body": "test.jpg",
		}

		result := convertMXCToLocalPath(mxcURL, content)
		
		// Should still return a result, even if not a valid mxc URL
		assert.NotEmpty(t, result)
	})
}

// TestFormatTimeFunctionInLib tests the custom formatTime function through template execution in lib package
func TestFormatTimeFunctionInLib(t *testing.T) {
	testCases := []struct {
		name      string
		timestamp string
		expected  string
	}{
		{
			name:      "ValidRFC3339",
			timestamp: "2025-08-27T10:00:18Z",
			expected:  "2025-08-27 10:00:18",
		},
		{
			name:      "EmptyString",
			timestamp: "",
			expected:  "",
		},
		{
			name:      "InvalidFormat",
			timestamp: "invalid-timestamp",
			expected:  "invalid-timestamp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			messages := []ExportMessage{
				{
					Sender:    "testuser",
					Timestamp: tc.timestamp,
					Content: map[string]interface{}{
						"msgtype": "m.text",
						"body":    "Test",
					},
				},
			}

			tmpDir, err := os.MkdirTemp("", "format_time_test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			outputFile := filepath.Join(tmpDir, "output.txt")
			file, err := os.Create(outputFile)
			require.NoError(t, err)

			templatePath := filepath.Join("../templates", "default.txt.tpl")
			err = ExportWithTemplate(file, templatePath, messages)
			require.NoError(t, err)

			file.Close()

			content, err := os.ReadFile(outputFile)
			require.NoError(t, err)

			contentStr := string(content)
			assert.Contains(t, contentStr, "Date: " + tc.expected)
		})
	}
}