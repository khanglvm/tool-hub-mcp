/*
Package config provides validation helpers for MCP server configurations.

This file contains shared validation functions used by CLI commands
to detect and prevent configuration issues.
*/
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// IsSelfReference checks if a server config refers to tool-hub-mcp itself.
// This prevents circular references where tool-hub-mcp tries to spawn itself.
func IsSelfReference(server *ServerConfig) bool {
	// Check 1: Direct command match
	binaryName := filepath.Base(os.Args[0])
	if server.Command == binaryName || server.Command == "tool-hub-mcp" {
		return true
	}

	// Check 2: npx execution of tool-hub-mcp
	if server.Command == "npx" {
		for _, arg := range server.Args {
			if arg == "@khanglvm/tool-hub-mcp" || arg == "tool-hub-mcp" {
				return true
			}
		}
	}

	return false
}

// ValidateServer checks if a server config is valid for import.
// Returns an error if validation fails.
func ValidateServer(name string, server *ServerConfig) error {
	// Check for empty command
	if server.Command == "" {
		return fmt.Errorf("server '%s': empty command", name)
	}

	// Check for self-reference
	if IsSelfReference(server) {
		return fmt.Errorf("server '%s': self-reference detected (tool-hub-mcp cannot import itself)", name)
	}

	return nil
}
