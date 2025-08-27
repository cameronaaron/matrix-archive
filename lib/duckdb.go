package archive

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

// DuckDBDatabase implements DatabaseInterface using DuckDB
type DuckDBDatabase struct {
	db     *sql.DB
	config *DatabaseConfig
}

// NewDuckDBDatabase creates a new DuckDB database instance
func NewDuckDBDatabase(config *DatabaseConfig) *DuckDBDatabase {
	if config == nil {
		config = &DatabaseConfig{
			DatabaseURL: ":memory:",
			IsInMemory:  true,
			MaxConns:    10,
			Debug:       false,
		}
	}
	return &DuckDBDatabase{
		config: config,
	}
}

// Connect establishes a connection to DuckDB
func (d *DuckDBDatabase) Connect(ctx context.Context) error {
	var err error

	// Construct connection string
	connStr := d.config.DatabaseURL
	if connStr == "" {
		if d.config.IsInMemory {
			connStr = ":memory:"
		} else {
			// Default to file-based database
			connStr = "matrix_archive.duckdb"
		}
	}

	// Open database connection
	d.db, err = sql.Open("duckdb", connStr)
	if err != nil {
		return fmt.Errorf("failed to open DuckDB connection: %w", err)
	}

	// Configure connection pool
	d.db.SetMaxOpenConns(d.config.MaxConns)
	d.db.SetMaxIdleConns(d.config.MaxConns / 2)
	d.db.SetConnMaxLifetime(time.Hour)

	// Test the connection
	if err := d.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping DuckDB: %w", err)
	}

	// Create tables if they don't exist
	if err := d.CreateTables(ctx); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	if d.config.Debug {
		log.Printf("Connected to DuckDB at: %s", connStr)
	}

	return nil
}

// Close closes the database connection
func (d *DuckDBDatabase) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Ping tests the database connection
func (d *DuckDBDatabase) Ping(ctx context.Context) error {
	if d.db == nil {
		return fmt.Errorf("database not connected")
	}
	return d.db.PingContext(ctx)
}

