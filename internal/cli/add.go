package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

// NewAddCmd creates the 'add' command for manually adding MCP servers.
//
// Supports two modes:
// 1. Interactive: Paste MCP config JSON, auto-detect format, preview, confirm
// 2. Flags: Specify --command, --arg, --env directly
func NewAddCmd() *cobra.Command {
	var (
		command     string
		args        []string
		envVars     []string
		jsonInput   string
		noConfirm   bool
	)

	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add MCP server(s) - paste config JSON or use flags",
		Long: `Add MCP server configuration(s) to tool-hub-mcp.

INTERACTIVE MODE (recommended):
  Paste any valid MCP configuration JSON. Supports formats from:
  • Claude Code (mcpServers)
  • OpenCode (mcp)
  • Single server object

  The tool will auto-detect the format, show you what will be added,
  and ask for confirmation before saving.

FLAG MODE:
  Specify server details directly with flags.`,
		Example: `  # Interactive mode - paste JSON when prompted
  tool-hub-mcp add

  # Paste JSON directly
  tool-hub-mcp add --json '{"jira": {"command": "npx", "args": ["-y", "@lvmk/jira-mcp"]}}'

  # Flag mode - specify details directly
  tool-hub-mcp add jira --command "npx" --arg "-y" --arg "@lvmk/jira-mcp"

  # Paste full Claude Code config
  tool-hub-mcp add --json '{
    "mcpServers": {
      "jira": {"command": "npx", "args": ["-y", "@lvmk/jira-mcp"]},
      "outline": {"command": "npx", "args": ["-y", "@outline/mcp"]}
    }
  }'`,
		RunE: func(cmd *cobra.Command, positionalArgs []string) error {
			// If JSON provided or no name, use interactive/JSON mode
			if jsonInput != "" || (len(positionalArgs) == 0 && command == "") {
				return runAddInteractive(jsonInput, noConfirm)
			}

			// Flag mode
			if len(positionalArgs) == 0 {
				return fmt.Errorf("server name required when using flag mode")
			}
			return runAddWithFlags(positionalArgs[0], command, args, envVars)
		},
	}

	cmd.Flags().StringVarP(&command, "command", "c", "", "Command to run the MCP server")
	cmd.Flags().StringArrayVarP(&args, "arg", "a", nil, "Arguments for the command")
	cmd.Flags().StringArrayVarP(&envVars, "env", "e", nil, "Environment variables (KEY=VALUE)")
	cmd.Flags().StringVarP(&jsonInput, "json", "j", "", "MCP config JSON (auto-detect format)")
	cmd.Flags().BoolVarP(&noConfirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

// runAddInteractive handles JSON input mode with preview and confirmation.
func runAddInteractive(jsonInput string, noConfirm bool) error {
	var input string

	if jsonInput != "" {
		input = jsonInput
	} else {
		// Prompt for input
		fmt.Println("📋 Paste your MCP configuration JSON (press Enter twice when done):")
		fmt.Println("   Supports: Claude Code, OpenCode, or single server format")
		fmt.Println()

		input = readMultilineInput()
		if strings.TrimSpace(input) == "" {
			return fmt.Errorf("no input provided")
		}
	}

	// Parse and detect format
	servers, format, err := parseAnyMCPConfig(input)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if len(servers) == 0 {
		return fmt.Errorf("no valid MCP servers found in input")
	}

	// Show preview
	fmt.Println()
	fmt.Printf("🔍 Detected format: %s\n", format)
	fmt.Printf("📦 Found %d server(s):\n\n", len(servers))

	for name, server := range servers {
		camelName := config.ToCamelCase(name)
		fmt.Printf("  %s", colorGreen(camelName))
		if camelName != name {
			fmt.Printf(" (from '%s')", name)
		}
		fmt.Println()
		fmt.Printf("    Command: %s %v\n", server.Command, server.Args)
		if len(server.Env) > 0 {
			fmt.Printf("    Env:     %d variable(s)\n", len(server.Env))
		}
		fmt.Println()
	}

	// Confirm
	if !noConfirm {
		fmt.Print("Add these servers? [Y/n] ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		
		if response != "" && response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Load existing config
	cfg, err := config.Load()
	if err != nil {
		cfg = config.NewConfig()
	}

	// Add servers
	for name, server := range servers {
		camelName := config.ToCamelCase(name)
		server.Source = "manual"
		cfg.Servers[camelName] = server
	}

	// Save
	configPath, err := config.GetDefaultConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\n✓ Added %d server(s) to %s\n", len(servers), configPath)
	return nil
}

// parseAnyMCPConfig attempts to parse various MCP config formats intelligently.
// Handles many variations including non-standard keys.
// Returns servers map, detected format name, and error.
func parseAnyMCPConfig(input string) (map[string]*config.ServerConfig, string, error) {
	input = strings.TrimSpace(input)

	// Parse as generic JSON first
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		return nil, "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Try to find servers in various wrapper keys
	wrapperKeys := []string{
		"mcpServers", "mcp_servers", "MCP_SERVERS", "MCPServers",
		"mcp", "MCP",
		"servers", "Servers", "SERVERS",
		"tools", "Tools", "TOOLS",
		"context_servers", // Zed format
	}

	for _, key := range wrapperKeys {
		if wrapped, ok := raw[key]; ok {
			if serversMap, ok := wrapped.(map[string]interface{}); ok {
				servers := parseServersMap(serversMap)
				if len(servers) > 0 {
					return servers, fmt.Sprintf("Wrapped (%s)", key), nil
				}
			}
		}
	}

	// Try to parse as direct servers map: {"serverName": {...}, ...}
	servers := parseServersMap(raw)
	if len(servers) > 0 {
		return servers, "Direct server map", nil
	}

	// Try single server object: {"command": "...", ...}
	if server := parseSingleServer(raw); server != nil {
		// Ask user for name since it's a single server
		return map[string]*config.ServerConfig{"server": server}, "Single server object", nil
	}

	return nil, "", fmt.Errorf("could not find valid MCP server configuration")
}

// parseServersMap parses a map of server name -> server config.
func parseServersMap(raw map[string]interface{}) map[string]*config.ServerConfig {
	result := make(map[string]*config.ServerConfig)

	for name, val := range raw {
		if serverMap, ok := val.(map[string]interface{}); ok {
			if server := parseSingleServer(serverMap); server != nil {
				result[name] = server
			}
		}
	}

	return result
}

// parseSingleServer attempts to parse a single server config from a map.
// Handles many key variations:
//   - command: command, cmd, exec, executable, run, bin, binary
//   - args: args, arguments, argv, params, parameters, options
//   - env: env, environment, envVars, env_vars, envvars
func parseSingleServer(raw map[string]interface{}) *config.ServerConfig {
	// Find command (required)
	command := findStringKey(raw, 
		"command", "cmd", "exec", "executable", "run", "bin", "binary",
		"Command", "CMD", "Cmd")
	if command == "" {
		return nil
	}

	// Find args
	args := findStringArrayKey(raw,
		"args", "arguments", "argv", "params", "parameters", "options",
		"Args", "Arguments", "ARGS")

	// Find env
	env := findStringMapKey(raw,
		"env", "environment", "envVars", "env_vars", "envvars",
		"Env", "Environment", "ENV")

	return &config.ServerConfig{
		Command: command,
		Args:    args,
		Env:     config.NormalizeEnvVars(env),
	}
}

// findStringKey looks for a string value under any of the given keys.
func findStringKey(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// findStringArrayKey looks for a string array under any of the given keys.
func findStringArrayKey(m map[string]interface{}, keys ...string) []string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			if arr, ok := val.([]interface{}); ok {
				result := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						result = append(result, s)
					}
				}
				if len(result) > 0 {
					return result
				}
			}
		}
	}
	return nil
}

