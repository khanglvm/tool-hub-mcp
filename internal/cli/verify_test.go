package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewVerifyCmd(t *testing.T) {
	cmd := NewVerifyCmd()

	if cmd == nil {
		t.Fatal("NewVerifyCmd() returned nil")
	}

	// Verify command properties
	if cmd.Use != "verify" {
		t.Errorf("Expected Use='verify', got %q", cmd.Use)
	}
}

func TestVerifyCommandHelp(t *testing.T) {
	cmd := NewVerifyCmd()
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
		"verify",
		"Verify",
		"configuration",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestVerifyCommandExecution(t *testing.T) {
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
			wantErr: false, // verify runs without args
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewVerifyCmd()
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
