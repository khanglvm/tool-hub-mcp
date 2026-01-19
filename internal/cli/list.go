package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

// NewListCmd creates the 'list' command for listing registered MCP servers.
func NewListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all registered MCP servers",
		Long:    `Display all MCP servers registered in ~/.tool-hub-mcp.json`,
		Example: `  tool-hub-mcp list
  tool-hub-mcp ls
  tool-hub-mcp list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(jsonOutput)
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

// runList displays all registered MCP servers.
func runList(jsonOutput bool) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("No servers configured.")
		fmt.Println("Run 'tool-hub-mcp setup' to import from AI CLI tools.")
		return nil
	}

	if len(cfg.Servers) == 0 {
		fmt.Println("No servers configured.")
		fmt.Println("Run 'tool-hub-mcp setup' to import from AI CLI tools.")
		return nil
	}

	fmt.Printf("Registered MCP Servers (%d):\n\n", len(cfg.Servers))

	for name, server := range cfg.Servers {
		source := server.Source
		if source == "" {
			source = "unknown"
		}
		fmt.Printf("  %s\n", name)
		fmt.Printf("    Command: %s %v\n", server.Command, server.Args)
		fmt.Printf("    Source:  %s\n", source)
		if len(server.Env) > 0 {
			fmt.Printf("    Env:     %d variables\n", len(server.Env))
		}
		fmt.Println()
	}

	return nil
}
