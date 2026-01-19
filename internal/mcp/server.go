/*
Package mcp implements the MCP server that exposes meta-tools.

The server uses stdio transport and exposes 5 meta-tools:
  - hub_list: List all registered MCP servers
  - hub_discover: Get tool definitions from a specific server
  - hub_search: Semantic search for tools across all servers
  - hub_execute: Execute a tool from a specific server
  - hub_help: Get detailed help/schema for a tool
*/
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/khanglvm/tool-hub-mcp/internal/spawner"
)

// Server represents the tool-hub-mcp MCP server.
type Server struct {
	config  *config.Config
	spawner *spawner.Pool
}

// NewServer creates a new MCP server with the given configuration.
func NewServer(cfg *config.Config) *Server {
	poolSize := 3
	if cfg.Settings != nil && cfg.Settings.ProcessPoolSize > 0 {
		poolSize = cfg.Settings.ProcessPoolSize
	}

	return &Server{
		config:  cfg,
		spawner: spawner.NewPool(poolSize),
	}
}

// Run starts the MCP server using stdio transport.
// This blocks until stdin is closed.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Bytes()
		
		response, err := s.handleRequest(line)
		if err != nil {
			// Send error response
			s.sendError(err)
			continue
		}

		if response != nil {
			s.sendResponse(response)
		}
	}

	return scanner.Err()
}

// MCPRequest represents an incoming MCP JSON-RPC request.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an outgoing MCP JSON-RPC response.
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error.
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// handleRequest processes an incoming MCP request.
func (s *Server) handleRequest(data []byte) (*MCPResponse, error) {
	var req MCPRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC request: %w", err)
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(&req)
	case "tools/list":
		return s.handleToolsList(&req)
	case "tools/call":
		return s.handleToolsCall(&req)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &MCPError{Code: -32601, Message: "Method not found"},
		}, nil
	}
}

// handleInitialize handles the MCP initialize request.
func (s *Server) handleInitialize(req *MCPRequest) (*MCPResponse, error) {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "tool-hub-mcp",
				"version": "0.1.0",
			},
		},
	}, nil
}

// handleToolsList returns the list of available meta-tools.
func (s *Server) handleToolsList(req *MCPRequest) (*MCPResponse, error) {
	tools := []map[string]interface{}{
		{
			"name":        "hub_list",
			"description": "List all registered MCP servers in tool-hub-mcp",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "hub_discover",
			"description": "Get tool definitions from a specific MCP server. Use this to see what tools are available on a server before executing them.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Name of the server (use hub_list to see available servers)",
					},
				},
				"required": []string{"server"},
			},
		},
		{
			"name":        "hub_search",
			"description": "Search for tools across all registered MCP servers using keywords",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (e.g., 'create issue', 'search documents')",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "hub_execute",
			"description": "Execute a tool from a specific MCP server",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Name of the server",
					},
					"tool": map[string]interface{}{
						"type":        "string",
						"description": "Name of the tool to execute",
					},
					"arguments": map[string]interface{}{
						"type":        "object",
						"description": "Arguments to pass to the tool",
					},
				},
				"required": []string{"server", "tool"},
			},
		},
		{
			"name":        "hub_help",
			"description": "Get detailed help and schema for a specific tool",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Name of the server",
					},
					"tool": map[string]interface{}{
						"type":        "string",
						"description": "Name of the tool",
					},
				},
				"required": []string{"server", "tool"},
			},
		},
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}, nil
}

// handleToolsCall handles tool execution requests.
func (s *Server) handleToolsCall(req *MCPRequest) (*MCPResponse, error) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	var result interface{}
	var err error

	switch params.Name {
	case "hub_list":
		result, err = s.execHubList()
	case "hub_discover":
		serverName, _ := params.Arguments["server"].(string)
		result, err = s.execHubDiscover(serverName)
	case "hub_search":
		query, _ := params.Arguments["query"].(string)
		result, err = s.execHubSearch(query)
	case "hub_execute":
		serverName, _ := params.Arguments["server"].(string)
		toolName, _ := params.Arguments["tool"].(string)
		args, _ := params.Arguments["arguments"].(map[string]interface{})
		result, err = s.execHubExecute(serverName, toolName, args)
	case "hub_help":
		serverName, _ := params.Arguments["server"].(string)
		toolName, _ := params.Arguments["tool"].(string)
		result, err = s.execHubHelp(serverName, toolName)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &MCPError{Code: -32602, Message: fmt.Sprintf("Unknown tool: %s", params.Name)},
		}, nil
	}

	if err != nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &MCPError{Code: -32000, Message: err.Error()},
		}, nil
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": result,
				},
			},
		},
	}, nil
}

// execHubList returns a list of registered servers.
func (s *Server) execHubList() (string, error) {
	if len(s.config.Servers) == 0 {
		return "No servers registered. Run 'tool-hub-mcp setup' to import configurations.", nil
	}

	var result string
	result = fmt.Sprintf("Registered MCP Servers (%d):\n", len(s.config.Servers))
	for name, server := range s.config.Servers {
		result += fmt.Sprintf("  • %s (source: %s)\n", name, server.Source)
	}
	return result, nil
}

// execHubDiscover returns tools from a specific server.
func (s *Server) execHubDiscover(serverName string) (string, error) {
	server, exists := s.config.Servers[serverName]
	if !exists {
		return "", fmt.Errorf("server '%s' not found", serverName)
	}

	// Spawn the server and get its tools
	tools, err := s.spawner.GetTools(serverName, server)
	if err != nil {
		return "", fmt.Errorf("failed to discover tools: %w", err)
	}

	result := fmt.Sprintf("Tools from '%s':\n", serverName)
	for _, tool := range tools {
		result += fmt.Sprintf("  • %s: %s\n", tool.Name, tool.Description)
	}
	return result, nil
}

// execHubSearch searches for tools across all servers.
func (s *Server) execHubSearch(query string) (string, error) {
	// TODO: Implement semantic search
	return fmt.Sprintf("Search for '%s' - semantic search not yet implemented. Use hub_discover to list tools from specific servers.", query), nil
}

// execHubExecute executes a tool from a server.
func (s *Server) execHubExecute(serverName, toolName string, args map[string]interface{}) (string, error) {
	server, exists := s.config.Servers[serverName]
	if !exists {
		return "", fmt.Errorf("server '%s' not found", serverName)
	}

	result, err := s.spawner.ExecuteTool(serverName, server, toolName, args)
	if err != nil {
		return "", fmt.Errorf("failed to execute tool: %w", err)
	}

	return result, nil
}

// execHubHelp returns help for a specific tool.
func (s *Server) execHubHelp(serverName, toolName string) (string, error) {
	server, exists := s.config.Servers[serverName]
	if !exists {
		return "", fmt.Errorf("server '%s' not found", serverName)
	}

	help, err := s.spawner.GetToolHelp(serverName, server, toolName)
	if err != nil {
		return "", fmt.Errorf("failed to get help: %w", err)
	}

	return help, nil
}

// sendResponse writes a JSON-RPC response to stdout.
func (s *Server) sendResponse(resp *MCPResponse) {
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

// sendError writes an error response to stdout.
func (s *Server) sendError(err error) {
	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      nil,
		Error:   &MCPError{Code: -32700, Message: err.Error()},
	}
	s.sendResponse(resp)
}
