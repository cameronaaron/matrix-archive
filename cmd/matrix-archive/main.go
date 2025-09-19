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
	rootCmd.AddCommand(keyRecoveryCmd)

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
		if err := archive.ListRooms(pattern); err != nil {
			log.Fatal(err)
		}
	},
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import messages from Matrix rooms into the database",
	Long:  "Import messages from Matrix rooms into DuckDB for archival. If no room ID is specified, imports from all joined rooms.",
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
	Long: `Export messages from the database to various formats based on file extension:
- .html: HTML format
- .txt: Plain text format
- .json: JSON format
- .yaml: YAML format`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		roomID, _ := cmd.Flags().GetString("room-id")
		localImages, _ := cmd.Flags().GetBool("local-images")
		if err := archive.ExportMessages(args[0], roomID, localImages); err != nil {
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
		if err := archive.DownloadImages(outputDir, thumbnails); err != nil {
			log.Fatal(err)
		}
	},
}

var beeperLoginCmd = &cobra.Command{
	Use:   "beeper-login",
	Short: "Authenticate with Beeper",
	Long:  "Authenticate with Beeper using email and passcode.",
	Run: func(cmd *cobra.Command, args []string) {
		domain, _ := cmd.Flags().GetString("domain")
		if err := archive.PerformBeeperLogin(domain, false); err != nil {
			log.Fatal(err)
		}
	},
}

var beeperLogoutCmd = &cobra.Command{
	Use:   "beeper-logout",
	Short: "Clear Beeper credentials",
	Long:  "Clear stored Beeper credentials.",
	Run: func(cmd *cobra.Command, args []string) {
		domain, _ := cmd.Flags().GetString("domain")
		if err := archive.PerformBeeperLogout(domain); err != nil {
			log.Fatal(err)
		}
	},
}

var keyRecoveryCmd = &cobra.Command{
	Use:   "key-recovery",
	Short: "Recover encryption keys using Matrix key backup",
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
	downloadImagesCmd.Flags().Bool("thumbnails", true, "Download thumbnails instead of full images")
	beeperLoginCmd.Flags().String("domain", "beeper.com", "Beeper domain to authenticate with")
	beeperLogoutCmd.Flags().String("domain", "beeper.com", "Beeper domain to clear credentials for")
	keyRecoveryCmd.Flags().String("recovery-key", "", "Matrix key backup recovery key (required)")
	keyRecoveryCmd.Flags().String("room-id", "", "Specific room ID to decrypt messages for (optional)")
}
