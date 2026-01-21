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
	"github.com/khanglvm/tool-hub-mcp/internal/version"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "tool-hub-mcp",
		Short: "Serverless MCP aggregator - reduce context tokens by 96%",
		Long: `tool-hub-mcp is a serverless MCP (Model Context Protocol) aggregator
that solves the context token consumption problem when using multiple MCP servers.

Instead of exposing dozens of individual MCP tools (consuming 60k+ tokens),
it provides a single unified MCP endpoint with 2 meta-tools:
  • hub_search   - Semantic search for tools across all servers
  • hub_execute  - Execute a tool from a server

Token savings: 38% reduction (2 meta-tools vs 100+ individual tools)`,
		Version: version.GetVersion(),
	}

	// Add subcommands
	rootCmd.AddCommand(cli.NewSetupCmd())
	rootCmd.AddCommand(cli.NewVersionCmd())
	rootCmd.AddCommand(cli.NewServeCmd())
	rootCmd.AddCommand(cli.NewAddCmd())
	rootCmd.AddCommand(cli.NewRemoveCmd())
	rootCmd.AddCommand(cli.NewListCmd())
	rootCmd.AddCommand(cli.NewVerifyCmd())
	
	// Benchmark command with speed subcommand
	benchmarkCmd := cli.NewBenchmarkCmd()
	benchmarkCmd.AddCommand(cli.NewSpeedBenchmarkCmd())
	rootCmd.AddCommand(benchmarkCmd)

	// Learning command group
	rootCmd.AddCommand(cli.NewLearningCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
