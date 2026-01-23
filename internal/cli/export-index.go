package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/khanglvm/tool-hub-mcp/internal/spawner"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

// ToolEntry represents a tool in the exported index.
type ToolEntry struct {
	Tool        string      `json:"tool"`
	Server      string      `json:"server"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// NewExportIndexCmd creates the export-index command.
func NewExportIndexCmd() *cobra.Command {
	var format string
	var output string

	cmd := &cobra.Command{
		Use:   "export-index",
		Short: "Export tool index for bash/grep search",
		Long: `Generate ~/.tool-hub-mcp-index.jsonl with all tools for offline grep/jq searching.

This command creates a local index file containing tool metadata from all registered
MCP servers. The index enables fast command-line searches without MCP overhead.

Default output: ~/.tool-hub-mcp-index.jsonl
Default format: JSONL (one tool per line)`,
		Example: `  # Export to default location
  tool-hub-mcp export-index

  # Export as JSON array
  tool-hub-mcp export-index --format json

  # Custom output path
  tool-hub-mcp export-index --output ./tools.jsonl

Grep usage examples:
  # Find Jira tools
  grep '"jira"' ~/.tool-hub-mcp-index.jsonl

  # Search descriptions
  grep -i "search" ~/.tool-hub-mcp-index.jsonl | jq -r '.tool'

  # Extract all tool names
  cat ~/.tool-hub-mcp-index.jsonl | jq -r '.tool'

  # Count tools per server
  cat ~/.tool-hub-mcp-index.jsonl | jq -r '.server' | sort | uniq -c`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExportIndex(format, output)
		},
	}

	cmd.Flags().StringVar(&format, "format", "jsonl", "Output format: json or jsonl")
	cmd.Flags().StringVar(&output, "output", "", "Output path (default: ~/.tool-hub-mcp-index.jsonl)")

	return cmd
}

// runExportIndex executes the export-index command.
func runExportIndex(format, output string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Servers) == 0 {
		fmt.Println("No servers configured.")
		fmt.Println("Run 'tool-hub-mcp setup' to import from AI CLI tools.")
		return nil
	}

	// Default output path
	if output == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		ext := ".jsonl"
		if format == "json" {
			ext = ".json"
		}
		output = filepath.Join(home, ".tool-hub-mcp-index"+ext)
	}

	// Acquire file lock to prevent concurrent writes
	lockFile, err := acquireFileLock(output)
	if err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	defer releaseFileLock(lockFile)

	// Create spawner pool
	pool := spawner.NewPool(cfg.Settings.ProcessPoolSize)
	defer pool.Close()

	// Collect tools from all servers
	var allTools []ToolEntry
	for name, serverCfg := range cfg.Servers {
		tools, err := pool.GetTools(name, serverCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch tools from %s: %v\n", name, err)
			continue
		}

		for _, tool := range tools {
			allTools = append(allTools, ToolEntry{
				Tool:        tool.Name,
				Server:      name,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			})
		}
	}

	// Write to file
	return writeIndex(allTools, output, format)
}

// writeIndex writes the tool index to a file.
func writeIndex(tools []ToolEntry, path, format string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create index file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	if format == "json" {
		// JSON array format
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(tools); err != nil {
			return fmt.Errorf("failed to encode tools: %w", err)
		}
	} else {
		// JSONL format (one per line)
		for _, tool := range tools {
			if err := encoder.Encode(tool); err != nil {
				return fmt.Errorf("failed to encode tool: %w", err)
			}
		}
	}

	fmt.Printf("✓ Exported %d tools to %s\n", len(tools), path)
	return nil
}

// acquireFileLock acquires an exclusive lock on the index file.
func acquireFileLock(path string) (*os.File, error) {
	lockPath := path + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	err = unix.Flock(int(lockFile.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		lockFile.Close()
		return nil, fmt.Errorf("failed to acquire lock (another export in progress?): %w", err)
	}

	return lockFile, nil
}

// releaseFileLock releases the file lock and removes the lock file.
func releaseFileLock(lockFile *os.File) error {
	if lockFile == nil {
		return nil
	}

	lockPath := lockFile.Name()

	// Release lock
	unix.Flock(int(lockFile.Fd()), unix.LOCK_UN)
	lockFile.Close()

	// Remove lock file
	return os.Remove(lockPath)
}

// RegenerateIndex silently regenerates the index file in the background.
// Called by setup/add/remove commands to keep index fresh.
func RegenerateIndex() {
	go func() {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		indexPath := filepath.Join(home, ".tool-hub-mcp-index.jsonl")

		// Acquire lock before writing
		lockFile, err := acquireFileLock(indexPath)
		if err != nil {
			// Gracefully skip if lock unavailable (another export in progress)
			return
		}
		defer releaseFileLock(lockFile)

		// Run export silently (errors ignored)
		_ = runExportIndex("jsonl", "")
	}()
}
