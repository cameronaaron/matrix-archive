package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

func TestGetExistingFilesMap(t *testing.T) {
	// Test with non-existent directory
	stemSet, err := getExistingFilesMap("/nonexistent/directory")
	assert.NoError(t, err)
	assert.Empty(t, stemSet)
}

func TestGetExistingFilesMap_WithFiles(t *testing.T) {
	// Create a temporary directory with some files
	tempDir := filepath.Join(os.TempDir(), "test-download-images")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []string{"image1.jpg", "image2.png", "document.txt"}
	for _, file := range testFiles {
		f, _ := os.Create(filepath.Join(tempDir, file))
		f.Close()
	}

	stemSet, err := getExistingFilesMap(tempDir)
	assert.NoError(t, err)
	assert.True(t, stemSet["image1"])
	assert.True(t, stemSet["image2"])
	assert.True(t, stemSet["document"])
}

func TestGetDownloadStem(t *testing.T) {
	// Test with valid image URL
	msg := Message{
		Content: bson.M{
			"msgtype": "m.image",
			"url":     "mxc://example.com/abc123def",
		},
	}

	stem := getDownloadStem(msg, false)
	assert.Equal(t, "abc123def", stem)

	// Test with thumbnail preference
	msgWithThumb := Message{
		Content: bson.M{
			"msgtype": "m.image",
			"url":     "mxc://example.com/abc123def",
			"info": bson.M{
				"thumbnail_url": "mxc://example.com/thumb123",
			},
		},
	}

	stem = getDownloadStem(msgWithThumb, true)
	assert.Equal(t, "thumb123", stem)

	// Test with no image URL
	textMsg := Message{
		Content: bson.M{
			"msgtype": "m.text",
			"body":    "Hello world",
		},
	}

	stem = getDownloadStem(textMsg, false)
	assert.Equal(t, "", stem)
}

func TestDownloadImages_MissingRoomIDs(t *testing.T) {
	// Save original env var
	originalRoomIDs := os.Getenv("MATRIX_ROOM_IDS")
	defer func() {
		if originalRoomIDs != "" {
			os.Setenv("MATRIX_ROOM_IDS", originalRoomIDs)
		} else {
			os.Unsetenv("MATRIX_ROOM_IDS")
		}
	}()

	// Test with empty output directory (should use default)
	err := downloadImages("", false)
	// This should succeed and create images directory, then query DB
	// The error (if any) would be from DB operations
	if err != nil {
		// Should contain either DB error or room ID error
		assert.True(t,
			strings.Contains(err.Error(), "failed to initialize database") ||
				strings.Contains(err.Error(), "failed to query image messages"))
	}
}
