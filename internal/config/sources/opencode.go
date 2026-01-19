package sources

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

// OpenCodeSource reads MCP configurations from OpenCode.
//
// Configuration locations:
//   - ~/.opencode.json (global user config)
//   - opencode.json (project-level config)
//   - ~/.config/opencode/opencode.json (XDG config)
//
// Format:
//   {
//     "mcp": {
//       "serverName": {
//         "type": "local",
//         "command": "npx",
//         "args": ["-y", "@package/name"],
//         "env": {"KEY": "value"},
//         "enabled": true
//       }
//     }
//   }
type OpenCodeSource struct{}

// openCodeConfig represents the OpenCode configuration file structure.
type openCodeConfig struct {
	MCP map[string]openCodeServerConfig `json:"mcp"`
}

// openCodeServerConfig represents a single MCP server in OpenCode config.
type openCodeServerConfig struct {
	Type    string            `json:"type"`    // "local" or "remote"
	Command string            `json:"command"` // For local type
	Args    []string          `json:"args"`
	URL     string            `json:"url"` // For remote type
	Env     map[string]string `json:"env"`
	Enabled *bool             `json:"enabled"` // Optional, defaults to true
}

// NewOpenCodeSource creates a new OpenCode configuration source.
func NewOpenCodeSource() *OpenCodeSource {
	return &OpenCodeSource{}
}

// Name returns the source identifier.
func (s *OpenCodeSource) Name() string {
	return "opencode"
}

// Scan searches for and parses OpenCode MCP configurations.
func (s *OpenCodeSource) Scan() (*SourceResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Try config locations in order of precedence
	paths := []string{
		filepath.Join(home, ".opencode.json"),
		"opencode.json",
		filepath.Join(home, ".config", "opencode", "opencode.json"),
	}

	for _, path := range paths {
		result, err := s.parseFile(path)
		if err == nil && result != nil {
			return result, nil
		}
	}

	return nil, nil
}

// parseFile reads and parses an OpenCode configuration file.
func (s *OpenCodeSource) parseFile(path string) (*SourceResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cfg openCodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if len(cfg.MCP) == 0 {
		return nil, nil
	}

	// Convert to internal format
	servers := make(map[string]*config.ServerConfig)
	for name, server := range cfg.MCP {
		// Skip disabled servers
		if server.Enabled != nil && !*server.Enabled {
			continue
		}

		// Only support local (command-based) servers for now
		if server.Type == "remote" {
			// TODO: Support remote MCP servers
			continue
		}

		servers[name] = &config.ServerConfig{
			Command: server.Command,
			Args:    server.Args,
			Env:     config.NormalizeEnvVars(server.Env),
			Source:  s.Name(),
		}
	}

	if len(servers) == 0 {
		return nil, nil
	}

	return &SourceResult{
		ConfigPath: path,
		Servers:    servers,
	}, nil
}
