/*
Package cli provides commands for managing the learning system.

These commands allow users to view learning statistics, export usage data,
clear history, and toggle tracking on/off.
*/
package cli

import (
	"github.com/spf13/cobra"
)

// NewLearningCmd creates the learning command group.
func NewLearningCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "learning",
		Short: "Manage learning system (usage tracking and tool ranking)",
		Long: `The learning system tracks tool usage to provide intelligent ranking
and recommendations via ε-greedy multi-armed bandit algorithm.

All data is stored locally in ~/.tool-hub-mcp/history.db with privacy
protection (SHA256 hashing of contexts).

Commands:
  status  Show learning statistics and top tools
  export  Export usage history as JSON
  clear   Delete all learning data
  disable Turn off tracking (temporary)
  enable  Turn on tracking`,
	}

	cmd.AddCommand(newLearningStatusCmd())
	cmd.AddCommand(newLearningExportCmd())
	cmd.AddCommand(newLearningClearCmd())
	cmd.AddCommand(newLearningDisableCmd())
	cmd.AddCommand(newLearningEnableCmd())

	return cmd
}
