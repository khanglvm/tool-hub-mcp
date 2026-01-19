package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

// NewVerifyCmd creates the 'verify' command for verifying configuration.
func NewVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify configuration and connections",
		Long: `Verify that the configuration is valid and optionally test
connections to registered MCP servers.`,
		Example: `  tool-hub-mcp verify`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerify()
		},
	}

	return cmd
}

// runVerify validates the configuration.
func runVerify() error {
	configPath, err := config.GetDefaultConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	fmt.Printf("✓ Config file: %s\n", configPath)
	fmt.Printf("✓ Servers registered: %d\n", len(cfg.Servers))

	// Validate each server
	for name, server := range cfg.Servers {
		if server.Command == "" {
			fmt.Printf("✗ %s: missing command\n", name)
		} else {
			fmt.Printf("✓ %s: %s\n", name, server.Command)
		}
	}

	return nil
}
