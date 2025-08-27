package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"text/tabwriter"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// listRooms lists all rooms the user has access to, optionally filtered by pattern
func listRooms(pattern string) error {
	client, err := GetMatrixClient()
	if err != nil {
		return fmt.Errorf("failed to get Matrix client: %w", err)
	}

	// Get joined rooms
	resp, err := client.JoinedRooms(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get joined rooms: %w", err)
	}

	// Compile pattern if provided
	var patternRegex *regexp.Regexp
	if pattern != "" {
		patternRegex, err = regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	// Create tabwriter for formatted output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Room ID\tDisplay Name")
	fmt.Fprintln(w, "-------\t------------")

	// Iterate through rooms
	fmt.Printf("Found %d joined rooms. Fetching room names...\n", len(resp.JoinedRooms))

	for i, roomID := range resp.JoinedRooms {
		// Get room state to get the name
		displayName, err := getRoomDisplayName(client, string(roomID))
		if err != nil {
			displayName = "Unknown"
		}

		// Apply pattern filter if specified
		if patternRegex != nil && !patternRegex.MatchString(displayName) {
			continue
		}

		fmt.Fprintf(w, "%s\t%s\n", roomID, displayName)

		// Show progress for large numbers of rooms
		if (i+1)%50 == 0 {
			fmt.Printf("Processed %d/%d rooms...\n", i+1, len(resp.JoinedRooms))
		}
	}

	w.Flush()
	return nil
}

// getRoomDisplayName gets the display name for a room
func getRoomDisplayName(client *mautrix.Client, roomID string) (string, error) {
	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get the room name from state
	var nameEvent event.RoomNameEventContent
	err := client.StateEvent(ctx, id.RoomID(roomID), event.StateRoomName, "", &nameEvent)
	if err == nil && nameEvent.Name != "" {
		return nameEvent.Name, nil
	}

	// If that fails, just return the room ID
	return roomID, nil
}
