package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/khanglvm/tool-hub-mcp/internal/spawner"
)

// NewListCmd creates the 'list' command for listing registered MCP servers.
func NewListCmd() *cobra.Command {
	var jsonOutput bool
	var showStatus bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all registered MCP servers",
		Long:    `Display all MCP servers registered in ~/.tool-hub-mcp.json`,
		Example: `  tool-hub-mcp list
  tool-hub-mcp ls
  tool-hub-mcp list --status  # test connections and show tool counts
  tool-hub-mcp list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(jsonOutput, showStatus)
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	cmd.Flags().BoolVarP(&showStatus, "status", "s", false, "Test connections and show tool counts")

	return cmd
}

// runList displays all registered MCP servers.
func runList(jsonOutput, showStatus bool) error {
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

	// Create spawner pool if status check requested
	var pool *spawner.Pool
	if showStatus {
		pool = spawner.NewPool(3)
	}

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

		// Check status if requested
		if showStatus {
			tools, err := pool.GetTools(name, server)
			if err != nil {
				fmt.Printf("    Status:  ✗ %s\n", err.Error())
			} else {
				fmt.Printf("    Status:  ✓ %d tools\n", len(tools))
			}
		}

		fmt.Println()
	}

	return nil
}
