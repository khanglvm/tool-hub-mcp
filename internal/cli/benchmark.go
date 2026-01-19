package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/khanglvm/tool-hub-mcp/internal/benchmark"
	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

// NewBenchmarkCmd creates the 'benchmark' command for token efficiency testing.
func NewBenchmarkCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "benchmark",
		Short: "Compare token consumption: traditional MCP vs tool-hub-mcp",
		Long: `Run a token efficiency benchmark comparing:

TRADITIONAL SETUP:
  Each MCP server exposes all its tools directly to the AI client.
  With N servers × ~10 tools/server × ~150 tokens/tool = massive token overhead.

TOOL-HUB-MCP SETUP:
  Single aggregator exposing only 5 meta-tools.
  AI discovers and executes tools on-demand via hub_* commands.

The benchmark estimates token savings based on your registered servers.`,
		Example: `  # Run benchmark with current config
  tool-hub-mcp benchmark

  # Output as JSON
  tool-hub-mcp benchmark --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchmark(jsonOutput)
		},
	}

	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")

	return cmd
}

// runBenchmark executes the token efficiency benchmark.
func runBenchmark(jsonOutput bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nRun 'tool-hub-mcp setup' first", err)
	}

	if len(cfg.Servers) == 0 {
		return fmt.Errorf("no servers configured. Run 'tool-hub-mcp setup' or 'tool-hub-mcp add' first")
	}

	// Run benchmark
	result := benchmark.RunBenchmark(cfg)

	// Also get actual token count for tool-hub-mcp definitions
	actualToolHubTokens := benchmark.CountActualToolHubTokens()

	if jsonOutput {
		// JSON output
		fmt.Printf(`{
  "traditional": {
    "servers": %d,
    "estimatedTools": %d,
    "estimatedTokens": %d
  },
  "toolHub": {
    "servers": 1,
    "tools": 5,
    "actualTokens": %d
  },
  "savings": {
    "tokens": %d,
    "percent": %.1f
  }
}
`, result.Traditional.ServerCount, result.Traditional.ToolCount, result.Traditional.DefinitionTokens,
			actualToolHubTokens, result.TokenSavings, result.SavingsPercent)
	} else {
		// Pretty output
		fmt.Println()
		fmt.Println("╔══════════════════════════════════════════════════════════════╗")
		fmt.Println("║           TOKEN EFFICIENCY BENCHMARK RESULTS                 ║")
		fmt.Println("╠══════════════════════════════════════════════════════════════╣")
		fmt.Println("║                                                              ║")
		fmt.Println("║  📊 TRADITIONAL MCP SETUP                                    ║")
		fmt.Printf("║     Servers: %-3d                                             ║\n", result.Traditional.ServerCount)
		fmt.Printf("║     Tools:   ~%-3d (estimated: %d servers × 10 tools)         ║\n", result.Traditional.ToolCount, result.Traditional.ServerCount)
		fmt.Printf("║     Tokens:  ~%-6d                                         ║\n", result.Traditional.DefinitionTokens)
		fmt.Println("║                                                              ║")
		fmt.Println("╠══════════════════════════════════════════════════════════════╣")
		fmt.Println("║                                                              ║")
		fmt.Println("║  🚀 TOOL-HUB-MCP SETUP                                       ║")
		fmt.Printf("║     Servers: %-3d                                             ║\n", result.ToolHub.ServerCount)
		fmt.Printf("║     Tools:   %-3d (hub_list, hub_discover, hub_search, ...)   ║\n", result.ToolHub.ToolCount)
		fmt.Printf("║     Tokens:  %-6d (actual)                                  ║\n", actualToolHubTokens)
		fmt.Println("║                                                              ║")
		fmt.Println("╠══════════════════════════════════════════════════════════════╣")
		fmt.Println("║                                                              ║")
		fmt.Println("║  💰 SAVINGS                                                  ║")
		fmt.Printf("║     Tokens saved:  ~%-6d                                    ║\n", result.TokenSavings)
		fmt.Printf("║     Reduction:     %.1f%%                                      ║\n", result.SavingsPercent)
		fmt.Println("║                                                              ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════╝")
		fmt.Println()

		// Show registered servers
		fmt.Printf("Servers included in benchmark (%d):\n", len(cfg.Servers))
		for name := range cfg.Servers {
			fmt.Printf("  • %s\n", name)
		}
		fmt.Println()
	}

	return nil
}
