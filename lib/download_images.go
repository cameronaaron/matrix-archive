package archive

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// downloadImages downloads images from messages to a local directory
func DownloadImages(outputDir string, thumbnails bool) error {
	// Initialize database connection with DuckDB
	if err := InitDuckDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer CloseDatabase()

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

	// Query all messages from DuckDB
	messages, err := GetDatabase().GetMessages(context.Background(), nil, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to query messages: %w", err)
	}

	// Filter for image messages
	var imageMessages []*Message
	for _, msg := range messages {
		if msg.IsImage() {
			imageMessages = append(imageMessages, msg)
		}
	}

	if len(imageMessages) == 0 {
		fmt.Println("No image messages found")
		return nil
	}

	// Get list of already downloaded files
	existingStemSet, err := GetExistingFilesMap(outputDir)
	if err != nil {
		return fmt.Errorf("failed to get existing files: %w", err)
	}

	// Filter messages to only download new ones
	var newMessages []*Message
	for _, msg := range imageMessages {
		stem := GetDownloadStem(*msg, thumbnails)
		if stem == "" {
			continue
		}
		if _, exists := existingStemSet[stem]; !exists {
			newMessages = append(newMessages, msg)
		}
	}

	skipCount := len(imageMessages) - len(newMessages)
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

// GetExistingFilesMap returns a map of existing file stems in the directory
func GetExistingFilesMap(dir string) (map[string]bool, error) {
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
func GetDownloadStem(msg Message, preferThumbnails bool) string {
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
func runDownloads(messages []*Message, downloadDir string, preferThumbnails bool) error {
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
		stem := GetDownloadStem(*msg, preferThumbnails)
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
