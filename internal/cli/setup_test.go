package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewSetupCmd(t *testing.T) {
	cmd := NewSetupCmd()

	if cmd == nil {
		t.Fatal("NewSetupCmd() returned nil")
	}

	// Verify command properties
	if cmd.Use != "setup" {
		t.Errorf("Expected Use='setup', got %q", cmd.Use)
	}
}

func TestSetupCommandHelp(t *testing.T) {
	cmd := NewSetupCmd()
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
		"setup",
		"MCP configurations",
		"Claude",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestSetupCommandFlags(t *testing.T) {
	cmd := NewSetupCmd()

	// Verify --yes flag exists (setup only has yes flag, not source)
	if cmd.Flags().Lookup("yes") == nil {
		t.Error("Flag 'yes' not registered")
	}
}

func TestSetupCommandFlagValues(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantYes bool
	}{
		{
			name:    "no flags",
			args:    []string{},
			wantYes: false,
		},
		{
			name:    "yes flag long",
			args:    []string{"--yes"},
			wantYes: true,
		},
		{
			name:    "yes flag short",
			args:    []string{"-y"},
			wantYes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSetupCmd()
			cmd.SetArgs(tt.args)

			// Parse flags
			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseFlags() failed: %v", err)
			}

			// Check flag values
			yes, _ := cmd.Flags().GetBool("yes")
			if yes != tt.wantYes {
				t.Errorf("yes flag = %v, want %v", yes, tt.wantYes)
			}
		})
	}
}

func TestSetupCommandExecution(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "help flag",
			args:    []string{"--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSetupCmd()
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
