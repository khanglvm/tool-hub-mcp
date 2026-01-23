package cli

import (
	"fmt"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/benchmark"
	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/khanglvm/tool-hub-mcp/internal/spawner"
	"github.com/spf13/cobra"
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
		fmt.Printf("║     Tools:   %-3d (actual/estimated per server)               ║\n", result.Traditional.ToolCount)
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

// NewSpeedBenchmarkCmd creates the 'benchmark speed' command for latency testing.
func NewSpeedBenchmarkCmd() *cobra.Command {
	var iterations int

	cmd := &cobra.Command{
		Use:   "speed",
		Short: "Measure tool-hub-mcp internal latency",
		Long: `Measure the time it takes for tool-hub-mcp to:
1. Spawn a child MCP process
2. Send a request (tools/list)
3. Receive and parse the response

This helps understand the overhead added by the aggregator pattern.`,
		Example: `  # Run speed benchmark
  tool-hub-mcp benchmark speed

  # Run with more iterations
  tool-hub-mcp benchmark speed --iterations 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSpeedBenchmark(iterations)
		},
	}

	cmd.Flags().IntVarP(&iterations, "iterations", "n", 3, "Number of iterations per server")

	return cmd
}

// runSpeedBenchmark measures internal latency for spawning and querying MCP servers.
func runSpeedBenchmark(iterations int) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Servers) == 0 {
		return fmt.Errorf("no servers configured")
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              SPEED BENCHMARK (Internal Latency)              ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Iterations per server: %-3d                                  ║\n", iterations)
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	pool := spawner.NewPool(5)
	totalTime := time.Duration(0)
	successCount := 0

	for name, serverCfg := range cfg.Servers {
		fmt.Printf("Testing: %s\n", name)

		var serverTotalTime time.Duration
		var serverSuccess int

		for i := 0; i < iterations; i++ {
			start := time.Now()

			tools, err := pool.GetTools(name, serverCfg)
			elapsed := time.Since(start)

			if err != nil {
				fmt.Printf("  Run %d: ERROR - %v\n", i+1, err)
				continue
			}

			serverTotalTime += elapsed
			serverSuccess++
			fmt.Printf("  Run %d: %v (%d tools discovered)\n", i+1, elapsed.Round(time.Millisecond), len(tools))
		}

		if serverSuccess > 0 {
			avgTime := serverTotalTime / time.Duration(serverSuccess)
			fmt.Printf("  Average: %v\n", avgTime.Round(time.Millisecond))
			totalTime += serverTotalTime
			successCount += serverSuccess
		}
		fmt.Println()
	}

	if successCount > 0 {
		overallAvg := totalTime / time.Duration(successCount)
		fmt.Println("═══════════════════════════════════════════════════════════════")
		fmt.Printf("Overall Average Latency: %v\n", overallAvg.Round(time.Millisecond))
		fmt.Println("═══════════════════════════════════════════════════════════════")
	}

	return nil
}
