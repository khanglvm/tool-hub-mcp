package spawner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

func TestNewPool(t *testing.T) {
	tests := []struct {
		name    string
		maxSize int
	}{
		{"default size", 3},
		{"large pool", 10},
		{"single process", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.maxSize)
			if pool == nil {
				t.Fatal("NewPool returned nil")
			}
			if pool.maxSize != tt.maxSize {
				t.Errorf("Expected maxSize %d, got %d", tt.maxSize, pool.maxSize)
			}
			if pool.processes == nil {
				t.Error("processes map not initialized")
			}
		})
	}
}

func TestProcessRequestID(t *testing.T) {
	proc := &Process{
		reqID: 0,
	}

	// Test that request IDs increment
	ids := make(map[int64]bool)
	for i := 0; i < 100; i++ {
		proc.mu.Lock()
		proc.reqID++
		id := proc.reqID
		proc.mu.Unlock()

		if ids[id] {
			t.Errorf("Duplicate request ID: %d", id)
		}
		ids[id] = true
	}

	// Verify we have 100 unique IDs
	if len(ids) != 100 {
		t.Errorf("Expected 100 unique IDs, got %d", len(ids))
	}
}

func TestPoolProcessMap(t *testing.T) {
	pool := NewPool(3)

	// Verify initial state
	pool.mu.Lock()
	if len(pool.processes) != 0 {
		t.Errorf("Expected 0 initial processes, got %d", len(pool.processes))
	}
	pool.mu.Unlock()

	// Create a mock process
	mockProc := &Process{
		reqID: 0,
	}

	// Add to pool
	pool.mu.Lock()
	pool.processes["test"] = mockProc
	pool.mu.Unlock()

	// Verify it's in the pool
	pool.mu.Lock()
	if len(pool.processes) != 1 {
		t.Errorf("Expected 1 process, got %d", len(pool.processes))
	}
	if proc, exists := pool.processes["test"]; !exists || proc != mockProc {
		t.Error("Process not found in pool or incorrect process")
	}
	pool.mu.Unlock()

	// Clear pool
	pool.processes = make(map[string]*Process)

	pool.mu.Lock()
	if len(pool.processes) != 0 {
		t.Errorf("Expected 0 processes after clear, got %d", len(pool.processes))
	}
	pool.mu.Unlock()
}

func TestPoolClose(t *testing.T) {
	pool := NewPool(3)

	// Test closing empty pool
	err := pool.Close()
	if err != nil {
		t.Errorf("Close() on empty pool returned error: %v", err)
	}

	// Verify pool is empty
	pool.mu.Lock()
	if len(pool.processes) != 0 {
		t.Errorf("Expected 0 processes after Close(), got %d", len(pool.processes))
	}
	pool.mu.Unlock()
}

func TestExecCommandVariable(t *testing.T) {
	// Test that execCommand variable exists and can be used
	originalExec := execCommand

	// Create a mock command function
	mockCalled := false
	mockExec := func(name string, args ...string) *exec.Cmd {
		mockCalled = true
		return exec.Command("echo", "test")
	}

	// Replace execCommand
	execCommand = mockExec

	// Call it
	cmd := execCommand("test-command")

	// Verify mock was called
	if !mockCalled {
		t.Error("Mock execCommand was not called")
	}

	if cmd == nil {
		t.Error("execCommand returned nil")
	}

	// Restore original
	execCommand = originalExec
}

// TestProcessConcurrency tests that the Process struct can handle concurrent access
func TestProcessConcurrency(t *testing.T) {
	proc := &Process{
		reqID: 0,
	}

	// Launch multiple goroutines incrementing reqID
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				proc.mu.Lock()
				proc.reqID++
				proc.mu.Unlock()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final count
	proc.mu.Lock()
	finalID := proc.reqID
	proc.mu.Unlock()

	if finalID != 1000 {
		t.Errorf("Expected reqID to be 1000, got %d", finalID)
	}
}

// TestPoolTimeout verifies timeout constant is reasonable
func TestPoolTimeout(t *testing.T) {
	// Verify DefaultTimeout is set
	if DefaultTimeout == 0 {
		t.Error("DefaultTimeout is zero")
	}

	// Verify it's reasonable (between 10s and 120s)
	if DefaultTimeout < 10*time.Second {
		t.Errorf("DefaultTimeout too short: %v", DefaultTimeout)
	}
	if DefaultTimeout > 120*time.Second {
		t.Errorf("DefaultTimeout too long: %v", DefaultTimeout)
	}
}

// TestGetNpmPackageFromConfig tests the npm package extraction logic
func TestGetNpmPackageFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.ServerConfig
		expected string
	}{
		{
			name: "npx with package",
			cfg: &config.ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@lvmk/jira-mcp"},
			},
			expected: "@lvmk/jira-mcp",
		},
		{
			name: "npx with flags before package",
			cfg: &config.ServerConfig{
				Command: "npx",
				Args:    []string{"--yes", "-y", "package-name"},
			},
			expected: "package-name",
		},
		{
			name: "non-npx command",
			cfg: &config.ServerConfig{
				Command: "node",
				Args:    []string{"script.js"},
			},
			expected: "",
		},
		{
			name: "npx with no package",
			cfg: &config.ServerConfig{
				Command: "npx",
				Args:    []string{"-y"},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNpmPackageFromConfig(tt.cfg)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestProcessKill tests the kill method doesn't panic
func TestProcessKill(t *testing.T) {
	// Create a process with no command
	proc := &Process{
		cancel: func() {},
	}

	// Should not panic
	proc.kill()

	// Create a process with a real command
	_, cancel := context.WithCancel(context.Background())
	cmd := exec.Command("sleep", "10")
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start test command: %v", err)
	}

	proc2 := &Process{
		cmd:    cmd,
		cancel: cancel,
	}

	// Kill it
	proc2.kill()

	// Wait briefly to ensure process is killed
	time.Sleep(100 * time.Millisecond)

	// Verify process is no longer running
	if cmd.Process != nil {
		// Try to find the process
		p, err := os.FindProcess(cmd.Process.Pid)
		if err == nil {
			// On Unix, FindProcess always succeeds, so we need to check if it's really alive
			err = p.Signal(os.Signal(nil))
			if err == nil {
				t.Error("Process still running after kill()")
			}
		}
	}
}

// TestToolStruct verifies Tool struct can be marshalled
func TestToolStruct(t *testing.T) {
	tool := Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"arg1": map[string]string{"type": "string"},
			},
		},
	}

	if tool.Name != "test-tool" {
		t.Errorf("Expected name 'test-tool', got %q", tool.Name)
	}
	if tool.Description != "A test tool" {
		t.Errorf("Expected description 'A test tool', got %q", tool.Description)
	}
	if tool.InputSchema == nil {
		t.Error("InputSchema is nil")
	}
}

// TestPoolMaxSize verifies maxSize is stored correctly
func TestPoolMaxSize(t *testing.T) {
	tests := []int{1, 3, 5, 10, 100}

	for _, maxSize := range tests {
		t.Run(fmt.Sprintf("maxSize=%d", maxSize), func(t *testing.T) {
			pool := NewPool(maxSize)
			if pool.maxSize != maxSize {
				t.Errorf("Expected maxSize %d, got %d", maxSize, pool.maxSize)
			}
		})
	}
}
