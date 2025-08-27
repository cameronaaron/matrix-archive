package archive

import (
	"context"
)

// DatabaseInterface defines the database operations needed by the archive tool
type DatabaseInterface interface {
	// Connection management
	Connect(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error

	// Message operations
	InsertMessage(ctx context.Context, message *Message) error
	InsertMessageBatch(ctx context.Context, messages []*Message) (int, error)
	GetMessage(ctx context.Context, eventID string) (*Message, error)
	GetMessages(ctx context.Context, filter *MessageFilter, limit int, offset int) ([]*Message, error)
	GetMessageCount(ctx context.Context, filter *MessageFilter) (int64, error)
	DeleteMessage(ctx context.Context, eventID string) error

	// Room operations
	GetRooms(ctx context.Context) ([]string, error)
	GetRoomMessageCount(ctx context.Context, roomID string) (int64, error)

	// Utility operations
	CreateTables(ctx context.Context) error
	Migrate(ctx context.Context) error
	
	// Analytics operations (for advanced analytics)
	ExecuteQuery(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error)
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	DatabaseURL string // For DuckDB, this could be file path or ":memory:"
	IsInMemory  bool
	MaxConns    int
	Debug       bool
}

// MessageFilter represents filters for querying messages (already defined in models.go but extending for SQL)
// This will be updated to work with SQL instead of BSON
