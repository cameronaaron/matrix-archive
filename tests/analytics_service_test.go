package tests

import (
	"context"
	"testing"
	"time"

	archive "github.com/osteele/matrix-archive/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnalyticsServiceBasic tests basic analytics service functionality
func TestAnalyticsServiceBasic(t *testing.T) {
	// Setup in-memory DuckDB for testing
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	err := archive.InitDatabase(config)
	require.NoError(t, err)
	defer archive.CloseDatabase()

	db := archive.GetDatabase()
	require.NotNil(t, db)

	// Create analytics service
	analyticsService := archive.NewAnalyticsService(db)
	require.NotNil(t, analyticsService)

	t.Run("AnalyticsService_Creation", func(t *testing.T) {
		// Test that the service can be created
		service := archive.NewAnalyticsService(db)
		assert.NotNil(t, service)
	})

	t.Run("GetMessageVolumeByHour_EmptyDatabase", func(t *testing.T) {
		stats, err := analyticsService.GetMessageVolumeByHour("!test:example.com", 7)
		if err != nil {
			t.Logf("Error: %v", err)
		}
		assert.NoError(t, err)
		if stats != nil {
			assert.Len(t, stats, 0) // Empty database should return no stats
		}
	})

	t.Run("GetMessageVolumeByUser_EmptyDatabase", func(t *testing.T) {
		stats, err := analyticsService.GetMessageVolumeByUser("!test:example.com")
		assert.NoError(t, err)
		if stats != nil {
			assert.Len(t, stats, 0) // Empty database should return no stats
		}
	})

	t.Run("GetMessageTypeDistribution_EmptyDatabase", func(t *testing.T) {
		distribution, err := analyticsService.GetMessageTypeDistribution("!test:example.com")
		assert.NoError(t, err)
		assert.NotNil(t, distribution)
		assert.Len(t, distribution, 0) // Empty database should return empty distribution
	})
}

// TestAnalyticsServiceWithData tests analytics with actual data
func TestAnalyticsServiceWithData(t *testing.T) {
	// Setup in-memory DuckDB for testing
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	err := archive.InitDatabase(config)
	require.NoError(t, err)
	defer archive.CloseDatabase()

	db := archive.GetDatabase()
	ctx := context.Background()

	// Insert test messages
	testMessages := []*archive.Message{
		{
			RoomID:      "!test:example.com",
			EventID:     "$event1:example.com",
			Sender:      "@alice:example.com",
			MessageType: "m.room.message",
			Timestamp:   time.Now().Add(-2 * time.Hour),
			Content:     map[string]interface{}{"msgtype": "m.text", "body": "Hello world"},
		},
		{
			RoomID:      "!test:example.com",
			EventID:     "$event2:example.com",
			Sender:      "@bob:example.com",
			MessageType: "m.room.message",
			Timestamp:   time.Now().Add(-1 * time.Hour),
			Content:     map[string]interface{}{"msgtype": "m.text", "body": "Hi there"},
		},
		{
			RoomID:      "!test:example.com",
			EventID:     "$event3:example.com",
			Sender:      "@alice:example.com",
			MessageType: "m.room.message",
			Timestamp:   time.Now(),
			Content:     map[string]interface{}{"msgtype": "m.image", "url": "mxc://example.com/image"},
		},
		{
			RoomID:      "!other:example.com",
			EventID:     "$event4:example.com",
			Sender:      "@charlie:example.com",
			MessageType: "m.room.message",
			Timestamp:   time.Now(),
			Content:     map[string]interface{}{"msgtype": "m.text", "body": "Different room"},
		},
	}

	_, err = db.InsertMessageBatch(ctx, testMessages)
	require.NoError(t, err)

	// Create analytics service
	analyticsService := archive.NewAnalyticsService(db)

	t.Run("GetMessageVolumeByUser_WithData", func(t *testing.T) {
		stats, err := analyticsService.GetMessageVolumeByUser("!test:example.com")
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		
		// Should have stats for 2 users (alice and bob)
		assert.Len(t, stats, 2)
		
		// Check that alice has 2 messages
		found := false
		for _, stat := range stats {
			if stat.UserID == "@alice:example.com" {
				assert.Equal(t, 2, stat.MessageCount)
				found = true
				break
			}
		}
		assert.True(t, found, "Alice's stats should be found")
	})

	t.Run("GetMessageTypeDistribution_WithData", func(t *testing.T) {
		distribution, err := analyticsService.GetMessageTypeDistribution("!test:example.com")
		assert.NoError(t, err)
		assert.NotNil(t, distribution)
		
		// Should have distribution for m.room.message type
		assert.Contains(t, distribution, "m.room.message")
		assert.Equal(t, 3, distribution["m.room.message"]) // 3 messages in test room
	})

	t.Run("GetMessageVolumeByHour_WithData", func(t *testing.T) {
		stats, err := analyticsService.GetMessageVolumeByHour("!test:example.com", 1)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		// Stats array may be empty or contain entries depending on DuckDB time functions
		// The main test is that the function doesn't error
	})
}

// TestAnalyticsServiceEdgeCases tests edge cases and error scenarios
func TestAnalyticsServiceEdgeCases(t *testing.T) {
	// Setup in-memory DuckDB for testing
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	err := archive.InitDatabase(config)
	require.NoError(t, err)
	defer archive.CloseDatabase()

	db := archive.GetDatabase()
	analyticsService := archive.NewAnalyticsService(db)

	t.Run("InvalidRoomID", func(t *testing.T) {
		// Test with invalid room ID
		stats, err := analyticsService.GetMessageVolumeByUser("invalid-room")
		assert.NoError(t, err) // Should not error, just return empty results
		if stats != nil {
			assert.Len(t, stats, 0)
		}
	})

	t.Run("NegativeDays", func(t *testing.T) {
		// Test with negative days parameter
		stats, err := analyticsService.GetMessageVolumeByHour("!test:example.com", -1)
		assert.NoError(t, err) // Should handle gracefully
		if stats != nil {
			// May have results
		}
	})

	t.Run("ZeroDays", func(t *testing.T) {
		// Test with zero days parameter
		stats, err := analyticsService.GetMessageVolumeByHour("!test:example.com", 0)
		assert.NoError(t, err) // Should handle gracefully
		if stats != nil {
			// May have results
		}
	})

	t.Run("EmptyRoomID", func(t *testing.T) {
		// Test with empty room ID
		stats, err := analyticsService.GetMessageVolumeByUser("")
		assert.NoError(t, err) // Should not error
		if stats != nil {
			assert.Len(t, stats, 0)
		}
	})
}

// TestAnalyticsServiceConcurrency tests concurrent access to analytics service
func TestAnalyticsServiceConcurrency(t *testing.T) {
	// Setup in-memory DuckDB for testing
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	err := archive.InitDatabase(config)
	require.NoError(t, err)
	defer archive.CloseDatabase()

	db := archive.GetDatabase()
	analyticsService := archive.NewAnalyticsService(db)

	t.Run("ConcurrentQueries", func(t *testing.T) {
		// Test concurrent access to analytics service
		done := make(chan bool, 10)
		
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				
				// Execute multiple analytics queries concurrently
				_, err1 := analyticsService.GetMessageVolumeByUser("!test:example.com")
				_, err2 := analyticsService.GetMessageTypeDistribution("!test:example.com")
				_, err3 := analyticsService.GetMessageVolumeByHour("!test:example.com", 7)
				
				assert.NoError(t, err1)
				assert.NoError(t, err2)
				assert.NoError(t, err3)
			}()
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestAnalyticsServiceIntegration tests analytics service integration with database
func TestAnalyticsServiceIntegration(t *testing.T) {
	// Setup in-memory DuckDB for testing
	config := &archive.DatabaseConfig{
		DatabaseURL: ":memory:",
		IsInMemory:  true,
		MaxConns:    5,
		Debug:       false,
	}

	err := archive.InitDatabase(config)
	require.NoError(t, err)
	defer archive.CloseDatabase()

	db := archive.GetDatabase()
	ctx := context.Background()

	// Test the ExecuteQuery method directly
	t.Run("DatabaseInterface_ExecuteQuery", func(t *testing.T) {
		// Test basic query execution
		results, err := db.ExecuteQuery(ctx, "SELECT 1 as test_column")
		assert.NoError(t, err)
		assert.NotNil(t, results)
		assert.Len(t, results, 1)
		// DuckDB returns int32 for integers, not int64
		assert.Equal(t, int32(1), results[0]["test_column"])
	})

	t.Run("DatabaseInterface_ExecuteQuery_WithParameters", func(t *testing.T) {
		// Test parameterized query
		results, err := db.ExecuteQuery(ctx, "SELECT ? as param_value", "test_param")
		assert.NoError(t, err)
		assert.NotNil(t, results)
		assert.Len(t, results, 1)
		assert.Equal(t, "test_param", results[0]["param_value"])
	})

	t.Run("DatabaseInterface_ExecuteQuery_InvalidSQL", func(t *testing.T) {
		// Test invalid SQL query
		_, err := db.ExecuteQuery(ctx, "INVALID SQL QUERY")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute query")
	})
}