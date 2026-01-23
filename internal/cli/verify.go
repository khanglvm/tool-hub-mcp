package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/spf13/cobra"
)

// NewVerifyCmd creates the 'verify' command for verifying configuration.
func NewVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify configuration and connections",
		Long: `Verify that the configuration is valid and optionally test
connections to registered MCP servers.`,
		Example: `  tool-hub-mcp verify`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerify()
		},
	}

	return cmd
}

// runVerify validates the configuration.
func runVerify() error {
	configPath, err := config.GetDefaultConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Check read permissions
	cfg, err := config.Load()
	if err != nil {
		return err // Will use our enhanced errors
	}

	fmt.Println("✓ Config file is readable")
	fmt.Printf("  Path: %s\n", configPath)
	fmt.Printf("  Servers: %d\n", len(cfg.Servers))

	// Check write permissions (warn but don't fail)
	writeCheckErr := checkConfigWritable(configPath)
	if writeCheckErr != nil {
		fmt.Println("⚠️  Config file is not writable")
		fmt.Printf("  %s\n", writeCheckErr)
	} else {
		fmt.Println("✓ Config file is writable")
	}

	// Validate each server
	for name, server := range cfg.Servers {
		if server.Command == "" {
			fmt.Printf("✗ %s: missing command\n", name)
			continue
		}

		// Check npx packages for existence
		if server.Command == "npx" && len(server.Args) > 0 {
			pkg := getNpmPackageName(server.Args)
			if pkg != "" {
				if err := validateNpmPackage(pkg); err != nil {
					fmt.Printf("✗ %s: package %s not found in npm registry\n", name, pkg)
					continue
				}
			}
		}

		fmt.Printf("✓ %s: %s\n", name, server.Command)
	}

	return nil
}

// getNpmPackageName extracts npm package name from args (handles -y flags).
func getNpmPackageName(args []string) string {
	for _, arg := range args {
		if arg == "-y" || arg == "--yes" {
			continue
		}
		// Skip flags
		if len(arg) > 0 && arg[0] == '-' {
			continue
		}
		// Return first non-flag arg as package name
		return arg
	}
	return ""
}

// validateNpmPackage checks if npm package exists using npm view.
func validateNpmPackage(pkg string) error {
	cmd := exec.Command("npm", "view", pkg, "name")
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("package not found: %s", pkg)
	}
	return nil
}

// checkConfigWritable tests if we can write to the config file
func checkConfigWritable(configPath string) error {
	// Import the checkWritePermission function from config package
	// This is a helper that won't fail the verification, just warn
	type permissionChecker interface {
		checkWritePermission(path string) error
	}

	// We'll use a simple write test instead
	// to avoid circular dependency issues
	return testWriteAccess(configPath)
}

// testWriteAccess attempts to verify write access
func testWriteAccess(path string) error {
	// Try to check if the config file or directory is writable
	// This is a simplified version of the check in saver.go
	dir := filepath.Dir(path)

	// Test directory writability by trying to create a temp file
	testFile := dir + "/.write-test"
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("directory not writable: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	// If file exists, test if we can write to it
	if _, err := os.Stat(path); err == nil {
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("file not writable: %w", err)
		}
		f.Close()
	}

	return nil
}
