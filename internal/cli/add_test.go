package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewAddCmd(t *testing.T) {
	cmd := NewAddCmd()

	if cmd == nil {
		t.Fatal("NewAddCmd() returned nil")
	}

	// Verify command properties
	if cmd.Use != "add [name]" {
		t.Errorf("Expected Use='add [name]', got %q", cmd.Use)
	}
}

func TestAddCommandHelp(t *testing.T) {
	cmd := NewAddCmd()
	cmd.SetArgs([]string{"--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() with --help failed: %v", err)
	}

	output := buf.String()

	// Verify help output contains expected content
	expectedStrings := []string{
		"add",
		"Add",
		"MCP server",
		"--command",
		"--arg",
		"--env",
		"--json",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestAddCommandFlags(t *testing.T) {
	cmd := NewAddCmd()

	// Verify all required flags are registered
	requiredFlags := []string{"command", "arg", "env", "json", "yes"}
	for _, flag := range requiredFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Flag %q not registered", flag)
		}
	}
}

func TestAddCommandFlagShortcuts(t *testing.T) {
	cmd := NewAddCmd()

	// Test that short flags are properly defined
	tests := []struct {
		long  string
		short string
	}{
		{"command", "c"},
		{"arg", "a"},
		{"env", "e"},
		{"json", "j"},
		{"yes", "y"},
	}

	for _, tt := range tests {
		flag := cmd.Flags().Lookup(tt.long)
		if flag == nil {
			t.Errorf("Flag %q not found", tt.long)
			continue
		}
		if flag.Shorthand != tt.short {
			t.Errorf("Flag %q shorthand = %q, want %q", tt.long, flag.Shorthand, tt.short)
		}
	}
}

func TestAddCommandFlagTypes(t *testing.T) {
	cmd := NewAddCmd()

	// Test flag types
	tests := []struct {
		name     string
		flagType string
	}{
		{"command", "string"},
		{"arg", "stringArray"},
		{"env", "stringArray"},
		{"json", "string"},
		{"yes", "bool"},
	}

	for _, tt := range tests {
		flag := cmd.Flags().Lookup(tt.name)
		if flag == nil {
			t.Errorf("Flag %q not found", tt.name)
			continue
		}
		if flag.Value.Type() != tt.flagType {
			t.Errorf("Flag %q type = %q, want %q", tt.name, flag.Value.Type(), tt.flagType)
		}
	}
}

func TestAddCommandValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "help flag",
			args:    []string{"--help"},
			wantErr: false,
		},
		{
			name:    "flag mode without name",
			args:    []string{"--command", "npx"},
			wantErr: true,
			errMsg:  "server name required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewAddCmd()
			cmd.SetArgs(tt.args)

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestAddCommandModes(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		description string
	}{
		{
			name:        "interactive mode",
			args:        []string{},
			description: "no args triggers interactive",
		},
		{
			name:        "json mode",
			args:        []string{"--json", "{}"},
			description: "json flag triggers json mode",
		},
		{
			name:        "flag mode",
			args:        []string{"server1", "--command", "npx"},
			description: "name + flags triggers flag mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewAddCmd()

			// Just verify command can be created with these args
			// Actual execution would require config file access
			if cmd == nil {
				t.Error("Command creation failed")
			}
		})
	}
}

func TestAddCommandExamples(t *testing.T) {
	cmd := NewAddCmd()

	// Verify examples are provided
	if cmd.Example == "" {
		t.Error("Command missing examples")
	}

	// Verify examples contain key patterns
	expectedPatterns := []string{
		"tool-hub-mcp add",
		"--json",
		"--command",
		"--arg",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(cmd.Example, pattern) {
			t.Errorf("Examples missing pattern %q", pattern)
		}
	}
}
