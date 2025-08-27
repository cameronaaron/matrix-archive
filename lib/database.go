package archive

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient   *mongo.Client
	mongoDatabase *mongo.Database
)

// InitMongoDB initializes the MongoDB connection
func InitMongoDB() error {
	var err error
	var clientOptions *options.ClientOptions

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI != "" {
		fmt.Printf("Connecting to %s\n", mongoURI)
		clientOptions = options.Client().ApplyURI(mongoURI)
	} else {
		// Default to local MongoDB
		clientOptions = options.Client().ApplyURI("mongodb://localhost:27017")
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	mongoClient, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Get database
	dbName := "matrix"
	if mongoURI != "" {
		// Extract database name from URI if provided
		// This is a simple extraction - in production you might want to use a proper URI parser
		uri := clientOptions.GetURI()
		if uri != "" {
			// Extract database name from the end of the URI path
			// This is a simplified approach
			dbName = "matrix" // Keep default for now
		}
	}

	mongoDatabase = mongoClient.Database(dbName)
	log.Printf("Connected to MongoDB database: %s", dbName)

	return nil
}

// GetCollection returns a MongoDB collection
func GetCollection(name string) *mongo.Collection {
	return mongoDatabase.Collection(name)
}

// GetMessagesCollection returns the messages collection
func GetMessagesCollection() *mongo.Collection {
	return GetCollection("message")
}

// GetMongoClient returns the MongoDB client for testing
func GetMongoClient() *mongo.Client {
	return mongoClient
}

// GetMongoDatabase returns the MongoDB database for testing
func GetMongoDatabase() *mongo.Database {
	return mongoDatabase
}

// CloseMongoDB closes the MongoDB connection
func CloseMongoDB() {
	if mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		mongoClient.Disconnect(ctx)
	}
}
