package archive

import (
	"context"
	"fmt"
	"time"
)

// AnalyticsService provides advanced DuckDB analytics for comprehensive message insights
type AnalyticsService interface {
	// Message volume analytics
	GetMessageVolumeByHour(roomID string, days int) ([]HourlyStats, error)
	GetMessageVolumeByUser(roomID string) ([]UserStats, error)

	// Content analytics
	GetMessageTypeDistribution(roomID string) (map[string]int, error)
}

// Analytics data structures
type HourlyStats struct {
	Hour         time.Time `json:"hour"`
	MessageCount int       `json:"message_count"`
	UserCount    int       `json:"user_count"`
}

type UserStats struct {
	UserID       string    `json:"user_id"`
	MessageCount int       `json:"message_count"`
	FirstMessage time.Time `json:"first_message"`
	LastMessage  time.Time `json:"last_message"`
}

// DuckDBAnalyticsService implements AnalyticsService using DuckDB
type DuckDBAnalyticsService struct {
	db DatabaseInterface
}

// NewAnalyticsService creates a new DuckDB analytics service
func NewAnalyticsService(db DatabaseInterface) AnalyticsService {
	return &DuckDBAnalyticsService{
		db: db,
	}
}

// GetMessageVolumeByHour returns hourly message statistics for a room
func (a *DuckDBAnalyticsService) GetMessageVolumeByHour(roomID string, days int) ([]HourlyStats, error) {
	ctx := context.Background()

	// Use a simpler approach that works with DuckDB
	query := `
		SELECT 
			date_trunc('hour', timestamp) as hour,
			COUNT(*) as message_count,
			COUNT(DISTINCT sender) as user_count
		FROM messages 
		WHERE room_id = ?
		GROUP BY date_trunc('hour', timestamp)
		ORDER BY hour
	`

	// Execute query using the database interface
	rows, err := a.db.ExecuteQuery(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute analytics query: %w", err)
	}

	var stats []HourlyStats
	for _, row := range rows {
		hour, ok := row["hour"].(time.Time)
		if !ok {
			continue
		}

		messageCount, ok := row["message_count"].(int64)
		if !ok {
			messageCount = 0
		}

		userCount, ok := row["user_count"].(int64)
		if !ok {
			userCount = 0
		}

		stats = append(stats, HourlyStats{
			Hour:         hour,
			MessageCount: int(messageCount),
			UserCount:    int(userCount),
		})
	}

	return stats, nil
}

// GetMessageVolumeByUser returns message count statistics per user for a room
func (a *DuckDBAnalyticsService) GetMessageVolumeByUser(roomID string) ([]UserStats, error) {
	ctx := context.Background()

	query := `
		SELECT 
			sender as user_id,
			COUNT(*) as message_count,
			MIN(timestamp) as first_message,
			MAX(timestamp) as last_message
		FROM messages 
		WHERE room_id = ?
		GROUP BY sender
		ORDER BY message_count DESC
	`

	rows, err := a.db.ExecuteQuery(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute user stats query: %w", err)
	}

	var stats []UserStats
	for _, row := range rows {
		userID, ok := row["user_id"].(string)
		if !ok {
			continue
		}

		messageCount, ok := row["message_count"].(int64)
		if !ok {
			messageCount = 0
		}

		firstMessage, ok := row["first_message"].(time.Time)
		if !ok {
			firstMessage = time.Time{}
		}

		lastMessage, ok := row["last_message"].(time.Time)
		if !ok {
			lastMessage = time.Time{}
		}

		stats = append(stats, UserStats{
			UserID:       userID,
			MessageCount: int(messageCount),
			FirstMessage: firstMessage,
			LastMessage:  lastMessage,
		})
	}

	return stats, nil
}

// GetMessageTypeDistribution returns distribution of message types in a room
func (a *DuckDBAnalyticsService) GetMessageTypeDistribution(roomID string) (map[string]int, error) {
	ctx := context.Background()

	query := `
		SELECT 
			message_type,
			COUNT(*) as count
		FROM messages 
		WHERE room_id = ?
		GROUP BY message_type
		ORDER BY count DESC
	`

	rows, err := a.db.ExecuteQuery(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute message type distribution query: %w", err)
	}

	distribution := make(map[string]int)
	for _, row := range rows {
		msgType, ok := row["message_type"].(string)
		if !ok {
			continue
		}

		count, ok := row["count"].(int64)
		if !ok {
			count = 0
		}

		distribution[msgType] = int(count)
	}

	return distribution, nil
}
