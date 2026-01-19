package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/khanglvm/tool-hub-mcp/internal/mcp"
)

// NewServeCmd creates the 'serve' command for running the MCP server.
//
// This is the main command that exposes the 5 meta-tools via stdio transport:
// - hub_list, hub_discover, hub_search, hub_execute, hub_help
func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server (stdio transport)",
		Long: `Start the tool-hub-mcp server using stdio transport.

This server exposes 5 meta-tools to AI clients:
  • hub_list     - List all registered MCP servers
  • hub_discover - Get tool definitions from a specific server
  • hub_search   - Semantic search for tools across all servers
  • hub_execute  - Execute a tool from a specific server
  • hub_help     - Get detailed help/schema for a tool

The server spawns child MCP servers on-demand when tools are executed.`,
		Example: `  # Run directly
  tool-hub-mcp serve

  # Add to Claude Code
  claude mcp add tool-hub -- tool-hub-mcp serve`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe()
		},
	}

	return cmd
}

// runServe starts the MCP server with stdio transport.
func runServe() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create and run MCP server
	server := mcp.NewServer(cfg)
	return server.Run()
}
