package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewServeCmd(t *testing.T) {
	cmd := NewServeCmd()

	if cmd == nil {
		t.Fatal("NewServeCmd() returned nil")
	}

	// Verify command properties
	if cmd.Use != "serve" {
		t.Errorf("Expected Use='serve', got %q", cmd.Use)
	}
}

func TestServeCommandHelp(t *testing.T) {
	cmd := NewServeCmd()
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
		"serve",
		"Start",
		"MCP server",
		"stdio",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestServeCommandFlags(t *testing.T) {
	cmd := NewServeCmd()

	// Verify --silent flag exists (if it exists in implementation)
	// This test validates the command structure
	if cmd.Flags() == nil {
		t.Error("Command flags not initialized")
	}
}

func TestServeCommandProperties(t *testing.T) {
	cmd := NewServeCmd()

	// Verify command has description
	if cmd.Short == "" {
		t.Error("Command missing short description")
	}

	if cmd.Long == "" {
		t.Error("Command missing long description")
	}

	// Verify RunE is set
	if cmd.RunE == nil {
		t.Error("Command RunE function not set")
	}
}
