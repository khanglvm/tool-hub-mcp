package config

import (
	"testing"
)

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "dash-case",
			input:    "jira-mcp",
			expected: "jiraMcp",
		},
		{
			name:     "snake_case",
			input:    "jira_mcp",
			expected: "jiraMcp",
		},
		{
			name:     "PascalCase",
			input:    "JiraMcp",
			expected: "jiraMcp",
		},
		{
			name:     "already camelCase",
			input:    "jiraMcp",
			expected: "jiraMcp",
		},
		{
			name:     "multiple dashes",
			input:    "my-cool-mcp-server",
			expected: "myCoolMcpServer",
		},
		{
			name:     "single word",
			input:    "jira",
			expected: "jira",
		},
		{
			name:     "uppercase word",
			input:    "JIRA",
			expected: "jira",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "mixed format",
			input:    "my-MCP_server",
			expected: "myMcpServer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToEnvVarCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "camelCase",
			input:    "jiraBaseUrl",
			expected: "JIRA_BASE_URL",
		},
		{
			name:     "already SCREAMING_SNAKE",
			input:    "JIRA_BASE_URL",
			expected: "JIRA_BASE_URL",
		},
		{
			name:     "dash-case",
			input:    "jira-base-url",
			expected: "JIRA_BASE_URL",
		},
		{
			name:     "single word",
			input:    "jira",
			expected: "JIRA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToEnvVarCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToEnvVarCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeEnvVars(t *testing.T) {
	input := map[string]string{
		"jiraBaseUrl":    "http://jira.example.com",
		"JIRA_USERNAME":  "user",
		"jira-password":  "pass",
	}

	result := NormalizeEnvVars(input)

	expected := map[string]string{
		"JIRA_BASE_URL": "http://jira.example.com",
		"JIRA_USERNAME": "user",
		"JIRA_PASSWORD": "pass",
	}

	for key, val := range expected {
		if result[key] != val {
			t.Errorf("NormalizeEnvVars()[%q] = %q, want %q", key, result[key], val)
		}
	}

	// Test nil input
	if NormalizeEnvVars(nil) != nil {
		t.Error("NormalizeEnvVars(nil) should return nil")
	}
}
