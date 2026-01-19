/*
Package benchmark provides token consumption benchmarking for tool-hub-mcp.

It compares context token consumption between:
1. Traditional MCP: Multiple individual servers with all their tools
2. tool-hub-mcp: Single aggregator with 5 meta-tools

Token estimation uses tiktoken-compatible counting (GPT-4/Claude approximation:
~4 characters per token for English text, ~3 for JSON/code).
*/
package benchmark

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

// TokenEstimate represents token consumption estimates.
type TokenEstimate struct {
	ServerCount      int    `json:"serverCount"`
	ToolCount        int    `json:"toolCount"`
	DefinitionTokens int    `json:"definitionTokens"`
	Description      string `json:"description"`
}

// BenchmarkResult contains comparison results.
type BenchmarkResult struct {
	Traditional   TokenEstimate `json:"traditional"`
	ToolHub       TokenEstimate `json:"toolHub"`
	TokenSavings  int           `json:"tokenSavings"`
	SavingsPercent float64      `json:"savingsPercent"`
}

// AverageToolsPerServer is the estimated number of tools per MCP server.
// Based on analysis of popular MCP servers:
// - Jira MCP: 13 tools
// - Outline MCP: 30+ tools
// - Figma MCP: 2-5 tools
// - Sequential Thinking: 1 tool
// Average: ~10 tools per server
const AverageToolsPerServer = 10

// AverageTokensPerTool is the estimated tokens per tool definition.
// A typical tool definition includes:
// - name: ~5 tokens
// - description: ~50 tokens
// - inputSchema: ~100 tokens (properties, types, descriptions)
// Total: ~150 tokens per tool
const AverageTokensPerTool = 150

// ToolHubTools is the fixed number of meta-tools exposed by tool-hub-mcp.
const ToolHubTools = 5

// RunBenchmark compares token consumption between traditional and tool-hub-mcp setups.
func RunBenchmark(cfg *config.Config) *BenchmarkResult {
	serverCount := len(cfg.Servers)
	
	// Estimate traditional setup
	traditionalTools := serverCount * AverageToolsPerServer
	traditionalTokens := traditionalTools * AverageTokensPerTool
	
	traditional := TokenEstimate{
		ServerCount:      serverCount,
		ToolCount:        traditionalTools,
		DefinitionTokens: traditionalTokens,
		Description:      fmt.Sprintf("%d MCP servers × ~%d tools/server", serverCount, AverageToolsPerServer),
	}
	
	// tool-hub-mcp setup (fixed 5 meta-tools)
	toolHubTokens := ToolHubTools * AverageTokensPerTool
	
	toolHub := TokenEstimate{
		ServerCount:      1,
		ToolCount:        ToolHubTools,
		DefinitionTokens: toolHubTokens,
		Description:      "1 tool-hub-mcp server with 5 meta-tools",
	}
	
	// Calculate savings
	savings := traditionalTokens - toolHubTokens
	savingsPercent := float64(savings) / float64(traditionalTokens) * 100
	
	return &BenchmarkResult{
		Traditional:    traditional,
		ToolHub:        toolHub,
		TokenSavings:   savings,
		SavingsPercent: savingsPercent,
	}
}

// GetToolHubToolDefinitions returns the actual tool definitions used by tool-hub-mcp.
func GetToolHubToolDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "hub_list",
			"description": "List all registered MCP servers in tool-hub-mcp",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "hub_discover",
			"description": "Get tool definitions from a specific MCP server. Use this to see what tools are available on a server before executing them.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Name of the server (use hub_list to see available servers)",
					},
				},
				"required": []string{"server"},
			},
		},
		{
			"name":        "hub_search",
			"description": "Search for tools across all registered MCP servers using keywords",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (e.g., 'create issue', 'search documents')",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "hub_execute",
			"description": "Execute a tool from a specific MCP server",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Name of the server",
					},
					"tool": map[string]interface{}{
						"type":        "string",
						"description": "Name of the tool to execute",
					},
					"arguments": map[string]interface{}{
						"type":        "object",
						"description": "Arguments to pass to the tool",
					},
				},
				"required": []string{"server", "tool"},
			},
		},
		{
			"name":        "hub_help",
			"description": "Get detailed help and schema for a specific tool",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Name of the server",
					},
					"tool": map[string]interface{}{
						"type":        "string",
						"description": "Name of the tool",
					},
				},
				"required": []string{"server", "tool"},
			},
		},
	}
}

// CountTokens estimates token count for a JSON structure.
// Uses approximation: ~3 characters per token for JSON/code.
func CountTokens(v interface{}) int {
	data, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	// JSON/code is more token-dense than natural language
	// Approximate: 3 characters per token
	return len(data) / 3
}

// CountActualToolHubTokens counts actual tokens in tool-hub-mcp definitions.
func CountActualToolHubTokens() int {
	tools := GetToolHubToolDefinitions()
	return CountTokens(tools)
}

// FormatResult formats the benchmark result for display.
func FormatResult(result *BenchmarkResult) string {
	var sb strings.Builder

	sb.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║           TOKEN EFFICIENCY BENCHMARK RESULTS                 ║\n")
	sb.WriteString("╠══════════════════════════════════════════════════════════════╣\n")
	sb.WriteString("║                                                              ║\n")
	sb.WriteString(fmt.Sprintf("║  📊 TRADITIONAL MCP SETUP                                    ║\n"))
	sb.WriteString(fmt.Sprintf("║     Servers: %-3d                                             ║\n", result.Traditional.ServerCount))
	sb.WriteString(fmt.Sprintf("║     Tools:   ~%-3d (estimated)                                ║\n", result.Traditional.ToolCount))
	sb.WriteString(fmt.Sprintf("║     Tokens:  ~%d                                          ║\n", result.Traditional.DefinitionTokens))
	sb.WriteString("║                                                              ║\n")
	sb.WriteString("╠══════════════════════════════════════════════════════════════╣\n")
	sb.WriteString("║                                                              ║\n")
	sb.WriteString(fmt.Sprintf("║  🚀 TOOL-HUB-MCP SETUP                                       ║\n"))
	sb.WriteString(fmt.Sprintf("║     Servers: %-3d                                             ║\n", result.ToolHub.ServerCount))
	sb.WriteString(fmt.Sprintf("║     Tools:   %-3d (meta-tools)                                ║\n", result.ToolHub.ToolCount))
	sb.WriteString(fmt.Sprintf("║     Tokens:  ~%d                                            ║\n", result.ToolHub.DefinitionTokens))
	sb.WriteString("║                                                              ║\n")
	sb.WriteString("╠══════════════════════════════════════════════════════════════╣\n")
	sb.WriteString("║                                                              ║\n")
	sb.WriteString(fmt.Sprintf("║  💰 SAVINGS                                                  ║\n"))
	sb.WriteString(fmt.Sprintf("║     Tokens saved: ~%d                                      ║\n", result.TokenSavings))
	sb.WriteString(fmt.Sprintf("║     Reduction:    %.1f%%                                      ║\n", result.SavingsPercent))
	sb.WriteString("║                                                              ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════╝\n")

	return sb.String()
}
