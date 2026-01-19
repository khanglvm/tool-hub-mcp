package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

// NewRemoveCmd creates the 'remove' command for removing MCP servers.
func NewRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove an MCP server",
		Long:    `Remove an MCP server from the configuration.`,
		Example: `  tool-hub-mcp remove jira
  tool-hub-mcp rm jira`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(args[0])
		},
	}

	return cmd
}

// runRemove removes an MCP server from the configuration.
func runRemove(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Try both original name and camelCase
	camelName := config.ToCamelCase(name)
	
	if _, exists := cfg.Servers[name]; exists {
		delete(cfg.Servers, name)
	} else if _, exists := cfg.Servers[camelName]; exists {
		delete(cfg.Servers, camelName)
	} else {
		return fmt.Errorf("server '%s' not found", name)
	}

	configPath, err := config.GetDefaultConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Removed server '%s'\n", name)
	return nil
}
