package archive

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

var supportedFormats = []string{"txt", "html", "json", "yaml"}

// ExportMessage represents a message for export with rich metadata
type ExportMessage struct {
	Sender      string                 `json:"sender" yaml:"sender"`
	DisplayName string                 `json:"display_name" yaml:"display_name"`
	UserID      string                 `json:"user_id" yaml:"user_id"`
	Timestamp   string                 `json:"timestamp" yaml:"timestamp"`
	Content     map[string]interface{} `json:"content" yaml:"content"`
	EventID     string                 `json:"event_id" yaml:"event_id"`
	MessageType string                 `json:"message_type" yaml:"message_type"`
	
	// Rich metadata
	Reactions   []MessageReaction `json:"reactions,omitempty" yaml:"reactions,omitempty"`
	RepliesTo   *ReplyInfo        `json:"replies_to,omitempty" yaml:"replies_to,omitempty"`
	IsEdited    bool              `json:"is_edited,omitempty" yaml:"is_edited,omitempty"`
	EditHistory []EditInfo        `json:"edit_history,omitempty" yaml:"edit_history,omitempty"`
	ThreadInfo  *ThreadInfo       `json:"thread_info,omitempty" yaml:"thread_info,omitempty"`
	UserAvatar  string            `json:"user_avatar,omitempty" yaml:"user_avatar,omitempty"`
	Platform    string            `json:"platform,omitempty" yaml:"platform,omitempty"`
}

// MessageReaction represents a reaction to a message
type MessageReaction struct {
	Emoji     string    `json:"emoji"`
	Users     []string  `json:"users"`
	Count     int       `json:"count"`
	EventID   string    `json:"event_id"`
	Timestamp time.Time `json:"timestamp"`
}

// ReplyInfo represents reply relationship information
type ReplyInfo struct {
	EventID     string `json:"event_id"`
	Sender      string `json:"sender"`
	DisplayName string `json:"display_name"`
	Content     string `json:"content"`
	Timestamp   string `json:"timestamp"`
}

// EditInfo represents message edit information
type EditInfo struct {
	EventID     string    `json:"event_id"`
	Timestamp   time.Time `json:"timestamp"`
	PrevContent string    `json:"previous_content"`
	NewContent  string    `json:"new_content"`
}

