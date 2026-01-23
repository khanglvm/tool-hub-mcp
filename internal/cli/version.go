/*
Package cli implements the version command for tool-hub-mcp.

The version command displays version, commit, and build date information.
*/
package cli

import (
	"fmt"

	"github.com/khanglvm/tool-hub-mcp/internal/version"
	"github.com/spf13/cobra"
)

// NewVersionCmd creates the 'version' command
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the current version, commit hash, and build date.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion()
		},
	}

	return cmd
}

func runVersion() error {
	v, c, d := version.GetVersionComponents()
	fmt.Printf("Version:  %s\n", v)
	fmt.Printf("Commit:   %s\n", c)
	fmt.Printf("Built:    %s\n", d)
	return nil
}