// CreateTables creates the necessary tables for the archive
func (d *DuckDBDatabase) CreateTables(ctx context.Context) error {
	// Create messages table with auto-incrementing ID
	createMessagesTable := `
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY,
			room_id VARCHAR NOT NULL,
			event_id VARCHAR NOT NULL UNIQUE,
			sender VARCHAR NOT NULL,
			user_id VARCHAR,
			message_type VARCHAR NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			content JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	// Create sequence for auto-incrementing ID (DuckDB specific)
	createSequence := `
		CREATE SEQUENCE IF NOT EXISTS seq_messages_id START 1;
	`

	// Create indexes for better query performance
	createIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_messages_room_id ON messages(room_id);",
		"CREATE INDEX IF NOT EXISTS idx_messages_event_id ON messages(event_id);",
		"CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender);",
		"CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_messages_room_timestamp ON messages(room_id, timestamp);",
	}

	// Execute sequence creation first
	if _, err := d.db.ExecContext(ctx, createSequence); err != nil {
		return fmt.Errorf("failed to create sequence: %w", err)
	}

	// Execute table creation
	if _, err := d.db.ExecContext(ctx, createMessagesTable); err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	// Execute index creation
	for _, indexSQL := range createIndexes {
		if _, err := d.db.ExecContext(ctx, indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// Migrate applies any necessary database migrations
func (d *DuckDBDatabase) Migrate(ctx context.Context) error {
	// For now, this is a no-op as we're starting fresh
	// In the future, we could add version-based migrations
	return nil
}

// ExecuteQuery executes a raw SQL query and returns results as map slices
func (d *DuckDBDatabase) ExecuteQuery(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		// Create slice of interface{} to hold values
		values := make([]interface{}, len(columns))
		valuePointers := make([]interface{}, len(columns))
		for i := range values {
			valuePointers[i] = &values[i]
		}

		// Scan row values
		if err := rows.Scan(valuePointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create map from column names to values
		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// InsertMessage inserts a single message into the database
func (d *DuckDBDatabase) InsertMessage(ctx context.Context, message *Message) error {
	insertSQL := `
		INSERT INTO messages (id, room_id, event_id, sender, user_id, message_type, timestamp, content)
		VALUES (nextval('seq_messages_id'), ?, ?, ?, ?, ?, ?, ?)
	`

	contentJSON, err := message.ContentJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize content: %w", err)
	}

	result, err := d.db.ExecContext(ctx, insertSQL,
		message.RoomID,
		message.EventID,
		message.Sender,
		message.UserID,
		message.MessageType,
		message.Timestamp,
		contentJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err == nil {
		message.ID = id
	}

	return nil
}

// InsertMessageBatch inserts multiple messages in a batch for better performance
func (d *DuckDBDatabase) InsertMessageBatch(ctx context.Context, messages []*Message) (int, error) {
	if len(messages) == 0 {
		return 0, nil
	}

	// Prepare batch insert statement
	insertSQL := `
		INSERT INTO messages (id, room_id, event_id, sender, user_id, message_type, timestamp, content)
		VALUES (nextval('seq_messages_id'), ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := d.db.PrepareContext(ctx, insertSQL)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare batch insert: %w", err)
	}
	defer stmt.Close()

	// Begin transaction for better performance
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	insertedCount := 0
	for _, message := range messages {
		contentJSON, err := message.ContentJSON()
		if err != nil {
			log.Printf("Warning: failed to serialize content for message %s: %v", message.EventID, err)
			continue
		}

		_, err = tx.StmtContext(ctx, stmt).ExecContext(ctx,
			message.RoomID,
			message.EventID,
			message.Sender,
			message.UserID,
			message.MessageType,
			message.Timestamp,
			contentJSON,
		)

		if err != nil {
			// Log error but continue with other messages
			log.Printf("Warning: failed to insert message %s: %v", message.EventID, err)
			continue
		}
		insertedCount++
	}

	if err := tx.Commit(); err != nil {
		return insertedCount, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return insertedCount, nil
}

// GetMessage retrieves a single message by event ID
func (d *DuckDBDatabase) GetMessage(ctx context.Context, eventID string) (*Message, error) {
	selectSQL := `
		SELECT id, room_id, event_id, sender, user_id, message_type, timestamp, content::VARCHAR as content_json
		FROM messages 
		WHERE event_id = ?
	`

	row := d.db.QueryRowContext(ctx, selectSQL, eventID)

	message := &Message{}
	var contentJSON string
	var id int64

	err := row.Scan(
		&id,
		&message.RoomID,
		&message.EventID,
		&message.Sender,
		&message.UserID,
		&message.MessageType,
		&message.Timestamp,
		&contentJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message not found: %s", eventID)
		}
		return nil, fmt.Errorf("failed to scan message: %w", err)
	}

	message.ID = id
	if err := message.SetContentFromJSON(contentJSON); err != nil {
		return nil, fmt.Errorf("failed to deserialize content: %w", err)
	}

	return message, nil
}

