package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewListCmd(t *testing.T) {
	cmd := NewListCmd()

	if cmd == nil {
		t.Fatal("NewListCmd() returned nil")
	}

	// Verify command properties
	if cmd.Use != "list" {
		t.Errorf("Expected Use='list', got %q", cmd.Use)
	}

	// Verify aliases
	aliases := cmd.Aliases
	if len(aliases) == 0 || aliases[0] != "ls" {
		t.Errorf("Expected alias 'ls', got %v", aliases)
	}

	// Verify flags are registered
	if cmd.Flags().Lookup("json") == nil {
		t.Error("Flag 'json' not registered")
	}
	if cmd.Flags().Lookup("status") == nil {
		t.Error("Flag 'status' not registered")
	}
}

func TestListCommandHelp(t *testing.T) {
	cmd := NewListCmd()

	// Just verify command has proper description
	if cmd.Short == "" {
		t.Error("Command missing short description")
	}

	if !strings.Contains(cmd.Short, "List") || !strings.Contains(cmd.Short, "MCP") {
		t.Errorf("Short description doesn't mention listing MCP servers: %q", cmd.Short)
	}

	// Verify flags exist in help
	if cmd.Flags().Lookup("json") == nil {
		t.Error("--json flag not found in help")
	}
	if cmd.Flags().Lookup("status") == nil {
		t.Error("--status flag not found in help")
	}
}

func TestListCommandAliases(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"list command", []string{}, false},
		{"help flag", []string{"--help"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewListCmd()
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

func TestListCommandFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantJSON   bool
		wantStatus bool
	}{
		{
			name:       "no flags",
			args:       []string{},
			wantJSON:   false,
			wantStatus: false,
		},
		{
			name:       "json flag",
			args:       []string{"--json"},
			wantJSON:   true,
			wantStatus: false,
		},
		{
			name:       "status flag",
			args:       []string{"--status"},
			wantJSON:   false,
			wantStatus: true,
		},
		{
			name:       "both flags",
			args:       []string{"--json", "--status"},
			wantJSON:   true,
			wantStatus: true,
		},
		{
			name:       "short flags",
			args:       []string{"-j", "-s"},
			wantJSON:   true,
			wantStatus: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewListCmd()
			cmd.SetArgs(tt.args)

			// Parse flags
			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseFlags() failed: %v", err)
			}

			// Check flag values
			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag != tt.wantJSON {
				t.Errorf("json flag = %v, want %v", jsonFlag, tt.wantJSON)
			}

			statusFlag, _ := cmd.Flags().GetBool("status")
			if statusFlag != tt.wantStatus {
				t.Errorf("status flag = %v, want %v", statusFlag, tt.wantStatus)
			}
		})
	}
}
