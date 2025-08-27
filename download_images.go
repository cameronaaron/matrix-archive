package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// downloadImages downloads images from messages to a local directory
func downloadImages(outputDir string, thumbnails bool) error {
	// Initialize database connection
	if err := InitMongoDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer CloseMongoDB()

	// Determine output directory
	if outputDir == "" {
		if thumbnails {
			outputDir = "thumbnails"
		} else {
			outputDir = "images"
		}
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Query image messages from database
	collection := GetMessagesCollection()
	filter := bson.M{"content.msgtype": "m.image"}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return fmt.Errorf("failed to query image messages: %w", err)
	}
	defer cursor.Close(context.Background())

	var messages []Message
	if err := cursor.All(context.Background(), &messages); err != nil {
		return fmt.Errorf("failed to decode messages: %w", err)
	}

	if len(messages) == 0 {
		fmt.Println("No image messages found")
		return nil
	}

	// Get list of already downloaded files
	existingStemSet, err := getExistingFilesMap(outputDir)
	if err != nil {
		return fmt.Errorf("failed to get existing files: %w", err)
	}

	// Filter messages to only download new ones
	var newMessages []Message
	for _, msg := range messages {
		stem := getDownloadStem(msg, thumbnails)
		if stem == "" {
			continue
		}
		if _, exists := existingStemSet[stem]; !exists {
			newMessages = append(newMessages, msg)
		}
	}

	skipCount := len(messages) - len(newMessages)
	if skipCount > 0 {
		noun := "thumbnails"
		if !thumbnails {
			noun = "images"
		}
		fmt.Printf("Skipping %d already-downloaded %s\n", skipCount, noun)
	}

	if len(newMessages) == 0 {
		fmt.Println("Nothing to do")
		return nil
	}

	noun := "thumbnails"
	if !thumbnails {
		noun = "images"
	}
	fmt.Printf("Downloading %d new %s...\n", len(newMessages), noun)

	// Download new images
	return runDownloads(newMessages, outputDir, thumbnails)
}

// getExistingFilesMap returns a map of existing file stems in the directory
func getExistingFilesMap(dir string) (map[string]bool, error) {
	stemSet := make(map[string]bool)

	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return stemSet, nil
		}
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			stem := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			stemSet[stem] = true
		}
	}

	return stemSet, nil
}

// getDownloadStem returns the file stem for a message's image
func getDownloadStem(msg Message, preferThumbnails bool) string {
	var imageURL string

	if preferThumbnails {
		imageURL = msg.ThumbnailURL()
	}
	if imageURL == "" {
		imageURL = msg.ImageURL()
	}

	if imageURL == "" {
		return ""
	}

	// Parse URL and extract path
	u, err := url.Parse(imageURL)
	if err != nil {
		return ""
	}

	return strings.TrimPrefix(u.Path, "/")
}

// runDownloads downloads images from the message list
func runDownloads(messages []Message, downloadDir string, preferThumbnails bool) error {
	client := &http.Client{}

	for _, msg := range messages {
		var imageURL string
		if preferThumbnails {
			imageURL = msg.ThumbnailURL()
		}
		if imageURL == "" {
			imageURL = msg.ImageURL()
		}

		if imageURL == "" {
			continue
		}

		// Convert mxc URL to download URL
		downloadURL, err := GetDownloadURL(imageURL)
		if err != nil {
			fmt.Printf("Failed to get download URL for %s: %v. Skipping...\n", imageURL, err)
			continue
		}

		// Get content type and validate it's an image
		resp, err := client.Head(downloadURL)
		if err != nil {
			fmt.Printf("Failed to check %s: %v. Skipping...\n", imageURL, err)
			continue
		}
		resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") {
			fmt.Printf("Skipping %s: %s\n", imageURL, contentType)
			continue
		}

		// Extract file extension from content type
		parts := strings.Split(contentType, "/")
		var ext string
		if len(parts) == 2 {
			ext = "." + parts[1]
		} else {
			ext = ".jpg" // fallback
		}

		// Download the image
		resp, err = client.Get(downloadURL)
		if err != nil {
			fmt.Printf("Failed to download %s: %v. Skipping...\n", imageURL, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Failed to download %s: HTTP %d. Skipping...\n", imageURL, resp.StatusCode)
			continue
		}

		// Create filename
		stem := getDownloadStem(msg, preferThumbnails)
		filename := filepath.Join(downloadDir, stem+ext)

		// Create directory for file if needed
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			fmt.Printf("Failed to create directory for %s: %v. Skipping...\n", filename, err)
			continue
		}

		// Create file
		file, err := os.Create(filename)
		if err != nil {
			fmt.Printf("Failed to create file %s: %v. Skipping...\n", filename, err)
			continue
		}

		// Copy data
		fmt.Printf("Downloading %s -> %s\n", imageURL, filename)
		_, err = io.Copy(file, resp.Body)
		file.Close()

		if err != nil {
			fmt.Printf("Failed to write %s: %v\n", filename, err)
			os.Remove(filename) // Clean up partial file
		}
	}

	return nil
}
