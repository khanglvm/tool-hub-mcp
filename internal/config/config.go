/*
Package config handles loading, saving, and transforming tool-hub-mcp configuration.

Configuration is stored in ~/.tool-hub-mcp.json and uses a unified camelCase format
regardless of the source (Claude Code, OpenCode, etc.).

Schema:
  {
    "servers": {
      "serverName": {
        "command": "npx",
        "args": ["-y", "@package/name"],
        "env": {"KEY": "value"},
        "source": "claude-code"
      }
    },
    "settings": {
      "cacheToolMetadata": true,
      "processPoolSize": 3,
      "timeoutSeconds": 30
    }
  }
*/
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the root configuration structure.
type Config struct {
	// Servers maps server names (camelCase) to their configurations.
	Servers map[string]*ServerConfig `json:"servers"`
	
	// Settings contains global configuration options.
	Settings *Settings `json:"settings,omitempty"`
}

// ServerConfig represents a single MCP server configuration.
type ServerConfig struct {
	// Command is the executable to run (e.g., "npx", "/path/to/binary").
	Command string `json:"command"`
	
	// Args are the command-line arguments.
	Args []string `json:"args,omitempty"`
	
	// Env contains environment variables for the server.
	Env map[string]string `json:"env,omitempty"`
	
	// Source indicates where this config was imported from (e.g., "claude-code").
	Source string `json:"source,omitempty"`
	
	// Metadata contains cached tool information.
	Metadata *ServerMetadata `json:"metadata,omitempty"`
}

// ServerMetadata contains cached information about a server's tools.
type ServerMetadata struct {
	// Description is a human-readable description of the server.
	Description string `json:"description,omitempty"`
	
	// Tools is a cached list of tool names.
	Tools []string `json:"tools,omitempty"`
	
	// LastUpdated is when the metadata was last refreshed.
	LastUpdated string `json:"lastUpdated,omitempty"`
}

// Settings contains global configuration options.
type Settings struct {
	// CacheToolMetadata enables caching of tool definitions.
	CacheToolMetadata bool `json:"cacheToolMetadata,omitempty"`
	
	// ProcessPoolSize is the max number of concurrent MCP server processes.
	ProcessPoolSize int `json:"processPoolSize,omitempty"`
	
	// TimeoutSeconds is the default timeout for MCP operations.
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
}

// NewConfig creates a new empty configuration with initialized maps.
func NewConfig() *Config {
	return &Config{
		Servers: make(map[string]*ServerConfig),
		Settings: &Settings{
			CacheToolMetadata: true,
			ProcessPoolSize:   3,
			TimeoutSeconds:    30,
		},
	}
}

// GetDefaultConfigPath returns the path to ~/.tool-hub-mcp.json
func GetDefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".tool-hub-mcp.json"), nil
}

// Load reads the configuration from the default path.
func Load() (*Config, error) {
	configPath, err := GetDefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(configPath)
}

// LoadOrCreate loads config or returns empty config if not found.
// This enables silent first-run setup in serve command.
func LoadOrCreate() (*Config, error) {
	configPath, err := GetDefaultConfigPath()
	if err != nil {
		return nil, err
	}

	cfg, err := LoadFrom(configPath)
	if err != nil {
		// Check if error is "not found"
		if _, ok := err.(*ConfigNotFoundError); ok {
			// Return empty config instead of error - allows serve to start
			return NewConfig(), nil
		}
		return nil, err
	}
	return cfg, nil
}
