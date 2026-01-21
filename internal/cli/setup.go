/*
Package cli implements the command-line interface for tool-hub-mcp.

Each command is implemented as a separate function that returns a *cobra.Command,
allowing for clean separation and easy testing.
*/
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/khanglvm/tool-hub-mcp/internal/config/sources"
)

// NewSetupCmd creates the 'setup' command for importing MCP configurations.
//
// The setup wizard:
// 1. Scans for AI CLI tools (Claude Code, OpenCode, etc.)
// 2. Presents found configurations for selection
// 3. Imports and transforms selected configs to unified camelCase format
// 4. Saves to ~/.tool-hub-mcp.json
func NewSetupCmd() *cobra.Command {
	var nonInteractive bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Import MCP configurations from AI CLI tools",
		Long: `Scan for AI CLI tools (Claude Code, OpenCode, etc.) and import their
MCP server configurations into tool-hub-mcp.

The setup wizard will:
  1. Detect installed AI CLI tools
  2. Parse their MCP configurations
  3. Transform to unified camelCase format
  4. Save to ~/.tool-hub-mcp.json

Supported sources:
  • Claude Code (~/.claude.json, .mcp.json)
  • OpenCode (~/.opencode.json, opencode.json)
  • Google Antigravity (~/.gemini/antigravity/mcp_config.json)
  • Gemini CLI (~/.gemini/settings.json)
  • Cursor (~/.cursor/mcp.json)
  • Windsurf (~/.codeium/windsurf/mcp_config.json)`,
		Example: `  # Interactive setup
  tool-hub-mcp setup

  # Non-interactive (import all found configs)
  tool-hub-mcp setup --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup(nonInteractive)
		},
	}

	cmd.Flags().BoolVarP(&nonInteractive, "yes", "y", false, "Non-interactive mode (import all)")

	return cmd
}

// runSetup executes the setup wizard logic.
func runSetup(nonInteractive bool) error {
	fmt.Println("🔍 Scanning for AI CLI tools...")
	fmt.Println()

	// Scan all config sources
	allSources := sources.GetAllSources()
	foundConfigs := make(map[string]*sources.SourceResult)

	for _, source := range allSources {
		result, err := source.Scan()
		if err != nil {
			// Log but continue - source might not be installed
			continue
		}
		if result != nil && len(result.Servers) > 0 {
			foundConfigs[source.Name()] = result
			fmt.Printf("  ✓ %s (%s) - %d MCP servers\n", 
				source.Name(), result.ConfigPath, len(result.Servers))
		}
	}

	if len(foundConfigs) == 0 {
		fmt.Println("  No MCP configurations found.")
		fmt.Println()
		fmt.Println("You can add servers manually with:")
		fmt.Println("  tool-hub-mcp add <name> --command <cmd>")
		return nil
	}

	fmt.Println()

	// Merge all configs
	mergedConfig := config.NewConfig()
	totalImported := 0
	skippedCount := 0
	skipReasons := make(map[string]int)

	for sourceName, result := range foundConfigs {
		for name, server := range result.Servers {
			// Transform server name to camelCase
			camelName := config.ToCamelCase(name)

			// Validation 1: Self-reference check
			if config.IsSelfReference(server) {
				skipReasons["self-reference"]++
				skippedCount++
				continue
			}

			// Validation 2: Empty command check
			if server.Command == "" {
				fmt.Printf("  ⚠️  Skipping %s: empty command\n", camelName)
				skipReasons["empty-command"]++
				skippedCount++
				continue
			}

			// Validation 3: Duplicate name check
			if _, exists := mergedConfig.Servers[camelName]; exists {
				fmt.Printf("  ⚠️  Server '%s' already exists, skipping\n", camelName)
				skipReasons["duplicate"]++
				skippedCount++
				continue
			}

			// Add source metadata
			server.Source = sourceName

			mergedConfig.Servers[camelName] = server
			totalImported++
		}
	}

	// Save config
	configPath, err := config.GetDefaultConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	if err := config.Save(mergedConfig, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Imported %d MCP servers to %s\n", totalImported, configPath)

	// Show skip summary
	if skippedCount > 0 {
		fmt.Printf("\nℹ️  Skipped %d servers:\n", skippedCount)
		for reason, count := range skipReasons {
			fmt.Printf("   - %s: %d\n", reason, count)
		}
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  Add tool-hub-mcp to your AI client:")
	fmt.Println()
	fmt.Println("  Claude Code:")
	fmt.Println("    claude mcp add tool-hub-mcp -- tool-hub-mcp serve")
	fmt.Println()
	fmt.Println("  OpenCode:")
	fmt.Println("    opencode mcp add tool-hub-mcp --command \"tool-hub-mcp serve\"")

	return nil
}
