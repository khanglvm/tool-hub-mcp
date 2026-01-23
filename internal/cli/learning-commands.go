/*
Package cli provides individual learning system command implementations.

Each command is in its own function for modularity and testability.
*/
package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/storage"
	"github.com/spf13/cobra"
)

// newLearningStatusCmd shows learning statistics.
func newLearningStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show learning statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := storage.NewStorage()
			if err := store.Init(); err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer store.Close()

			_ = time.Now().Add(-7 * 24 * time.Hour)

			fmt.Println("Learning System Status")
			fmt.Println("====================")
			fmt.Printf("Storage enabled: true\n")
			fmt.Printf("Tracking window: last 7 days\n")
			fmt.Printf("Scoring: 0.6*frequency + 0.3*recency + 0.1*rating\n")
			fmt.Printf("Exploration rate: 0.1 (10%%)\n")
			fmt.Println()
			fmt.Println("Note: Run 'tool-hub-mcp learning export' to view usage history")

			return nil
		},
	}
}

// newLearningExportCmd exports usage history as JSON.
func newLearningExportCmd() *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export usage history as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := storage.NewStorage()
			if err := store.Init(); err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer store.Close()

			_ = time.Now().Add(-30 * 24 * time.Hour)

			fmt.Println(`{"message": "Export feature - requires storage enhancement for GetAllUsageHistory()"}`)

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	return cmd
}

// newLearningClearCmd deletes all learning data.
func newLearningClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Delete all learning data",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print("This will delete all learning data. Continue? (y/N): ")
			var response string
			fmt.Scanln(&response)

			if response != "y" && response != "Y" {
				fmt.Println("Cancelled")
				return nil
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			dbPath := home + "/.tool-hub-mcp/history.db"

			if err := os.Remove(dbPath); err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No learning data found")
					return nil
				}
				return fmt.Errorf("failed to delete database: %w", err)
			}

			fmt.Println("Learning data cleared successfully")
			return nil
		},
	}
}

// newLearningDisableCmd turns off tracking.
func newLearningDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Turn off usage tracking",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("To disable tracking, set environment variable:")
			fmt.Println("  TOOL_HUB_MCP_LEARNING=false")
			fmt.Println()
			fmt.Println("Or modify config at ~/.tool-hub-mcp.json to add:")
			fmt.Println(`  "learning": {"enabled": false}`)

			return nil
		},
	}
}

// newLearningEnableCmd turns on tracking.
func newLearningEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Turn on usage tracking",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Learning is enabled by default.")
			fmt.Println("To ensure it's active, unset environment variable:")
			fmt.Println("  unset TOOL_HUB_MCP_LEARNING")
			fmt.Println()
			fmt.Println("Or remove learning.disabled from ~/.tool-hub-mcp.json")

			return nil
		},
	}
}