// ThreadInfo represents thread/conversation information
type ThreadInfo struct {
	RootEventID string `json:"root_event_id"`
	ReplyCount  int    `json:"reply_count"`
	IsRoot      bool   `json:"is_root"`
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
		// Get all rooms from database
		db := GetDatabase()
		rooms, err := db.GetRooms(context.Background())
		if err != nil {
			return fmt.Errorf("failed to get rooms from database: %w", err)
		}
		if len(rooms) == 0 {
			return fmt.Errorf("no rooms found in database")
		}
		roomID = rooms[0]
		fmt.Printf("No room ID specified, using first room found: %s\n", roomID)
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

	// If no messages found in database, automatically import them first
	if len(messages) == 0 {
		fmt.Printf("No messages found in database for room %s. Importing messages...\n", roomID)

		// Import messages from Matrix into the database
		// Note: Don't close the database here since we're still using it
		err := ImportMessagesFromSpecificRoomWithoutClosing(roomID, 0) // 0 = no limit
		if err != nil {
			return fmt.Errorf("failed to import messages: %w", err)
		}

		// Query again after import
		messages, err = GetDatabase().GetMessages(context.Background(), filter, 0, 0)
		if err != nil {
			return fmt.Errorf("failed to query messages after import: %w", err)
		}
	}

	fmt.Printf("Writing %d messages to %q\n", len(messages), filename)

	// Convert messages to export format with enhanced user information
	exportMessages, err := convertToExportMessages(messages, roomID, localImages)
	if err != nil {
		return fmt.Errorf("failed to convert messages: %w", err)
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

// convertToExportMessages converts messages to export format with enhanced user information
func convertToExportMessages(messages []*Message, roomID string, localImages bool) ([]ExportMessage, error) {
	if len(messages) == 0 {
		return []ExportMessage{}, nil
	}

	// Build a mapping of bridge IDs to real usernames from message content
	bridgeUserMap := buildBridgeUserMapping(messages)

	// Get Matrix client to query user information
	client, err := GetMatrixClient()
	if err != nil {
		log.Printf("Warning: Could not get Matrix client for user info: %v", err)
		// Fall back to basic conversion without display names
		return convertToExportMessagesWithBridgeMapping(messages, localImages, bridgeUserMap)
	}

	// Create a cache for user display names to avoid repeated API calls
	userDisplayNameCache := make(map[string]string)
	
	exportMessages := make([]ExportMessage, len(messages))
	
	for i, msg := range messages {
		// Get display name for the user - try bridge mapping first
		displayName := getUserDisplayName(client, roomID, msg.Sender, userDisplayNameCache)
		
		// If we have a real username from bridge mapping, use that instead
		if realUsername, exists := bridgeUserMap[msg.Sender]; exists {
			displayName = realUsername
		}
		
		// Extract username from sender (@username:server.com -> username)
		senderRegex := regexp.MustCompile(`@(.+):.+`)
		username := msg.Sender
		if matches := senderRegex.FindStringSubmatch(msg.Sender); len(matches) > 1 {
			username = matches[1]
		}

		// Use real username if available
		if realUsername, exists := bridgeUserMap[msg.Sender]; exists {
			username = realUsername
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

		exportMessages[i] = ExportMessage{
			Sender:      username,
			DisplayName: displayName,
			UserID:      msg.Sender,
			Timestamp:   timestamp,
			Content:     content,
			EventID:     msg.EventID,
			MessageType: msg.MessageType,
		}
	}

	return exportMessages, nil
}

// BridgeUserCorrelation stores correlation data for bridge users
type BridgeUserCorrelation struct {
	username   string
	platform   string
	confidence float64
	timestamp  time.Time
	context    string
}

// buildBridgeUserMapping extracts real usernames from message content patterns with sophisticated correlation
func buildBridgeUserMapping(messages []*Message) map[string]string {
	bridgeUserMap := make(map[string]string)
	
	// Keep track of all correlations for each bridge user
	bridgeCorrelations := make(map[string][]BridgeUserCorrelation)
	
	// Comprehensive regex patterns to match Discord/Telegram usernames in message content
	usernameRegex := regexp.MustCompile(`<([^@][^>]+):(discord|telegram)>`)
	htmlUsernameRegex := regexp.MustCompile(`&lt;([^@][^&]+):(discord|telegram)&gt;`)
	
	// Enhanced patterns for bridge bot replies and mentions
	bridgeReplyRegex := regexp.MustCompile(`\(re @GrapheneOSBridgeBot: <([^@][^>]+):(discord|telegram)>`)
	htmlBridgeReplyRegex := regexp.MustCompile(`\(re @GrapheneOSBridgeBot: &lt;([^@][^&]+):(discord|telegram)&gt;`)
	
	// Additional patterns for various mention formats
	hrefUsernameRegex := regexp.MustCompile(`<a href="([^@][^"]+):(discord|telegram)">([^<]+)</a>`)
	bridgeMentionRegex := regexp.MustCompile(`@GrapheneOSBridgeBot:\s*<([^@][^>]+):(discord|telegram)>`)
	htmlBridgeMentionRegex := regexp.MustCompile(`@GrapheneOSBridgeBot:\s*&lt;([^@][^&]+):(discord|telegram)&gt;`)
	bridgeReplacementRegex := regexp.MustCompile(`@GrapheneOSBridgeBot:\s*([^:\s]+):(discord|telegram)`)
	multiplePlatformRegex := regexp.MustCompile(`([^@\s]+):(discord|telegram|matrix\.org)`)
	
	// Track usernames to Discord IDs for cross-correlation
	discordUsernameToIDs := make(map[string][]string)
	
	// First pass: collect all username correlations with context and timing using comprehensive patterns
	for _, msg := range messages {
		var textToScan []string
		
		// Collect text content to scan
		if bodyInterface, exists := msg.Content["body"]; exists {
			if body, ok := bodyInterface.(string); ok {
				textToScan = append(textToScan, body)
			}
		}
		
		if formattedBodyInterface, exists := msg.Content["formatted_body"]; exists {
			if formattedBody, ok := formattedBodyInterface.(string); ok {
				textToScan = append(textToScan, formattedBody)
			}
		}
		
		// Scan all text content for comprehensive username patterns
		for _, text := range textToScan {
			// Pattern 1: Direct mentions of usernames in messages (high confidence for bridge users)
			if strings.Contains(msg.Sender, "discordgo_") {
				matches := usernameRegex.FindAllStringSubmatch(text, -1)
				for _, match := range matches {
					if len(match) >= 3 {
						username := match[1]
						platform := match[2]
						
						if platform == "discord" {
							correlation := BridgeUserCorrelation{
								username:   username,
								platform:   platform,
								confidence: 0.8, // High confidence for self-mentions
								timestamp:  msg.Timestamp,
								context:    "self-mention",
							}
							bridgeCorrelations[msg.Sender] = append(bridgeCorrelations[msg.Sender], correlation)
							discordUsernameToIDs[username] = append(discordUsernameToIDs[username], msg.Sender)
						}
					}
				}
				
				// HTML-encoded patterns
				matches = htmlUsernameRegex.FindAllStringSubmatch(text, -1)
				for _, match := range matches {
					if len(match) >= 3 {
						username := match[1]
						platform := match[2]
						
						if platform == "discord" {
							correlation := BridgeUserCorrelation{
								username:   username,
								platform:   platform,
								confidence: 0.8,
								timestamp:  msg.Timestamp,
								context:    "self-mention-html",
							}
							bridgeCorrelations[msg.Sender] = append(bridgeCorrelations[msg.Sender], correlation)
							discordUsernameToIDs[username] = append(discordUsernameToIDs[username], msg.Sender)
						}
					}
				}
			}
			
			// Pattern 2: Href link patterns from formatted messages
			matches := hrefUsernameRegex.FindAllStringSubmatch(text, -1)
			for _, match := range matches {
				if len(match) >= 4 {
					username := match[1]
					platform := match[2]
					
					if platform == "discord" && strings.Contains(msg.Sender, "discordgo_") {
						correlation := BridgeUserCorrelation{
							username:   username,
							platform:   platform,
							confidence: 0.7,
							timestamp:  msg.Timestamp,
							context:    "href-link",
						}
						bridgeCorrelations[msg.Sender] = append(bridgeCorrelations[msg.Sender], correlation)
						discordUsernameToIDs[username] = append(discordUsernameToIDs[username], msg.Sender)
					}
				}
			}
			
			// Pattern 3: Multi-platform pattern matching
			matches = multiplePlatformRegex.FindAllStringSubmatch(text, -1)
			for _, match := range matches {
				if len(match) >= 3 {
					username := match[1]
					platform := match[2]
					
					if platform == "discord" && strings.Contains(msg.Sender, "discordgo_") {
						correlation := BridgeUserCorrelation{
							username:   username,
							platform:   platform,
							confidence: 0.6,
							timestamp:  msg.Timestamp,
							context:    "multi-platform",
						}
						bridgeCorrelations[msg.Sender] = append(bridgeCorrelations[msg.Sender], correlation)
						discordUsernameToIDs[username] = append(discordUsernameToIDs[username], msg.Sender)
					}
				}
			}
		}
	}
	
	// Second pass: analyze reply patterns and temporal proximity for high-confidence mapping
	for i, msg := range messages {
		// Look for bridge bot replies that mention usernames
		if bodyInterface, exists := msg.Content["body"]; exists {
			if body, ok := bodyInterface.(string); ok {
				// Bridge bot reply patterns
				matches := bridgeReplyRegex.FindAllStringSubmatch(body, -1)
				for _, match := range matches {
					if len(match) >= 3 {
						username := match[1]
						platform := match[2]
						
						if platform == "discord" {
							// Look for bridge users in nearby messages (within 10 messages for better coverage)
							for j := max(0, i-10); j <= min(len(messages)-1, i+10); j++ {
								nearbyMsg := messages[j]
								if strings.Contains(nearbyMsg.Sender, "discordgo_") {
									// Calculate distance-based confidence
									distance := abs(i - j)
									confidence := 1.0 - (float64(distance) * 0.05) // Decrease confidence with distance
									if confidence < 0.3 {
										confidence = 0.3
									}
									
									correlation := BridgeUserCorrelation{
										username:   username,
										platform:   platform,
										confidence: confidence,
										timestamp:  nearbyMsg.Timestamp,
										context:    fmt.Sprintf("bridge-reply-proximity-dist-%d", distance),
									}
									bridgeCorrelations[nearbyMsg.Sender] = append(bridgeCorrelations[nearbyMsg.Sender], correlation)
									discordUsernameToIDs[username] = append(discordUsernameToIDs[username], nearbyMsg.Sender)
								}
							}
						}
					}
				}
				
				// HTML bridge reply patterns
				matches = htmlBridgeReplyRegex.FindAllStringSubmatch(body, -1)
				for _, match := range matches {
					if len(match) >= 3 {
						username := match[1]
						platform := match[2]
						
						if platform == "discord" {
							// Look for bridge users in nearby messages
							for j := max(0, i-10); j <= min(len(messages)-1, i+10); j++ {
								nearbyMsg := messages[j]
								if strings.Contains(nearbyMsg.Sender, "discordgo_") {
									distance := abs(i - j)
									confidence := 1.0 - (float64(distance) * 0.05)
									if confidence < 0.3 {
										confidence = 0.3
									}
									
									correlation := BridgeUserCorrelation{
										username:   username,
										platform:   platform,
										confidence: confidence,
										timestamp:  nearbyMsg.Timestamp,
										context:    fmt.Sprintf("bridge-reply-proximity-html-dist-%d", distance),
									}
									bridgeCorrelations[nearbyMsg.Sender] = append(bridgeCorrelations[nearbyMsg.Sender], correlation)
									discordUsernameToIDs[username] = append(discordUsernameToIDs[username], nearbyMsg.Sender)
								}
							}
						}
					}
				}
				
				// Bridge bot direct mentions
				matches = bridgeMentionRegex.FindAllStringSubmatch(body, -1)
				for _, match := range matches {
					if len(match) >= 3 {
						username := match[1]
						platform := match[2]
						
						if platform == "discord" {
							// Associate with sender if it's a bridge user
							if strings.Contains(msg.Sender, "discordgo_") {
								correlation := BridgeUserCorrelation{
									username:   username,
									platform:   platform,
									confidence: 0.9,
									timestamp:  msg.Timestamp,
									context:    "bridge-mention",
								}
								bridgeCorrelations[msg.Sender] = append(bridgeCorrelations[msg.Sender], correlation)
								discordUsernameToIDs[username] = append(discordUsernameToIDs[username], msg.Sender)
							}
						}
					}
				}
				
				// HTML bridge bot mentions
				matches = htmlBridgeMentionRegex.FindAllStringSubmatch(body, -1)
				for _, match := range matches {
					if len(match) >= 3 {
						username := match[1]
						platform := match[2]
						
						if platform == "discord" && strings.Contains(msg.Sender, "discordgo_") {
							correlation := BridgeUserCorrelation{
								username:   username,
								platform:   platform,
								confidence: 0.9,
								timestamp:  msg.Timestamp,
								context:    "bridge-mention-html",
							}
							bridgeCorrelations[msg.Sender] = append(bridgeCorrelations[msg.Sender], correlation)
							discordUsernameToIDs[username] = append(discordUsernameToIDs[username], msg.Sender)
						}
					}
				}
				
				// Bridge replacement patterns
				matches = bridgeReplacementRegex.FindAllStringSubmatch(body, -1)
				for _, match := range matches {
					if len(match) >= 3 {
						username := match[1]
						platform := match[2]
						
						if platform == "discord" && strings.Contains(msg.Sender, "discordgo_") {
							correlation := BridgeUserCorrelation{
								username:   username,
								platform:   platform,
								confidence: 0.7,
								timestamp:  msg.Timestamp,
								context:    "bridge-replacement",
							}
							bridgeCorrelations[msg.Sender] = append(bridgeCorrelations[msg.Sender], correlation)
							discordUsernameToIDs[username] = append(discordUsernameToIDs[username], msg.Sender)
						}
					}
				}
			}
		}
	}
	
	// Third pass: find direct username frequency in bridge user messages
	bridgeUserCounts := make(map[string]map[string]int)
	
	for _, msg := range messages {
		if !strings.Contains(msg.Sender, "discordgo_") {
			continue
		}
		
		var textToScan []string
		
		if bodyInterface, exists := msg.Content["body"]; exists {
			if body, ok := bodyInterface.(string); ok {
				textToScan = append(textToScan, body)
			}
		}
		
		if formattedBodyInterface, exists := msg.Content["formatted_body"]; exists {
			if formattedBody, ok := formattedBodyInterface.(string); ok {
				textToScan = append(textToScan, formattedBody)
			}
		}
		
		// Look for username patterns in this bridge user's messages
		for _, text := range textToScan {
			matches := usernameRegex.FindAllStringSubmatch(text, -1)
			for _, match := range matches {
				if len(match) >= 3 {
					username := match[1]
					platform := match[2]
					
					if platform == "discord" {
						if bridgeUserCounts[msg.Sender] == nil {
							bridgeUserCounts[msg.Sender] = make(map[string]int)
						}
						bridgeUserCounts[msg.Sender][username]++
					}
				}
			}
			
			matches = htmlUsernameRegex.FindAllStringSubmatch(text, -1)
			for _, match := range matches {
				if len(match) >= 3 {
					username := match[1]
					platform := match[2]
					
					if platform == "discord" {
						if bridgeUserCounts[msg.Sender] == nil {
							bridgeUserCounts[msg.Sender] = make(map[string]int)
						}
						bridgeUserCounts[msg.Sender][username]++
					}
				}
			}
		}
	}
	
	// Analyze all correlations and determine best username for each bridge user
	for bridgeID, correlations := range bridgeCorrelations {
		usernameConfidence := make(map[string]float64)
		
		// Calculate weighted confidence scores
		for _, correlation := range correlations {
			usernameConfidence[correlation.username] += correlation.confidence
		}
		
		// Find the username with highest confidence
		bestUsername := ""
		maxConfidence := 0.0
		
		for username, confidence := range usernameConfidence {
			if confidence > maxConfidence {
				maxConfidence = confidence
				bestUsername = username
			}
		}
		
		if bestUsername != "" {
			bridgeUserMap[bridgeID] = bestUsername
			log.Printf("Mapped %s -> %s (confidence: %.2f)", bridgeID, bestUsername, maxConfidence)
		}
	}
	
	// Fallback: map each bridge user to their most commonly mentioned username from frequency analysis
	
	for _, msg := range messages {
		if !strings.Contains(msg.Sender, "discordgo_") {
			continue
		}
		
		var textToScan []string
		
		if bodyInterface, exists := msg.Content["body"]; exists {
			if body, ok := bodyInterface.(string); ok {
				textToScan = append(textToScan, body)
			}
		}
		
		if formattedBodyInterface, exists := msg.Content["formatted_body"]; exists {
			if formattedBody, ok := formattedBodyInterface.(string); ok {
				textToScan = append(textToScan, formattedBody)
			}
		}
		
		// Look for username patterns in this bridge user's messages
		for _, text := range textToScan {
			matches := usernameRegex.FindAllStringSubmatch(text, -1)
			for _, match := range matches {
				if len(match) >= 3 {
					username := match[1]
					platform := match[2]
					
					if platform == "discord" {
						if bridgeUserCounts[msg.Sender] == nil {
							bridgeUserCounts[msg.Sender] = make(map[string]int)
						}
						bridgeUserCounts[msg.Sender][username]++
					}
				}
			}
			
			matches = htmlUsernameRegex.FindAllStringSubmatch(text, -1)
			for _, match := range matches {
				if len(match) >= 3 {
					username := match[1]
					platform := match[2]
					
					if platform == "discord" {
						if bridgeUserCounts[msg.Sender] == nil {
							bridgeUserCounts[msg.Sender] = make(map[string]int)
						}
						bridgeUserCounts[msg.Sender][username]++
					}
				}
			}
		}
	}
	
	// Map each bridge user to their most commonly mentioned username
	for bridgeID, usernameCounts := range bridgeUserCounts {
		// Skip if we already have a high-confidence mapping
		if _, exists := bridgeUserMap[bridgeID]; exists {
			continue
		}
		
		maxCount := 0
		bestUsername := ""
		
		for username, count := range usernameCounts {
			if count > maxCount {
				maxCount = count
				bestUsername = username
			}
		}
		
		if bestUsername != "" {
			bridgeUserMap[bridgeID] = bestUsername
			log.Printf("Mapped %s -> %s (frequency: %d)", bridgeID, bestUsername, maxCount)
		}
	}
	
	// Count unique Discord usernames found
	uniqueUsernames := make(map[string]bool)
	for _, correlations := range bridgeCorrelations {
		for _, correlation := range correlations {
			if correlation.platform == "discord" {
				uniqueUsernames[correlation.username] = true
			}
		}
	}
	for _, usernameCounts := range bridgeUserCounts {
		for username := range usernameCounts {
			uniqueUsernames[username] = true
		}
	}
	
	// Alternative approach: look for messages from GrapheneOSBridgeBot that mention users
	for _, msg := range messages {
		if strings.Contains(msg.Sender, "grapheneosbridge") {
			var textToScan []string
			
			if bodyInterface, exists := msg.Content["body"]; exists {
				if body, ok := bodyInterface.(string); ok {
					textToScan = append(textToScan, body)
				}
			}
			
			if formattedBodyInterface, exists := msg.Content["formatted_body"]; exists {
				if formattedBody, ok := formattedBodyInterface.(string); ok {
					textToScan = append(textToScan, formattedBody)
				}
			}
			
			for _, text := range textToScan {
				// Look for patterns like "<username:discord> message content"
				if strings.Contains(text, ":discord>") || strings.Contains(text, ":telegram>") {
					matches := usernameRegex.FindAllStringSubmatch(text, -1)
					for _, match := range matches {
						if len(match) >= 3 {
							username := match[1]
							platform := match[2]
							
							if platform == "discord" {
								// This message from the bridge bot contains this username
								// We could try to associate it with a bridge ID, but this is complex
								log.Printf("Bridge bot mentioned user: %s:%s", username, platform)
							}
						}
					}
					
					matches = htmlUsernameRegex.FindAllStringSubmatch(text, -1)
					for _, match := range matches {
						if len(match) >= 3 {
							username := match[1]
							platform := match[2]
							
							if platform == "discord" {
								log.Printf("Bridge bot mentioned user (HTML): %s:%s", username, platform)
							}
						}
					}
				}
			}
		}
	}
	
	log.Printf("Built bridge user mapping for %d users from %d Discord usernames found", len(bridgeUserMap), len(uniqueUsernames))
	for bridgeID, realName := range bridgeUserMap {
		log.Printf("  %s -> %s", bridgeID, realName)
	}
	
	return bridgeUserMap
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// convertToExportMessagesWithBridgeMapping converts messages with bridge user mapping fallback
func convertToExportMessagesWithBridgeMapping(messages []*Message, localImages bool, bridgeUserMap map[string]string) ([]ExportMessage, error) {
	exportMessages := make([]ExportMessage, len(messages))
	
	for i, msg := range messages {
		// Extract username from sender (@username:server.com -> username)
		senderRegex := regexp.MustCompile(`@(.+):.+`)
		username := msg.Sender
		if matches := senderRegex.FindStringSubmatch(msg.Sender); len(matches) > 1 {
			username = matches[1]
		}

		// Use real username if available from bridge mapping
		displayName := username
		if realUsername, exists := bridgeUserMap[msg.Sender]; exists {
			username = realUsername
			displayName = realUsername
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

		exportMessages[i] = ExportMessage{
			Sender:      username,
			DisplayName: displayName,
			UserID:      msg.Sender,
			Timestamp:   timestamp,
			Content:     content,
			EventID:     msg.EventID,
			MessageType: msg.MessageType,
		}
	}

	return exportMessages, nil
}

// convertToExportMessagesBasic converts messages without enhanced user info (fallback)
func convertToExportMessagesBasic(messages []*Message, localImages bool) ([]ExportMessage, error) {
	// Build bridge user mapping even in basic mode to get real usernames
	bridgeUserMap := buildBridgeUserMapping(messages)
	
	exportMessages := make([]ExportMessage, len(messages))
	
	for i, msg := range messages {
		// Extract username from sender (@username:server.com -> username)
		senderRegex := regexp.MustCompile(`@(.+):.+`)
		username := msg.Sender
		if matches := senderRegex.FindStringSubmatch(msg.Sender); len(matches) > 1 {
			username = matches[1]
		}

		// Use real username if available from bridge mapping
		displayName := username
		if realUsername, exists := bridgeUserMap[msg.Sender]; exists {
			username = realUsername
			displayName = realUsername
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

		exportMessages[i] = ExportMessage{
			Sender:      username,
			DisplayName: displayName, // Use extracted display name
			UserID:      msg.Sender,
			Timestamp:   timestamp,
			Content:     content,
			EventID:     msg.EventID,
			MessageType: msg.MessageType,
		}
	}
	
	return exportMessages, nil
}

// getUserDisplayName gets the display name for a user in a room
func getUserDisplayName(client *mautrix.Client, roomID, userID string, cache map[string]string) string {
	// Check cache first
	if displayName, exists := cache[userID]; exists {
		return displayName
	}
	
	// Default fallback
	senderRegex := regexp.MustCompile(`@(.+):.+`)
	defaultName := userID
	if matches := senderRegex.FindStringSubmatch(userID); len(matches) > 1 {
		defaultName = matches[1]
	}
	
	// Try to get user display name from room member state
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Query the room member state for this user
	stateKey := userID
	var memberEvent event.Event
	err := client.StateEvent(ctx, id.RoomID(roomID), event.StateMember, stateKey, &memberEvent)
	if err != nil {
		log.Printf("Warning: Could not get member info for %s in room %s: %v", userID, roomID, err)
		cache[userID] = defaultName
		return defaultName
	}
	
	// Extract display name from member event
	if memberEvent.Content.Raw != nil {
		if displayNameRaw, exists := memberEvent.Content.Raw["displayname"]; exists {
			if displayName, ok := displayNameRaw.(string); ok && displayName != "" {
				cache[userID] = displayName
				return displayName
			}
		}
	}
	
	// Cache and return the default name
	cache[userID] = defaultName
	return defaultName
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
			return t.Format("January 2, 2006 at 3:04 PM")
		},
		"now": func() string {
			return time.Now().Format(time.RFC3339)
		},
		"substr": func(s string, start, length int) string {
			if start < 0 || start >= len(s) {
				return ""
			}
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"countUniqueUsers": func(messages []ExportMessage) int {
			users := make(map[string]bool)
			for _, msg := range messages {
				users[msg.DisplayName] = true
			}
			return len(users)
		},
		"countPlatforms": func(messages []ExportMessage) int {
			platforms := make(map[string]bool)
			for _, msg := range messages {
				if msg.Platform != "" {
					platforms[msg.Platform] = true
				}
			}
			return len(platforms)
		},
		"countReactions": func(messages []ExportMessage) int {
			total := 0
			for _, msg := range messages {
				total += len(msg.Reactions)
			}
			return total
		},
		"countBridgeUsers": func(messages []ExportMessage) int {
			bridgeUsers := make(map[string]bool)
			for _, msg := range messages {
				if strings.Contains(msg.UserID, "discordgo_") {
					bridgeUsers[msg.DisplayName] = true
				}
			}
			return len(bridgeUsers)
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
	}

	tmpl, err := template.New("export").Funcs(funcMap).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Pass messages directly to template (not wrapped in a map)
	return tmpl.Execute(file, messages)
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

// extractReactions finds all reactions to a specific message
func extractReactions(messages []*Message, eventID string) []MessageReaction {
	var reactions []MessageReaction
	reactionMap := make(map[string]*MessageReaction)
	
	for _, msg := range messages {
		if relatesTo, exists := msg.Content["m.relates_to"]; exists {
			if relatesMap, ok := relatesTo.(map[string]interface{}); ok {
				if relType, exists := relatesMap["rel_type"]; exists && relType == "m.annotation" {
					if targetEvent, exists := relatesMap["event_id"]; exists && targetEvent == eventID {
						if emoji, exists := relatesMap["key"]; exists {
							if emojiStr, ok := emoji.(string); ok {
								if existing, found := reactionMap[emojiStr]; found {
									existing.Users = append(existing.Users, msg.Sender)
									existing.Count++
								} else {
									reaction := &MessageReaction{
										Emoji:     emojiStr,
										Users:     []string{msg.Sender},
										Count:     1,
										EventID:   msg.EventID,
										Timestamp: msg.Timestamp,
									}
									reactionMap[emojiStr] = reaction
								}
							}
						}
					}
				}
			}
		}
	}
	
	// Convert map to slice
	for _, reaction := range reactionMap {
		reactions = append(reactions, *reaction)
	}
	
	return reactions
}

// extractReplyInfo extracts reply information from message content
func extractReplyInfo(content map[string]interface{}) *ReplyInfo {
	if relatesTo, exists := content["m.relates_to"]; exists {
		if relatesMap, ok := relatesTo.(map[string]interface{}); ok {
			if relType, exists := relatesMap["rel_type"]; exists && relType == "m.reply" {
				if inReplyTo, exists := relatesMap["m.in_reply_to"]; exists {
					if replyMap, ok := inReplyTo.(map[string]interface{}); ok {
						if eventID, exists := replyMap["event_id"]; exists {
							if eventIDStr, ok := eventID.(string); ok {
								return &ReplyInfo{
									EventID: eventIDStr,
									// Note: We'd need to look up the original message for more details
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// isMessageEdited checks if a message has been edited
func isMessageEdited(content map[string]interface{}) bool {
	if relatesTo, exists := content["m.relates_to"]; exists {
		if relatesMap, ok := relatesTo.(map[string]interface{}); ok {
			if relType, exists := relatesMap["rel_type"]; exists && relType == "m.replace" {
				return true
			}
		}
	}
	return false
}

// extractEditHistory finds edit history for a message
func extractEditHistory(messages []*Message, eventID string) []EditInfo {
	var edits []EditInfo
	
	for _, msg := range messages {
		if relatesTo, exists := msg.Content["m.relates_to"]; exists {
			if relatesMap, ok := relatesTo.(map[string]interface{}); ok {
				if relType, exists := relatesMap["rel_type"]; exists && relType == "m.replace" {
					if targetEvent, exists := relatesMap["event_id"]; exists && targetEvent == eventID {
						edit := EditInfo{
							EventID:   msg.EventID,
							Timestamp: msg.Timestamp,
						}
						if body, exists := msg.Content["body"]; exists {
							if bodyStr, ok := body.(string); ok {
								edit.NewContent = bodyStr
							}
						}
						edits = append(edits, edit)
					}
				}
			}
		}
	}
	
	return edits
}

// generateUserAvatar creates a simple avatar from display name
func generateUserAvatar(displayName string) string {
	if displayName == "" {
		return "?"
	}
	// Return first character, uppercased
	return strings.ToUpper(string([]rune(displayName)[0]))
}

// detectPlatform detects the platform from user ID
func detectPlatform(userID string) string {
	if strings.Contains(userID, "discordgo_") {
		return "Discord"
	}
	if strings.Contains(userID, "telegram") {
		return "Telegram"
	}
	if strings.Contains(userID, "matrix.org") || strings.Contains(userID, "beeper.local") {
		return "Matrix"
	}
	return "Unknown"
}
