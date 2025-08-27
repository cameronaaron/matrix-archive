package archive

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var supportedFormats = []string{"txt", "html", "json", "yaml"}

// ExportMessage represents a message for export
type ExportMessage struct {
	Sender    string                 `json:"sender" yaml:"sender"`
	Timestamp string                 `json:"timestamp" yaml:"timestamp"`
	Content   map[string]interface{} `json:"content" yaml:"content"`
}

// exportMessages exports messages to a file in various formats
func ExportMessages(filename, roomID string, localImages bool) error {
	// Initialize database connection with DuckDB
	if err := InitDuckDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer CloseDatabase()

	// Determine format from file extension
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if ext == "" {
		ext = "html"
	}

	if !IsValidFormat(ext) {
		return fmt.Errorf("unsupported format %s, supported formats: %v", ext, supportedFormats)
	}

	// Determine room ID
	if roomID == "" {
		roomIDs, err := GetMatrixRoomIDs()
		if err != nil {
			return fmt.Errorf("failed to get room IDs: %w", err)
		}
		if len(roomIDs) == 0 {
			return fmt.Errorf("no room IDs configured")
		}
		roomID = roomIDs[0]
	} else if !strings.HasPrefix(roomID, "!") {
		// If roomID doesn't look like a room ID, try to find it by name
		foundRoomID, err := findRoomByName(roomID)
		if err != nil {
			return fmt.Errorf("failed to find room by name: %w", err)
		}
		roomID = foundRoomID
	}

	// Query messages from DuckDB
	filter := &MessageFilter{
		RoomID: roomID,
	}

	messages, err := GetDatabase().GetMessages(context.Background(), filter, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to query messages: %w", err)
	}

	fmt.Printf("Writing %d messages to %q\n", len(messages), filename)

	// Convert messages to export format
	exportMessages := make([]ExportMessage, len(messages))
	for i, msg := range messages {
		exportMsg, err := convertToExportMessage(*msg, localImages)
		if err != nil {
			return fmt.Errorf("failed to convert message: %w", err)
		}
		exportMessages[i] = exportMsg
	}

	// Export based on format
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	switch ext {
	case "json":
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		return encoder.Encode(exportMessages)

	case "yaml":
		encoder := yaml.NewEncoder(file)
		defer encoder.Close()
		return encoder.Encode(exportMessages)

	case "html":
		templatePath := "templates/default.html.tpl"
		return ExportWithTemplate(file, templatePath, exportMessages)

	case "txt":
		templatePath := "templates/default.txt.tpl"
		return ExportWithTemplate(file, templatePath, exportMessages)

	default:
		return fmt.Errorf("unsupported format: %s", ext)
	}
}

// convertToExportMessage converts a Message to ExportMessage
func convertToExportMessage(msg Message, localImages bool) (ExportMessage, error) {
	// Extract username from sender (@username:server.com -> username)
	senderRegex := regexp.MustCompile(`@(.+):.+`)
	sender := msg.Sender
	if matches := senderRegex.FindStringSubmatch(msg.Sender); len(matches) > 1 {
		sender = matches[1]
	}

	// Convert timestamp to ISO format
	timestamp := msg.Timestamp.Format(time.RFC3339)

	// Process content
	content := make(map[string]interface{})
	for k, v := range msg.Content {
		content[k] = v
	}

	// Handle image URLs
	if localImages {
		content = convertToLocalImages(content)
	} else {
		content = convertToDownloadURLs(content)
	}

	return ExportMessage{
		Sender:    sender,
		Timestamp: timestamp,
		Content:   content,
	}, nil
}

// convertToDownloadURLs converts mxc URLs to download URLs
func convertToDownloadURLs(content map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range content {
		if k == "url" {
			if urlStr, ok := v.(string); ok {
				if strings.HasPrefix(urlStr, "mxc://") {
					if downloadURL, err := GetDownloadURL(urlStr); err == nil {
						result[k] = downloadURL
					} else {
						result[k] = urlStr
					}
				} else {
					result[k] = urlStr
				}
			} else {
				result[k] = v
			}
		} else if subMap, ok := v.(map[string]interface{}); ok {
			result[k] = convertToDownloadURLs(subMap)
		} else {
			result[k] = v
		}
	}
	return result
}

