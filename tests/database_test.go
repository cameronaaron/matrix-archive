package tests

import (
	archive "github.com/osteele/matrix-archive/lib"
	"testing"
)

func TestDatabasePlaceholder(t *testing.T) {
	// Simple placeholder test to ensure package compiles
	_ = archive.GetCollection
}
