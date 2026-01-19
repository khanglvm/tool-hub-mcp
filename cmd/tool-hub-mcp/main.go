/*
Package main is the entry point for tool-hub-mcp CLI.

tool-hub-mcp is a serverless MCP aggregator that reduces context token
consumption by exposing a single unified endpoint with meta-tools instead
of individual MCP servers.

Usage:
  tool-hub-mcp [command]

Available Commands:
  setup       Import MCP configurations from AI CLI tools
  serve       Run the MCP server (stdio transport)
  add         Add an MCP server manually
  remove      Remove an MCP server
  list        List all registered MCP servers
  verify      Verify configuration and connections
  help        Help about any command

Examples:
  # Import configs from Claude Code and OpenCode
  tool-hub-mcp setup

  # Run as MCP server
  tool-hub-mcp serve

  # Add a server manually
  tool-hub-mcp add jira --command "npx -y @lvmk/jira-mcp"
*/
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/khanglvm/tool-hub-mcp/internal/cli"
)

// Version information (set via ldflags during build)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "tool-hub-mcp",
		Short: "Serverless MCP aggregator - reduce context tokens by 96%",
		Long: `tool-hub-mcp is a serverless MCP (Model Context Protocol) aggregator 
that solves the context token consumption problem when using multiple MCP servers.

Instead of exposing dozens of individual MCP tools (consuming 60k+ tokens),
it provides a single unified MCP endpoint with 5 meta-tools:
  • hub_list     - List all registered servers
  • hub_discover - Get tools from a specific server
  • hub_search   - Semantic search for tools
  • hub_execute  - Execute a tool from a server
  • hub_help     - Get detailed help for a tool

Token savings: 96%+ reduction (5 meta-tools vs 100+ individual tools)`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Add subcommands
	rootCmd.AddCommand(cli.NewSetupCmd())
	rootCmd.AddCommand(cli.NewServeCmd())
	rootCmd.AddCommand(cli.NewAddCmd())
	rootCmd.AddCommand(cli.NewRemoveCmd())
	rootCmd.AddCommand(cli.NewListCmd())
	rootCmd.AddCommand(cli.NewVerifyCmd())
	
	// Benchmark command with speed subcommand
	benchmarkCmd := cli.NewBenchmarkCmd()
	benchmarkCmd.AddCommand(cli.NewSpeedBenchmarkCmd())
	rootCmd.AddCommand(benchmarkCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
