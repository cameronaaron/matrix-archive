package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

func main() {
	// Load environment variables from .env file if it exists
	godotenv.Load()

	var rootCmd = &cobra.Command{
		Use:   "matrix-archive",
		Short: "Matrix Archive Tools - Import and export messages from Matrix rooms",
		Long: `Matrix Archive Tools allows you to import messages from Matrix rooms
into a database and export them in various formats for archival and research purposes.

Use this responsibly and ethically. Don't re-publish people's messages
without their knowledge and consent.`,
	}

	rootCmd.AddCommand(listRoomsCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(downloadImagesCmd)
	rootCmd.AddCommand(beeperLoginCmd)
	rootCmd.AddCommand(beeperLogoutCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var listRoomsCmd = &cobra.Command{
	Use:   "list [pattern]",
	Short: "List room IDs and display names",
	Long:  "List all Matrix rooms that the user has access to, optionally filtered by a regex pattern.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pattern := ""
		if len(args) > 0 {
			pattern = args[0]
		}
		if err := listRooms(pattern); err != nil {
			log.Fatal(err)
		}
	},
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import messages from Matrix rooms into the database",
	Long:  "Import messages from the configured Matrix rooms into MongoDB for archival.",
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		if err := importMessages(limit); err != nil {
			log.Fatal(err)
		}
	},
}

var exportCmd = &cobra.Command{
	Use:   "export [filename]",
	Short: "Export messages to a file",
	Long: `Export messages from the database to various formats based on file extension:
- .html: HTML format
- .txt: Plain text format  
- .json: JSON format
- .yaml: YAML format`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := "archive.html"
		if len(args) > 0 {
			filename = args[0]
		}
		roomID, _ := cmd.Flags().GetString("room-id")
		localImages, _ := cmd.Flags().GetBool("local-images")

		if err := exportMessages(filename, roomID, localImages); err != nil {
			log.Fatal(err)
		}
	},
}

var downloadImagesCmd = &cobra.Command{
	Use:   "download-images [output-dir]",
	Short: "Download images from messages",
	Long:  "Download all images referenced in messages to a local directory.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		outputDir := ""
		if len(args) > 0 {
			outputDir = args[0]
		}
		thumbnails, _ := cmd.Flags().GetBool("thumbnails")

		if err := downloadImages(outputDir, thumbnails); err != nil {
			log.Fatal(err)
		}
	},
}

var beeperLoginCmd = &cobra.Command{
	Use:   "beeper-login",
	Short: "Authenticate with Beeper using email and passcode",
	Long: `Authenticate with Beeper using email and passcode authentication.
This will prompt for your email address and then send a login code to your email.
The credentials will be saved for future use.`,
	Run: func(cmd *cobra.Command, args []string) {
		domain, _ := cmd.Flags().GetString("domain")
		if domain == "" {
			domain = "beeper.com"
		}

		if err := performBeeperLogin(domain); err != nil {
			log.Fatal(err)
		}
	},
}

var beeperLogoutCmd = &cobra.Command{
	Use:   "beeper-logout",
	Short: "Clear saved Beeper authentication credentials",
	Long: `Clear all saved Beeper authentication credentials from both environment variables
and the persistent credentials file. You will need to run beeper-login again to authenticate.`,
	Run: func(cmd *cobra.Command, args []string) {
		domain, _ := cmd.Flags().GetString("domain")
		if domain == "" {
			domain = "beeper.com"
		}

		if err := performBeeperLogout(domain); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	importCmd.Flags().IntP("limit", "l", 0, "Limit the number of messages to import")

	exportCmd.Flags().StringP("room-id", "r", "", "Specific room ID to export")
	exportCmd.Flags().BoolP("local-images", "", true, "Use local image paths instead of Matrix URLs")

	downloadImagesCmd.Flags().BoolP("thumbnails", "t", true, "Download thumbnails instead of full images")

	beeperLoginCmd.Flags().StringP("domain", "d", "beeper.com", "Beeper domain to authenticate with")
	beeperLogoutCmd.Flags().StringP("domain", "d", "beeper.com", "Beeper domain to clear credentials for")
}

// performBeeperLogin handles the Beeper login process
func performBeeperLogin(domain string) error {
	auth := NewBeeperAuth(domain)

	fmt.Printf("Starting Beeper authentication for domain: %s\n", domain)

	if err := auth.Login(); err != nil {
		return fmt.Errorf("beeper login failed: %w", err)
	}

	// Save credentials
	auth.SaveCredentials()

	// Test Matrix connection (this will cache the Matrix token)
	fmt.Println("Testing Matrix connection...")
	client, err := auth.GetMatrixClient()
	if err != nil {
		return fmt.Errorf("failed to create Matrix client: %w", err)
	}

	whoami, err := client.Whoami(context.Background())
	if err != nil {
		return fmt.Errorf("failed to verify Matrix connection: %w", err)
	}

	// Save credentials again to include the Matrix token
	auth.SaveCredentials()

	fmt.Printf("Successfully authenticated! Matrix User ID: %s\n", whoami.UserID)
	fmt.Println("Credentials saved. You can now use other commands with Beeper authentication.")
	fmt.Println("Set USE_BEEPER_AUTH=true to use Beeper authentication by default.")

	return nil
}

// performBeeperLogout handles the Beeper logout process
func performBeeperLogout(domain string) error {
	auth := NewBeeperAuth(domain)

	// Clear environment variables
	os.Unsetenv("BEEPER_ACCESS_TOKEN")
	os.Unsetenv("BEEPER_USER_ID")
	os.Unsetenv("BEEPER_DEVICE_ID")
	os.Unsetenv("BEEPER_HOMESERVER")
	os.Unsetenv("BEEPER_TOKEN")
	os.Unsetenv("BEEPER_EMAIL")
	os.Unsetenv("BEEPER_USERNAME")

	// Clear saved credentials file
	if err := auth.ClearCredentials(); err != nil {
		return fmt.Errorf("failed to clear saved credentials: %w", err)
	}

	fmt.Println("Successfully logged out of Beeper.")
	fmt.Println("All credentials have been cleared.")
	return nil
}
