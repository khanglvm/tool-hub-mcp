/*
Package config provides unit tests for validation functions.
*/
package config

import (
	"strings"
	"testing"
)

func TestIsSelfReference(t *testing.T) {
	tests := []struct {
		name     string
		server   *ServerConfig
		expected bool
	}{
		{
			name: "Direct binary name match",
			server: &ServerConfig{
				Command: "tool-hub-mcp",
			},
			expected: true,
		},
		{
			name: "npx with tool-hub-mcp package",
			server: &ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@khanglvm/tool-hub-mcp", "serve"},
			},
			expected: true,
		},
		{
			name: "npx with bare tool-hub-mcp",
			server: &ServerConfig{
				Command: "npx",
				Args:    []string{"tool-hub-mcp", "serve"},
			},
			expected: true,
		},
		{
			name: "npx with different package",
			server: &ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@lvmk/jira-mcp"},
			},
			expected: false,
		},
		{
			name: "Different binary",
			server: &ServerConfig{
				Command: "uvx",
				Args:    []string{"jira-mcp"},
			},
			expected: false,
		},
		{
			name: "npx without tool-hub-mcp",
			server: &ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-figma"},
			},
			expected: false,
		},
		{
			name: "npx with multiple args including tool-hub-mcp",
			server: &ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@khanglvm/tool-hub-mcp"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSelfReference(tt.server)
			if result != tt.expected {
				t.Errorf("IsSelfReference() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateServer(t *testing.T) {
	tests := []struct {
		name        string
		serverName  string
		server      *ServerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:       "Valid server",
			serverName: "jira",
			server: &ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@lvmk/jira-mcp"},
			},
			expectError: false,
		},
		{
			name:        "Empty command",
			serverName:  "bad-server",
			server:      &ServerConfig{Command: ""},
			expectError: true,
			errorMsg:    "empty command",
		},
		{
			name:       "Self-reference - direct",
			serverName: "tool-hub",
			server: &ServerConfig{
				Command: "tool-hub-mcp",
			},
			expectError: true,
			errorMsg:    "self-reference",
		},
		{
			name:       "Self-reference - npx",
			serverName: "tool-hub",
			server: &ServerConfig{
				Command: "npx",
				Args:    []string{"@khanglvm/tool-hub-mcp"},
			},
			expectError: true,
			errorMsg:    "self-reference",
		},
		{
			name:       "Valid server with env vars",
			serverName: "figma",
			server: &ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-figma"},
				Env:     map[string]string{"FIGMA_TOKEN": "test"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServer(tt.serverName, tt.server)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateServer() expected error containing '%s', got nil", tt.errorMsg)
					return
				}
				// Check if error message contains expected substring
				if tt.errorMsg != "" {
					errMsg := err.Error()
					found := false
					for _, substr := range []string{tt.errorMsg, tt.serverName} {
						if strings.Contains(errMsg, substr) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("ValidateServer() error = %v, expected to contain '%s' or '%s'", err, tt.errorMsg, tt.serverName)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateServer() unexpected error: %v", err)
				}
			}
		})
	}
}
