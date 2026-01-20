/*
Package spawner handles on-demand spawning and management of child MCP servers.

The spawner maintains a pool of active processes and handles:
  - Lazy spawning (only when a tool is first executed)
  - Process lifecycle management
  - Communication with child MCP servers via stdio
  - Timeout handling
*/
package spawner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

// Tool represents a tool definition from a child MCP server.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// Pool manages a pool of child MCP server processes.
type Pool struct {
	maxSize int
	mu      sync.Mutex
	
	// processes maps server names to active processes
	processes map[string]*Process
}

// Process represents a running MCP server process.
type Process struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
	// reqID is an atomic counter for generating request IDs
	// We use a counter instead of UnixNano to avoid JavaScript precision issues
	// (JS Number.MAX_SAFE_INTEGER = 2^53-1 = 9007199254740991)
	reqID  int64
}


// NewPool creates a new process pool.
func NewPool(maxSize int) *Pool {
	return &Pool{
		maxSize:   maxSize,
		processes: make(map[string]*Process),
	}
}

// GetTools spawns a server (if needed) and returns its tool list.
func (p *Pool) GetTools(name string, cfg *config.ServerConfig) ([]Tool, error) {
	proc, err := p.getOrSpawn(name, cfg)
	if err != nil {
		return nil, err
	}

	// Send tools/list request
	response, err := proc.sendRequest("tools/list", nil)
	if err != nil {
		return nil, err
	}

	// Parse response
	var result struct {
		Tools []Tool `json:"tools"`
	}
	
	resultBytes, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, err
	}

	return result.Tools, nil
}

// ExecuteTool executes a tool on a child server.
func (p *Pool) ExecuteTool(name string, cfg *config.ServerConfig, toolName string, args map[string]interface{}) (string, error) {
	proc, err := p.getOrSpawn(name, cfg)
	if err != nil {
		return "", err
	}

	// Send tools/call request
	params := map[string]interface{}{
		"name":      toolName,
		"arguments": args,
	}

	response, err := proc.sendRequest("tools/call", params)
	if err != nil {
		return "", err
	}

	// Format response as string
	resultBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	return string(resultBytes), nil
}

// GetToolHelp gets detailed help for a specific tool.
func (p *Pool) GetToolHelp(name string, cfg *config.ServerConfig, toolName string) (string, error) {
	tools, err := p.GetTools(name, cfg)
	if err != nil {
		return "", err
	}

	for _, tool := range tools {
		if tool.Name == toolName {
			helpBytes, err := json.MarshalIndent(tool, "", "  ")
			if err != nil {
				return "", err
			}
			return string(helpBytes), nil
		}
	}

	return "", fmt.Errorf("tool '%s' not found on server '%s'", toolName, name)
}

// getOrSpawn returns an existing process or spawns a new one.
func (p *Pool) getOrSpawn(name string, cfg *config.ServerConfig) (*Process, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if proc, exists := p.processes[name]; exists {
		return proc, nil
	}

	// Spawn new process
	proc, err := p.spawn(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize the server
	if err := proc.initialize(); err != nil {
		proc.kill()
		return nil, fmt.Errorf("failed to initialize server: %w", err)
	}

	p.processes[name] = proc
	return proc, nil
}

// spawn starts a new MCP server process.
func (p *Pool) spawn(cfg *config.ServerConfig) (*Process, error) {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	
	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// CRITICAL: Create stderr pipe and drain it in background to prevent
	// pipe buffer deadlock. Some MCPs write to stderr during startup and 
	// if the buffer fills up (~64KB), it blocks the entire process including stdout.
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Drain stderr in background to prevent blocking
	go func() {
		io.Copy(io.Discard, stderr)
	}()

	return &Process{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}, nil
}


// initialize sends the MCP initialize request and initialized notification.
func (proc *Process) initialize() error {
	// Step 1: Send initialize request
	_, err := proc.sendRequest("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "tool-hub-mcp",
			"version": "0.1.0",
		},
	})
	if err != nil {
		return err
	}

	// Step 2: Send initialized notification (required by MCP protocol)
	// This is a notification, not a request - no response expected
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	notifBytes, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	notifBytes = append(notifBytes, '\n')
	
	proc.mu.Lock()
	_, err = proc.stdin.Write(notifBytes)
	proc.mu.Unlock()
	
	return err
}


// DefaultTimeout is the maximum time to wait for an MCP response.
// Set to 60s to handle npx package downloads on cold start.
const DefaultTimeout = 60 * time.Second

// sendRequest sends a JSON-RPC request and waits for response with timeout.
func (proc *Process) sendRequest(method string, params interface{}) (interface{}, error) {
	proc.mu.Lock()
	defer proc.mu.Unlock()

	// Generate a safe request ID using atomic counter
	// This avoids JavaScript precision issues with large UnixNano values
	proc.reqID++
	reqID := proc.reqID

	// Build request
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      reqID,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}

	// Send request
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	reqBytes = append(reqBytes, '\n')

	if _, err := proc.stdin.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response with timeout
	responseChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)

	go func() {
		line, err := proc.stdout.ReadBytes('\n')
		if err != nil {
			errorChan <- fmt.Errorf("failed to read response: %w", err)
			return
		}
		responseChan <- line
	}()

	select {
	case line := <-responseChan:
		var resp struct {
			JSONRPC string      `json:"jsonrpc"`
			ID      interface{} `json:"id"`
			Result  interface{} `json:"result"`
			Error   *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}

		if err := json.Unmarshal(line, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if resp.Error != nil {
			return nil, fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		return resp.Result, nil

	case err := <-errorChan:
		return nil, err

	case <-time.After(DefaultTimeout):
		return nil, fmt.Errorf("timeout after %v waiting for MCP response", DefaultTimeout)
	}
}

// kill terminates the process.
func (proc *Process) kill() {
	if proc.cmd != nil && proc.cmd.Process != nil {
		proc.cmd.Process.Kill()
	}
}
