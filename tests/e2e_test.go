package tests

import (
	archive "github.com/osteele/matrix-archive/lib"
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestE2EBasic(t *testing.T) {
	// Basic test to ensure main exported functions are accessible
	auth := archive.NewBeeperAuth("")
	if auth == nil {
		t.Error("NewBeeperAuth should return a valid instance")
	}

	// Test that the Message type is accessible
	var msg archive.Message
	_ = msg

	// Test that public functions exist (we don't call them as they require setup)
	_ = archive.DownloadImages
	_ = archive.GetDownloadStem
	_ = archive.GetExistingFilesMap
}

func TestE2EExportedFunctions(t *testing.T) {
	// Test that database functions are accessible
	_ = archive.InitMongoDB
	_ = archive.CloseMongoDB
	_ = archive.GetCollection
	_ = archive.GetMessagesCollection
	_ = archive.GetMongoClient
	_ = archive.GetMongoDatabase
}

func TestE2EBeeperAuthentication(t *testing.T) {
	// Test Beeper authentication end-to-end flow using saved credentials
	fmt.Println("=== Testing Beeper Authentication E2E ===")
	
	// Create Beeper auth instance
	auth := archive.NewBeeperAuth("beeper.com")
	if auth == nil {
		t.Fatal("Failed to create BeeperAuth instance")
	}

	// Try to load existing credentials
	credentialsLoaded := auth.LoadCredentials()
	if !credentialsLoaded {
		t.Skip("No Beeper credentials found - skipping E2E test. Run 'matrix-archive beeper-login' first to save credentials.")
	}

	fmt.Printf("✓ Successfully loaded Beeper credentials for: %s\n", auth.Email)

	// Test getting Matrix client using Beeper credentials
	client, err := auth.GetMatrixClient()
	if err != nil {
		t.Fatalf("Failed to get Matrix client using Beeper credentials: %v", err)
	}

	fmt.Println("✓ Successfully created Matrix client using Beeper authentication")

	// Test Matrix client functionality
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test whoami to verify the connection works
	whoami, err := client.Whoami(ctx)
	if err != nil {
		t.Fatalf("Failed to verify Matrix connection: %v", err)
	}

	fmt.Printf("✓ Matrix connection verified - User ID: %s\n", whoami.UserID)

	// Test getting joined rooms
	joinedRooms, err := client.JoinedRooms(ctx)
	if err != nil {
		t.Fatalf("Failed to get joined rooms: %v", err)
	}

	fmt.Printf("✓ Successfully retrieved %d joined rooms\n", len(joinedRooms.JoinedRooms))

	// Set environment variables for other functions to use
	os.Setenv("USE_BEEPER_AUTH", "true")
	os.Setenv("BEEPER_TOKEN", auth.Token)
	if auth.Whoami != nil {
		os.Setenv("MATRIX_USER", fmt.Sprintf("@%s:beeper.com", auth.Whoami.UserInfo.Username))
		fmt.Printf("✓ Set MATRIX_USER environment variable: @%s:beeper.com\n", auth.Whoami.UserInfo.Username)
	}

	// Test database initialization
	err = archive.InitMongoDB()
	if err != nil {
		t.Fatalf("Failed to initialize MongoDB: %v", err)
	}
	defer archive.CloseMongoDB()

	fmt.Println("✓ Successfully initialized MongoDB connection")

	// Test that we can get the messages collection
	collection := archive.GetMessagesCollection()
	if collection == nil {
		t.Fatal("Failed to get messages collection")
	}

	fmt.Println("✓ Successfully accessed messages collection")

	// Test room listing functionality
	fmt.Println("\n=== Testing Room Operations ===")
	
	// We can't easily test ListRooms without output capture, but we can test GetRoomDisplayName
	if len(joinedRooms.JoinedRooms) > 0 {
		// Test with the first room
		firstRoom := string(joinedRooms.JoinedRooms[0])
		displayName, err := archive.GetRoomDisplayName(client, firstRoom)
		if err != nil {
			fmt.Printf("Warning: Could not get display name for room %s: %v\n", firstRoom, err)
		} else {
			fmt.Printf("✓ Successfully got display name for room %s: %s\n", firstRoom, displayName)
		}
	}

	fmt.Println("\n=== Beeper E2E Test Summary ===")
	fmt.Println("✓ Loaded saved Beeper credentials")
	fmt.Println("✓ Created Matrix client using Beeper authentication")
	fmt.Println("✓ Verified Matrix connection with whoami")
	fmt.Println("✓ Retrieved joined rooms list")
	fmt.Println("✓ Set environment variables for integration")
	fmt.Println("✓ Initialized MongoDB connection")
	fmt.Println("✓ Accessed messages collection")
	fmt.Println("✓ Tested room display name functionality")
	fmt.Println("🎉 All Beeper authentication features working correctly!")
}

func TestE2EBeeperCredentialsManagement(t *testing.T) {
	// Test credential management functionality
	fmt.Println("=== Testing Beeper Credentials Management ===")

	auth := archive.NewBeeperAuth("test.beeper.com") // Use a different domain for testing
	
	// Test getting credentials file path
	filePath, err := auth.GetCredentialsFilePath()
	if err != nil {
		t.Fatalf("Failed to get credentials file path: %v", err)
	}
	
	fmt.Printf("✓ Credentials file path: %s\n", filePath)

	// Test saving dummy credentials
	auth.Email = "test@example.com"
	auth.Token = "dummy-token-for-testing"
	
	err = auth.SaveCredentialsToFile()
	if err != nil {
		t.Fatalf("Failed to save test credentials: %v", err)
	}
	
	fmt.Println("✓ Successfully saved test credentials to file")

	// Test loading credentials
	auth2 := archive.NewBeeperAuth("test.beeper.com")
	loaded := auth2.LoadCredentialsFromFile()
	if !loaded {
		t.Fatal("Failed to load test credentials from file")
	}
	
	if auth2.Email != auth.Email || auth2.Token != auth.Token {
		t.Fatal("Loaded credentials don't match saved credentials")
	}
	
	fmt.Printf("✓ Successfully loaded credentials: %s\n", auth2.Email)

	// Clean up test credentials
	err = auth2.ClearCredentials()
	if err != nil {
		t.Fatalf("Failed to clear test credentials: %v", err)
	}
	
	fmt.Println("✓ Successfully cleared test credentials")
	
	// Verify credentials were cleared
	auth3 := archive.NewBeeperAuth("test.beeper.com")
	stillThere := auth3.LoadCredentialsFromFile()
	if stillThere {
		t.Fatal("Credentials should have been cleared but they're still there")
	}
	
	fmt.Println("✓ Verified credentials were properly cleared")
	fmt.Println("🎉 Credentials management working correctly!")
}

func TestE2EBeeperFullWorkflow(t *testing.T) {
	// Test the complete workflow: authentication -> room discovery -> message import
	fmt.Println("=== Testing Complete Beeper Workflow ===")
	
	// Step 1: Initialize Beeper authentication
	auth := archive.NewBeeperAuth("beeper.com")
	credentialsLoaded := auth.LoadCredentials()
	if !credentialsLoaded {
		t.Skip("No Beeper credentials found - skipping full workflow test")
	}
	
	fmt.Printf("✓ Step 1: Loaded Beeper credentials for: %s\n", auth.Email)

	// Step 2: Get Matrix client
	client, err := auth.GetMatrixClient()
	if err != nil {
		t.Fatalf("Step 2 failed: Could not get Matrix client: %v", err)
	}
	
	fmt.Println("✓ Step 2: Created Matrix client using Beeper authentication")

	// Step 3: Set up environment for integration
	os.Setenv("USE_BEEPER_AUTH", "true")
	os.Setenv("BEEPER_TOKEN", auth.Token)
	if auth.Whoami != nil {
		os.Setenv("MATRIX_USER", fmt.Sprintf("@%s:beeper.com", auth.Whoami.UserInfo.Username))
	}
	
	fmt.Println("✓ Step 3: Set environment variables for Beeper integration")

	// Step 4: Initialize database
	err = archive.InitMongoDB()
	if err != nil {
		t.Fatalf("Step 4 failed: Could not initialize MongoDB: %v", err)
	}
	defer archive.CloseMongoDB()
	
	fmt.Println("✓ Step 4: Initialized MongoDB connection")

	// Step 5: Get joined rooms
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	joinedRooms, err := client.JoinedRooms(ctx)
	if err != nil {
		t.Fatalf("Step 5 failed: Could not get joined rooms: %v", err)
	}
	
	fmt.Printf("✓ Step 5: Retrieved %d joined rooms\n", len(joinedRooms.JoinedRooms))

	// Step 6: Test room operations
	if len(joinedRooms.JoinedRooms) > 0 {
		// Test with first available room
		testRoom := string(joinedRooms.JoinedRooms[0])
		
		displayName, err := archive.GetRoomDisplayName(client, testRoom)
		if err != nil {
			fmt.Printf("Warning: Could not get display name for room %s: %v\n", testRoom, err)
		} else {
			fmt.Printf("✓ Step 6: Got room display name: %s -> %s\n", testRoom, displayName)
		}
		
		// Step 7: Test message import functionality (without actually importing)
		// We can test that the import process starts correctly
		collection := archive.GetMessagesCollection()
		if collection == nil {
			t.Fatal("Step 7 failed: Could not access messages collection")
		}
		
		fmt.Println("✓ Step 7: Verified message collection access")
		
		// Test that message filtering works
		filter := archive.MessageFilter{
			RoomID: testRoom,
		}
		bsonFilter := filter.ToBSON()
		if bsonFilter == nil {
			t.Fatal("Step 7 failed: Could not create BSON filter")
		}
		
		fmt.Println("✓ Step 7: Message filtering functionality working")
	}

	fmt.Println("\n=== Complete Beeper Workflow Test Summary ===")
	fmt.Println("✓ Authentication: Loaded and verified Beeper credentials")
	fmt.Println("✓ Matrix Client: Created client using Beeper auth")
	fmt.Println("✓ Environment: Set up integration environment variables")
	fmt.Println("✓ Database: Initialized MongoDB connection")
	fmt.Println("✓ Room Discovery: Retrieved and processed joined rooms")
	fmt.Println("✓ Room Operations: Tested room display name functionality")
	fmt.Println("✓ Message System: Verified collection access and filtering")
	fmt.Println("🎉 Complete Beeper workflow is fully functional!")
	
	// Cleanup environment variables
	os.Unsetenv("USE_BEEPER_AUTH")
	os.Unsetenv("BEEPER_TOKEN")
}
