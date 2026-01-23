package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRemoveCmd(t *testing.T) {
	cmd := NewRemoveCmd()

	if cmd == nil {
		t.Fatal("NewRemoveCmd() returned nil")
	}

	// Verify command properties
	if cmd.Use != "remove <name>" {
		t.Errorf("Expected Use='remove <name>', got %q", cmd.Use)
	}

	// Verify aliases
	aliases := cmd.Aliases
	if len(aliases) == 0 || aliases[0] != "rm" {
		t.Errorf("Expected alias 'rm', got %v", aliases)
	}
}

func TestRemoveCommandHelp(t *testing.T) {
	cmd := NewRemoveCmd()
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
		"remove",
		"Remove",
		"server",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestRemoveCommandMissingName(t *testing.T) {
	cmd := NewRemoveCmd()
	cmd.SetArgs([]string{}) // No server name provided

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()

	// Should return error when no name provided
	if err == nil {
		t.Error("Expected error when no server name provided, got nil")
	}
}

func TestRemoveCommandValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: true,
			errMsg:  "",
		},
		{
			name:    "help flag",
			args:    []string{"--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRemoveCmd()
			cmd.SetArgs(tt.args)

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
