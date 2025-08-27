package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestInitMongoDB_DefaultURI(t *testing.T) {
	// Clear any existing MONGODB_URI
	originalURI := os.Getenv("MONGODB_URI")
	os.Unsetenv("MONGODB_URI")
	defer func() {
		if originalURI != "" {
			os.Setenv("MONGODB_URI", originalURI)
		}
	}()

	// This test requires a local MongoDB instance
	// Skip if MongoDB is not available
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	testClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
		return
	}
	defer testClient.Disconnect(ctx)

	if err := testClient.Ping(ctx, nil); err != nil {
		t.Skip("MongoDB not available for testing")
		return
	}

	// Clean up any existing connection
	CloseMongoDB()

	err = InitMongoDB()
	assert.NoError(t, err)
	assert.NotNil(t, mongoClient)
	assert.NotNil(t, mongoDatabase)
	assert.Equal(t, "matrix", mongoDatabase.Name())

	// Clean up
	CloseMongoDB()
}

func TestInitMongoDB_CustomURI(t *testing.T) {
	// Set custom URI
	originalURI := os.Getenv("MONGODB_URI")
	os.Setenv("MONGODB_URI", "mongodb://localhost:27017/testdb")
	defer func() {
		if originalURI != "" {
			os.Setenv("MONGODB_URI", originalURI)
		} else {
			os.Unsetenv("MONGODB_URI")
		}
	}()

	// This test requires a local MongoDB instance
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	testClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
		return
	}
	defer testClient.Disconnect(ctx)

	if err := testClient.Ping(ctx, nil); err != nil {
		t.Skip("MongoDB not available for testing")
		return
	}

	// Clean up any existing connection
	CloseMongoDB()

	err = InitMongoDB()
	assert.NoError(t, err)
	assert.NotNil(t, mongoClient)
	assert.NotNil(t, mongoDatabase)

	// Clean up
	CloseMongoDB()
}

func TestGetCollection(t *testing.T) {
	// This test requires a MongoDB connection
	originalURI := os.Getenv("MONGODB_URI")
	os.Unsetenv("MONGODB_URI")
	defer func() {
		if originalURI != "" {
			os.Setenv("MONGODB_URI", originalURI)
		}
	}()

	// Check if MongoDB is available
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	testClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
		return
	}
	defer testClient.Disconnect(ctx)

	if err := testClient.Ping(ctx, nil); err != nil {
		t.Skip("MongoDB not available for testing")
		return
	}

	// Initialize MongoDB
	err = InitMongoDB()
	assert.NoError(t, err)
	defer CloseMongoDB()

	// Test GetCollection
	collection := GetCollection("test")
	assert.NotNil(t, collection)
	assert.Equal(t, "test", collection.Name())
	assert.Equal(t, "matrix", collection.Database().Name())
}

func TestGetMessagesCollection(t *testing.T) {
	// This test requires a MongoDB connection
	originalURI := os.Getenv("MONGODB_URI")
	os.Unsetenv("MONGODB_URI")
	defer func() {
		if originalURI != "" {
			os.Setenv("MONGODB_URI", originalURI)
		}
	}()

	// Check if MongoDB is available
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	testClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
		return
	}
	defer testClient.Disconnect(ctx)

	if err := testClient.Ping(ctx, nil); err != nil {
		t.Skip("MongoDB not available for testing")
		return
	}

	// Initialize MongoDB
	err = InitMongoDB()
	assert.NoError(t, err)
	defer CloseMongoDB()

	// Test GetMessagesCollection
	collection := GetMessagesCollection()
	assert.NotNil(t, collection)
	assert.Equal(t, "message", collection.Name())
	assert.Equal(t, "matrix", collection.Database().Name())
}

func TestCloseMongoDB(t *testing.T) {
	// Test closing when no connection exists
	mongoClient = nil
	mongoDatabase = nil

	// Should not panic
	CloseMongoDB()

	// This test requires a MongoDB connection for full coverage
	originalURI := os.Getenv("MONGODB_URI")
	os.Unsetenv("MONGODB_URI")
	defer func() {
		if originalURI != "" {
			os.Setenv("MONGODB_URI", originalURI)
		}
	}()

	// Check if MongoDB is available
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	testClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
		return
	}
	defer testClient.Disconnect(ctx)

	if err := testClient.Ping(ctx, nil); err != nil {
		t.Skip("MongoDB not available for testing")
		return
	}

	// Initialize and then close
	err = InitMongoDB()
	assert.NoError(t, err)

	CloseMongoDB()
	// After closing, the client should be disconnected
	// We can't easily test this without affecting the global state
}

func TestInitMongoDB_ConnectionError(t *testing.T) {
	// Set an invalid URI to test connection error
	originalURI := os.Getenv("MONGODB_URI")
	os.Setenv("MONGODB_URI", "mongodb://invalid:99999")
	defer func() {
		if originalURI != "" {
			os.Setenv("MONGODB_URI", originalURI)
		} else {
			os.Unsetenv("MONGODB_URI")
		}
	}()

	// Clean up any existing connection
	CloseMongoDB()

	err := InitMongoDB()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to MongoDB")
}

func TestGetCollection_WithoutInit(t *testing.T) {
	// Save current state
	originalClient := mongoClient
	originalDB := mongoDatabase
	defer func() {
		mongoClient = originalClient
		mongoDatabase = originalDB
	}()

	// Clear database reference
	mongoDatabase = nil

	// This should panic since mongoDatabase is nil
	assert.Panics(t, func() {
		GetCollection("test")
	})
}

func TestGetMessagesCollection_WithoutInit(t *testing.T) {
	// Save current state
	originalClient := mongoClient
	originalDB := mongoDatabase
	defer func() {
		mongoClient = originalClient
		mongoDatabase = originalDB
	}()

	// Clear database reference
	mongoDatabase = nil

	// This should panic since mongoDatabase is nil
	assert.Panics(t, func() {
		GetMessagesCollection()
	})
}
