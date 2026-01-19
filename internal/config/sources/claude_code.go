package sources

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

// ClaudeCodeSource reads MCP configurations from Claude Code.
//
// Configuration locations:
//   - ~/.claude.json (global user config)
//   - .mcp.json (project-level config)
//
// Format:
//   {
//     "mcpServers": {
//       "server-name": {
//         "command": "npx",
//         "args": ["-y", "@package/name"],
//         "env": {"KEY": "value"}
//       }
//     }
//   }
type ClaudeCodeSource struct{}

// claudeCodeConfig represents the Claude Code configuration file structure.
type claudeCodeConfig struct {
	MCPServers map[string]claudeServerConfig `json:"mcpServers"`
}

// claudeServerConfig represents a single MCP server in Claude Code config.
type claudeServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// NewClaudeCodeSource creates a new Claude Code configuration source.
func NewClaudeCodeSource() *ClaudeCodeSource {
	return &ClaudeCodeSource{}
}

// Name returns the source identifier.
func (s *ClaudeCodeSource) Name() string {
	return "claude-code"
}

// Scan searches for and parses Claude Code MCP configurations.
func (s *ClaudeCodeSource) Scan() (*SourceResult, error) {
	// Try global config first
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	globalPath := filepath.Join(home, ".claude.json")
	result, err := s.parseFile(globalPath)
	if err == nil && result != nil {
		return result, nil
	}

	// Try project-level config
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	projectPath := filepath.Join(cwd, ".mcp.json")
	return s.parseFile(projectPath)
}

// parseFile reads and parses a Claude Code configuration file.
func (s *ClaudeCodeSource) parseFile(path string) (*SourceResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, not an error
		}
		return nil, err
	}

	var cfg claudeCodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if len(cfg.MCPServers) == 0 {
		return nil, nil
	}

	// Convert to internal format
	servers := make(map[string]*config.ServerConfig)
	for name, server := range cfg.MCPServers {
		servers[name] = &config.ServerConfig{
			Command: server.Command,
			Args:    server.Args,
			Env:     config.NormalizeEnvVars(server.Env),
			Source:  s.Name(),
		}
	}

	return &SourceResult{
		ConfigPath: path,
		Servers:    servers,
	}, nil
}