// convertToLocalImages converts image URLs to local file paths
func convertToLocalImages(content map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range content {
		if k == "url" && IsImageContent(content) {
			if urlStr, ok := v.(string); ok {
				result[k] = convertMXCToLocalPath(urlStr, content)
			} else {
				result[k] = v
			}
		} else if subMap, ok := v.(map[string]interface{}); ok {
			result[k] = convertToLocalImages(subMap)
		} else {
			result[k] = v
		}
	}
	return result
}

// isImageContent checks if content represents an image message
func IsImageContent(content map[string]interface{}) bool {
	if msgtype, ok := content["msgtype"].(string); ok {
		return msgtype == "m.image"
	}
	return false
}

// convertMXCToLocalPath converts an mxc URL to a local file path
func convertMXCToLocalPath(mxcURL string, content map[string]interface{}) string {
	if !strings.HasPrefix(mxcURL, "mxc://") {
		return mxcURL
	}

	// Extract path from mxc URL
	parts := strings.Split(strings.TrimPrefix(mxcURL, "mxc://"), "/")
	if len(parts) < 2 {
		return mxcURL
	}

	// Get file extension from mimetype
	ext := ""
	if info, ok := content["info"].(map[string]interface{}); ok {
		if mimetype, ok := info["mimetype"].(string); ok {
			if parts := strings.Split(mimetype, "/"); len(parts) == 2 {
				ext = "." + parts[1]
			}
		}
	}

	// Use thumbnail if available
	if info, ok := content["info"].(map[string]interface{}); ok {
		if thumbURL, ok := info["thumbnail_url"].(string); ok && thumbURL != "" {
			if strings.HasPrefix(thumbURL, "mxc://") {
				thumbParts := strings.Split(strings.TrimPrefix(thumbURL, "mxc://"), "/")
				if len(thumbParts) >= 2 {
					return "thumbnails/" + strings.Join(thumbParts[1:], "/") + ext
				}
			}
		}
	}

	return "thumbnails/" + strings.Join(parts[1:], "/") + ext
}

// exportWithTemplate exports messages using a template
func ExportWithTemplate(file *os.File, templatePath string, messages []ExportMessage) error {
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	// Create template with custom functions
	funcMap := template.FuncMap{
		"formatTime": func(timeStr string) string {
			if timeStr == "" {
				return ""
			}
			// Parse RFC3339 timestamp string
			t, err := time.Parse(time.RFC3339, timeStr)
			if err != nil {
				// If parsing fails, return the original string
				return timeStr
			}
			// Format it in a human-readable format
			return t.Format("2006-01-02 15:04:05")
		},
	}

	tmpl, err := template.New("export").Funcs(funcMap).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := map[string]interface{}{
		"messages": messages,
	}

	return tmpl.Execute(file, data)
}

// findRoomByName finds a room ID by display name
func findRoomByName(roomName string) (string, error) {
	client, err := GetMatrixClient()
	if err != nil {
		return "", fmt.Errorf("failed to get Matrix client: %w", err)
	}

	resp, err := client.JoinedRooms(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get joined rooms: %w", err)
	}

	for _, roomID := range resp.JoinedRooms {
		displayName, err := GetRoomDisplayName(client, string(roomID))
		if err != nil {
			continue
		}

		if strings.Contains(displayName, roomName) {
			return string(roomID), nil
		}
	}

	return "", fmt.Errorf("room not found: %s", roomName)
}

// isValidFormat checks if a format is supported
func IsValidFormat(format string) bool {
	for _, f := range supportedFormats {
		if f == format {
			return true
		}
	}
	return false
}
