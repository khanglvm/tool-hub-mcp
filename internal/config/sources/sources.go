/*
Package sources provides readers for MCP configurations from various AI CLI tools.

Supported sources:
  - Claude Code: ~/.claude.json, .mcp.json
  - OpenCode: ~/.opencode.json, opencode.json
  - Google Antigravity: ~/.gemini/antigravity/mcp_config.json
  - Gemini CLI: ~/.gemini/settings.json
  - Cursor: ~/.cursor/mcp.json
  - Windsurf: ~/.codeium/windsurf/mcp_config.json
  - Roo Code: ~/.roo/mcp.json
*/
package sources

import (
	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

// Source represents an MCP configuration source (e.g., Claude Code, OpenCode).
type Source interface {
	// Name returns the source identifier (e.g., "claude-code").
	Name() string

	// Scan searches for and parses MCP configurations.
	// Returns nil if no configuration is found.
	Scan() (*SourceResult, error)
}

// SourceResult contains the parsed MCP servers from a source.
type SourceResult struct {
	// ConfigPath is the path to the configuration file that was read.
	ConfigPath string

	// Servers maps server names to their configurations.
	Servers map[string]*config.ServerConfig
}

// GetAllSources returns all available configuration sources.
// Sources are returned in priority order (more specific sources first).
func GetAllSources() []Source {
	return []Source{
		NewClaudeCodeSource(),
		NewOpenCodeSource(),
		// Future sources can be added here:
		// NewAntigravitySource(),
		// NewGeminiCLISource(),
		// NewCursorSource(),
		// NewWindsurfSource(),
		// NewRooCodeSource(),
	}
}
