package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewBenchmarkCmd(t *testing.T) {
	cmd := NewBenchmarkCmd()

	if cmd == nil {
		t.Fatal("NewBenchmarkCmd() returned nil")
	}

	// Verify command properties
	if cmd.Use != "benchmark" {
		t.Errorf("Expected Use='benchmark', got %q", cmd.Use)
	}
}

func TestBenchmarkCommandHelp(t *testing.T) {
	cmd := NewBenchmarkCmd()
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
		"benchmark",
		"token",
		"efficiency",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestBenchmarkCommandAliases(t *testing.T) {
	cmd := NewBenchmarkCmd()

	// Verify aliases if they exist
	if cmd.Aliases != nil && len(cmd.Aliases) > 0 {
		// Just verify aliases are set up correctly
		t.Logf("Command has %d aliases: %v", len(cmd.Aliases), cmd.Aliases)
	}
}

func TestBenchmarkCommandExecution(t *testing.T) {
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
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: false, // benchmark should run without args
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewBenchmarkCmd()
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

func TestBenchmarkCommandProperties(t *testing.T) {
	cmd := NewBenchmarkCmd()

	// Verify command has required properties
	if cmd.Short == "" {
		t.Error("Command missing short description")
	}

	if cmd.RunE == nil && cmd.Run == nil {
		t.Error("Command missing execution function")
	}
}
