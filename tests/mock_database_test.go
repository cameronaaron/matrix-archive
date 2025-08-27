package tests

import (
	"testing"

	archive "github.com/osteele/matrix-archive/lib"
	"github.com/stretchr/testify/assert"
)

// TestMockDatabaseOperations tests database operations with DuckDB
func TestMockDatabaseOperations(t *testing.T) {
	t.Run("Database_DuckDB_InMemory", func(t *testing.T) {
		// Test DuckDB initialization with in-memory database
		config := &archive.DatabaseConfig{
			DatabaseURL: ":memory:",
			IsInMemory:  true,
		}
		
		err := archive.InitDatabase(config)
		assert.NoError(t, err)
		defer archive.CloseDatabase()

		// Test that we can get the database interface
		db := archive.GetDatabase()
		assert.NotNil(t, db)
	})

	t.Run("Database_DuckDB_FileSystem", func(t *testing.T) {
		// Test DuckDB initialization with file system database
		config := &archive.DatabaseConfig{
			DatabaseURL: "/tmp/test_db.duckdb",
			IsInMemory:  false,
		}
		
		err := archive.InitDatabase(config)
		assert.NoError(t, err)
		defer archive.CloseDatabase()

		// Test that we can get the database interface
		db := archive.GetDatabase()
		assert.NotNil(t, db)
	})

	t.Run("Database_Analytics_Service", func(t *testing.T) {
		// Test analytics service creation
		config := &archive.DatabaseConfig{
			DatabaseURL: ":memory:",
			IsInMemory:  true,
		}
		
		err := archive.InitDatabase(config)
		assert.NoError(t, err)
		defer archive.CloseDatabase()

		// Test analytics service creation
		analyticsService := archive.NewAnalyticsService(archive.GetDatabase())
		assert.NotNil(t, analyticsService)

		// Test empty database analytics
		stats, err := analyticsService.GetMessageVolumeByHour("!test:example.com", 7)
		assert.NoError(t, err)
		assert.Empty(t, stats)
	})
}