// findStringMapKey looks for a string map under any of the given keys.
func findStringMapKey(m map[string]interface{}, keys ...string) map[string]string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			if obj, ok := val.(map[string]interface{}); ok {
				result := make(map[string]string)
				for k, v := range obj {
					if s, ok := v.(string); ok {
						result[k] = s
					}
				}
				if len(result) > 0 {
					return result
				}
			}
		}
	}
	return nil
}

// readMultilineInput reads input until two consecutive newlines.
func readMultilineInput() string {
	reader := bufio.NewReader(os.Stdin)
	var lines []string
	emptyCount := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimRight(line, "\r\n")
		
		if line == "" {
			emptyCount++
			if emptyCount >= 1 {
				break
			}
		} else {
			emptyCount = 0
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// runAddWithFlags handles the traditional flag-based mode.
func runAddWithFlags(name, command string, args, envVars []string) error {
	if command == "" {
		return fmt.Errorf("--command is required")
	}

	// Load existing config
	cfg, err := config.Load()
	if err != nil {
		cfg = config.NewConfig()
	}

	// Parse environment variables
	env := make(map[string]string)
	for _, e := range envVars {
		key, value := parseEnvVar(e)
		if key != "" {
			env[key] = value
		}
	}

	// Create server config
	server := &config.ServerConfig{
		Command: command,
		Args:    args,
		Env:     env,
		Source:  "manual",
	}

	// Transform name to camelCase
	camelName := config.ToCamelCase(name)
	cfg.Servers[camelName] = server

	// Save config
	configPath, err := config.GetDefaultConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Added server '%s' to %s\n", camelName, configPath)
	return nil
}

// parseEnvVar splits "KEY=VALUE" into key and value.
func parseEnvVar(s string) (string, string) {
	for i, c := range s {
		if c == '=' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

// colorGreen returns text with green ANSI color.
func colorGreen(s string) string {
	return "\033[32m" + s + "\033[0m"
}
