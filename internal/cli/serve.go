package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/khanglvm/tool-hub-mcp/internal/mcp"
	"github.com/khanglvm/tool-hub-mcp/internal/version"
)

// NewServeCmd creates the 'serve' command for running the MCP server.
//
// This is the main command that exposes the 5 meta-tools via stdio transport:
// - hub_list, hub_discover, hub_search, hub_execute, hub_help
func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server (stdio transport)",
		Long: `Start the tool-hub-mcp server using stdio transport.

This server exposes 5 meta-tools to AI clients:
  • hub_list     - List all registered MCP servers
  • hub_discover - Get tool definitions from a specific server
  • hub_search   - Semantic search for tools across all servers
  • hub_execute  - Execute a tool from a specific server
  • hub_help     - Get detailed help/schema for a tool

The server spawns child MCP servers on-demand when tools are executed.`,
		Example: `  # Run directly
  tool-hub-mcp serve

  # Add to Claude Code
  claude mcp add tool-hub -- tool-hub-mcp serve`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe()
		},
	}

	return cmd
}

// runServe starts the MCP server with stdio transport and signal handling.
// Implements graceful shutdown on SIGINT/SIGTERM/SIGQUIT.
func runServe() error {
	// Load configuration (creates empty config if missing)
	cfg, err := config.LoadOrCreate()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create MCP server
	server := mcp.NewServer(cfg)

	// Run one-time setup if no servers configured (blocking)
	if len(cfg.Servers) == 0 {
		log.Printf("No servers configured, running setup...")
		count, err := RunSetupNonInteractive()
		if err != nil {
			log.Printf("Setup failed: %v", err)
			// Continue with empty config - server will still work
		} else {
			log.Printf("Setup complete: %d servers imported", count)

			// Reload config with new servers
			newCfg, err := config.LoadOrCreate()
			if err != nil {
				log.Printf("Failed to reload config: %v", err)
			} else {
				server.ReloadConfig(newCfg)
			}
		}
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	// Start background tasks with server context
	go checkForUpdates(server.Context())
	server.StartBackgroundDiscovery()

	// Run server in separate goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Run()
	}()

	// Wait for either signal or server error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v, shutting down gracefully...", sig)

		// Close server (triggers cleanup chain)
		if err := server.Close(); err != nil {
			log.Printf("Error during shutdown: %v", err)
			return err
		}

		log.Println("Shutdown complete")
		return nil

	case err := <-errChan:
		// Server.Run() returned (stdin closed or error)
		// Still need to cleanup resources
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("Error during cleanup: %v", closeErr)
		}
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}
}

// checkForUpdates checks for new version in background (context-aware).
func checkForUpdates(parentCtx context.Context) {
	// Check if cancelled before starting
	select {
	case <-parentCtx.Done():
		return
	default:
	}

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	latest, err := version.CheckUpdate(ctx)
	if err != nil {
		log.Printf("Update check failed: %v", err)
		return
	}

	if latest != "" && latest != version.Version {
		log.Printf("Update available: %s (current: %s)", latest, version.Version)
		log.Printf("Downloading in background...")

		tempPath, err := version.DownloadUpdate(ctx, latest)
		if err != nil {
			log.Printf("Download failed: %v", err)
			return
		}

		log.Printf("Update downloaded to %s. Will apply on next restart.", tempPath)
	}
}