// GetMessages retrieves messages based on filter criteria
func (d *DuckDBDatabase) GetMessages(ctx context.Context, filter *MessageFilter, limit int, offset int) ([]*Message, error) {
	query, args := d.buildSelectQuery(filter, limit, offset)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		message := &Message{}
		var contentJSON string
		var id int64

		err := rows.Scan(
			&id,
			&message.RoomID,
			&message.EventID,
			&message.Sender,
			&message.UserID,
			&message.MessageType,
			&message.Timestamp,
			&contentJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		message.ID = id
		if err := message.SetContentFromJSON(contentJSON); err != nil {
			log.Printf("Warning: failed to deserialize content for message %s: %v", message.EventID, err)
			continue
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return messages, nil
}

// GetMessageCount returns the total count of messages matching the filter
func (d *DuckDBDatabase) GetMessageCount(ctx context.Context, filter *MessageFilter) (int64, error) {
	query, args := d.buildCountQuery(filter)

	row := d.db.QueryRowContext(ctx, query, args...)

	var count int64
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get message count: %w", err)
	}

	return count, nil
}

// DeleteMessage deletes a message by event ID
func (d *DuckDBDatabase) DeleteMessage(ctx context.Context, eventID string) error {
	deleteSQL := "DELETE FROM messages WHERE event_id = ?"

	result, err := d.db.ExecContext(ctx, deleteSQL, eventID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("message not found: %s", eventID)
	}

	return nil
}

// GetRooms returns a list of unique room IDs in the database
func (d *DuckDBDatabase) GetRooms(ctx context.Context) ([]string, error) {
	selectSQL := "SELECT DISTINCT room_id FROM messages ORDER BY room_id"

	rows, err := d.db.QueryContext(ctx, selectSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query rooms: %w", err)
	}
	defer rows.Close()

	var rooms []string
	for rows.Next() {
		var roomID string
		if err := rows.Scan(&roomID); err != nil {
			return nil, fmt.Errorf("failed to scan room ID: %w", err)
		}
		rooms = append(rooms, roomID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rooms: %w", err)
	}

	return rooms, nil
}

// GetRoomMessageCount returns the number of messages in a specific room
func (d *DuckDBDatabase) GetRoomMessageCount(ctx context.Context, roomID string) (int64, error) {
	selectSQL := "SELECT COUNT(*) FROM messages WHERE room_id = ?"

	row := d.db.QueryRowContext(ctx, selectSQL, roomID)

	var count int64
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get room message count: %w", err)
	}

	return count, nil
}

// buildSelectQuery constructs a SELECT query with WHERE clauses based on the filter
func (d *DuckDBDatabase) buildSelectQuery(filter *MessageFilter, limit int, offset int) (string, []interface{}) {
	baseQuery := `
		SELECT id, room_id, event_id, sender, user_id, message_type, timestamp, content::VARCHAR as content_json
		FROM messages
	`

	whereClause, args := d.buildWhereClause(filter)

	query := baseQuery + whereClause + " ORDER BY timestamp ASC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", offset)
	}

	return query, args
}

// buildCountQuery constructs a COUNT query with WHERE clauses based on the filter
func (d *DuckDBDatabase) buildCountQuery(filter *MessageFilter) (string, []interface{}) {
	baseQuery := "SELECT COUNT(*) FROM messages"

	whereClause, args := d.buildWhereClause(filter)

	return baseQuery + whereClause, args
}

// buildWhereClause constructs WHERE clause and arguments based on the filter
func (d *DuckDBDatabase) buildWhereClause(filter *MessageFilter) (string, []interface{}) {
	if filter == nil {
		return "", nil
	}

	var conditions []string
	var args []interface{}

	if filter.RoomID != "" {
		conditions = append(conditions, "room_id = ?")
		args = append(args, filter.RoomID)
	}

	if filter.EventID != "" {
		conditions = append(conditions, "event_id = ?")
		args = append(args, filter.EventID)
	}

	if filter.Sender != "" {
		conditions = append(conditions, "sender = ?")
		args = append(args, filter.Sender)
	}

	if filter.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *filter.EndTime)
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

// Global database instance and configuration
var (
	database DatabaseInterface
	dbConfig *DatabaseConfig
)

// InitDatabase initializes the database connection using the provided config
func InitDatabase(config *DatabaseConfig) error {
	// Create DuckDB instance
	duckDB := NewDuckDBDatabase(config)

	// Connect to database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := duckDB.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set global database instance
	database = duckDB
	dbConfig = config

	return nil
}

// InitDuckDB initializes DuckDB with default configuration (for backward compatibility)
func InitDuckDB() error {
	// Get database URL from environment, default to file-based
	dbURL := os.Getenv("DUCKDB_URL")
	if dbURL == "" {
		dbURL = "matrix_archive.duckdb"
	}

	config := &DatabaseConfig{
		DatabaseURL: dbURL,
		IsInMemory:  dbURL == ":memory:",
		MaxConns:    10,
		Debug:       os.Getenv("DB_DEBUG") == "true",
	}

	return InitDatabase(config)
}

// GetDatabase returns the global database instance
func GetDatabase() DatabaseInterface {
	return database
}

// CloseDatabase closes the global database connection
func CloseDatabase() error {
	if database != nil {
		return database.Close()
	}
	return nil
}
