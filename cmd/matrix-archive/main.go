package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	archive "github.com/osteele/matrix-archive/lib"
)

func main() {
	godotenv.Load()

	var rootCmd = &cobra.Command{
		Use:   "matrix-archive",
		Short: "Matrix Archive - Professional chat history management",
		Long: `Matrix Archive is a comprehensive tool for importing, exporting, and managing Matrix chat histories.

Features:
  • Import and decrypt E2EE messages from Matrix/Beeper
  • Export to multiple formats (HTML, JSON, YAML, TXT)  
  • Advanced username mapping for bridge users
  • Rich metadata extraction and professional templates
  • Secure credential and encryption key management

Usage Examples:
  matrix-archive auth login                    # Authenticate with Beeper
  matrix-archive import --room-id "!room:..."  # Import room messages
  matrix-archive export archive.html           # Export to HTML
  matrix-archive crypto recover-keys --recovery-key "key"  # Recover encryption keys`,
	}

	rootCmd.AddCommand(listRoomsCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(cryptoCmd)
	rootCmd.AddCommand(mediaCmd)

	// Add subcommands to groups
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	cryptoCmd.AddCommand(cryptoRecoverKeysCmd)
	mediaCmd.AddCommand(mediaDownloadCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var listRoomsCmd = &cobra.Command{
	Use:   "list",
	Short: "List available Matrix rooms",
	Long:  "Display all Matrix rooms with their IDs and display names. Supports pattern filtering.",
	Run: func(cmd *cobra.Command, args []string) {
		pattern, _ := cmd.Flags().GetString("pattern")
		if err := archive.ListRooms(pattern); err != nil {
			log.Fatal(err)
		}
	},
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import Matrix messages to local database",
	Long:  "Import and decrypt Matrix messages from specified rooms or all joined rooms. Supports E2EE message decryption.",
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		roomID, _ := cmd.Flags().GetString("room-id")
		if err := archive.ImportMessages(limit, roomID); err != nil {
			log.Fatal(err)
		}
	},
}

var exportCmd = &cobra.Command{
	Use:   "export [filename]",
	Short: "Export messages to various formats",
	Long:  "Export stored messages to HTML, JSON, YAML, or TXT format with rich metadata and professional templates. Supports advanced username mapping for bridge users.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		roomID, _ := cmd.Flags().GetString("room-id")
		localImages, _ := cmd.Flags().GetBool("local-images")
		if err := archive.ExportMessages(args[0], roomID, localImages); err != nil {
			log.Fatal(err)
		}
	},
}

var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "Media file management",
	Long:  "Download and manage media files from Matrix messages.",
}

var mediaDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download media files from messages",
	Long:  "Download images and other media files referenced in Matrix messages to local storage.",
	Run: func(cmd *cobra.Command, args []string) {
		thumbnails, _ := cmd.Flags().GetBool("thumbnails")
		if err := archive.DownloadImages("", thumbnails); err != nil {
			log.Fatal(err)
		}
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication management",
	Long:  "Manage authentication credentials for Matrix and Beeper services.",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Beeper",
	Long:  "Authenticate with Beeper using email and verification code.",
	Run: func(cmd *cobra.Command, args []string) {
		domain, _ := cmd.Flags().GetString("domain")
		if err := archive.PerformBeeperLogin(domain, false); err != nil {
			log.Fatal(err)
		}
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear authentication credentials",
	Long:  "Clear stored Beeper authentication credentials from local storage.",
	Run: func(cmd *cobra.Command, args []string) {
		domain, _ := cmd.Flags().GetString("domain")
		if err := archive.PerformBeeperLogout(domain); err != nil {
			log.Fatal(err)
		}
	},
}

var cryptoCmd = &cobra.Command{
	Use:   "crypto",
	Short: "Encryption and key management",
	Long:  "Manage encryption keys and cryptographic operations for Matrix message decryption.",
}

var cryptoRecoverKeysCmd = &cobra.Command{
	Use:   "recover-keys",
	Short: "Recover encryption keys from backup",
	Long:  "Recover encryption keys from Matrix key backup using a recovery key to decrypt historical messages.",
	Run: func(cmd *cobra.Command, args []string) {
		recoveryKey, _ := cmd.Flags().GetString("recovery-key")
		roomID, _ := cmd.Flags().GetString("room-id")

		if recoveryKey == "" {
			log.Fatal("Recovery key is required. Use --recovery-key flag.")
		}

		if err := archive.PerformKeyRecovery(recoveryKey, roomID); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	importCmd.Flags().Int("limit", 0, "Limit the number of messages to import (0 = no limit)")
	importCmd.Flags().String("room-id", "", "Import from a specific room (optional, imports all joined rooms if not specified)")
	exportCmd.Flags().String("room-id", "", "Export from a specific room (optional)")
	exportCmd.Flags().Bool("local-images", true, "Use local image paths instead of Matrix URLs")
	mediaDownloadCmd.Flags().Bool("thumbnails", true, "Download thumbnails instead of full images")
	authLoginCmd.Flags().String("domain", "beeper.com", "Beeper domain to authenticate with")
	authLogoutCmd.Flags().String("domain", "beeper.com", "Beeper domain to clear credentials for")
	cryptoRecoverKeysCmd.Flags().String("recovery-key", "", "Matrix key backup recovery key (required)")
	cryptoRecoverKeysCmd.Flags().String("room-id", "", "Specific room ID to decrypt messages for (optional)")
}
