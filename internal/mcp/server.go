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
	"strings"

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

// handleToolsList returns the list of available meta-tools with AI-native descriptions.
func (s *Server) handleToolsList(req *MCPRequest) (*MCPResponse, error) {
	// Build dynamic server list for AI context (runtime, not hardcoded)
	serverList := s.getServerNames()
	
	tools := []map[string]interface{}{
		{
			"name": "hub_list",
			"description": `Gateway to external tools and integrations. Lists all available MCP servers.

USE THIS TOOL FIRST WHEN:
• User asks about available tools, integrations, or capabilities
• User wants to interact with external services, APIs, or data sources  
• You need to discover what external tools are available before taking action
• User mentions ANY external service, platform, or integration by name

IMPORTANT: This tool hub does NOT know what tools are available until you call hub_list.
You MUST call this tool first to discover currently configured servers and their capabilities.

Returns: List of registered MCP server names that you can then explore with hub_discover.`,
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name": "hub_discover",
			"description": fmt.Sprintf(`Explore tools from a specific external integration. Shows available operations and required parameters.

USE THIS TOOL WHEN:
• You have called hub_list and know which server to explore
• You need to see what tools/operations a specific server provides
• You need parameter schemas before calling hub_execute

CURRENTLY REGISTERED: %s

Returns: List of tools with descriptions and parameter schemas from the specified server.`, serverList),
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Name of the server to explore",
						"enum":        s.getServerNamesList(),
					},
				},
				"required": []string{"server"},
			},
		},
		{
			"name": "hub_search",
			"description": `Find the right tool across all integrations using natural language.

USE THIS TOOL WHEN:
• User describes what they want to do but doesn't name a specific tool
• You're unsure which server has the capability the user needs
• User asks to "find", "search for", or "look for" a capability

HOW IT WORKS: Searches registered server names for matches to your query.
For best results, include server names or related keywords in your query.

Returns: Matching servers from those currently registered. Use hub_discover to see their tools.`,
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "What you want to do, or server name to find",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name": "hub_execute",
			"description": fmt.Sprintf(`Run a tool from an external integration.

USE THIS TOOL WHEN:
• You've discovered available tools (via hub_discover) and know which one to run
• You have the tool name and required arguments ready

WORKFLOW:
1. hub_list() → see available servers
2. hub_discover(server) → see tools and parameters  
3. hub_execute(server, tool, arguments) → run the tool

CURRENTLY REGISTERED: %s`, serverList),
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Server name",
						"enum":        s.getServerNamesList(),
					},
					"tool": map[string]interface{}{
						"type":        "string",
						"description": "Tool name (from hub_discover)",
					},
					"arguments": map[string]interface{}{
						"type":        "object",
						"description": "Tool arguments (schema from hub_discover)",
					},
				},
				"required": []string{"server", "tool"},
			},
		},
		{
			"name": "hub_help",
			"description": `Get detailed parameter schema for a specific tool.

USE THIS TOOL WHEN:
• You need the exact parameter format before calling hub_execute
• Tool execution failed due to incorrect arguments

Returns: Full JSON schema with types, descriptions, and required fields.`,
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Server name",
						"enum":        s.getServerNamesList(),
					},
					"tool": map[string]interface{}{
						"type":        "string",
						"description": "Tool name",
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

// buildServerCatalog creates a formatted list of servers.
// Note: No hardcoded descriptions - descriptions come from the servers themselves via hub_discover.
func (s *Server) buildServerCatalog() string {
	catalog := ""
	for name := range s.config.Servers {
		catalog += fmt.Sprintf("  • %s\n", name)
	}
	return catalog
}

// getServerNames returns a comma-separated list of server names.
func (s *Server) getServerNames() string {
	names := []string{}
	for name := range s.config.Servers {
		names = append(names, name)
	}
	result := ""
	for i, name := range names {
		if i > 0 {
			result += ", "
		}
		result += name
	}
	return result
}

// getServerNamesList returns server names as a slice for enum.
func (s *Server) getServerNamesList() []string {
	names := []string{}
	for name := range s.config.Servers {
		names = append(names, name)
	}
	return names
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

// execHubSearch searches for tools across all servers by matching against registered server names.
// No hardcoded keyword mappings - matches dynamically against actual configured servers.
func (s *Server) execHubSearch(query string) (string, error) {
	query = strings.ToLower(query)
	
	// Match against actual registered server names (dynamic, no hardcoding)
	matchedServers := []string{}
	for name := range s.config.Servers {
		nameLower := strings.ToLower(name)
		// Match if query contains server name or server name contains query
		if strings.Contains(query, nameLower) || strings.Contains(nameLower, query) {
			matchedServers = append(matchedServers, name)
		}
	}
	
	if len(matchedServers) == 0 {
		// No match, return all servers as suggestions
		var result strings.Builder
		result.WriteString(fmt.Sprintf("No direct match for '%s'. Available servers:\n\n", query))
		for name := range s.config.Servers {
			result.WriteString(fmt.Sprintf("  • %s\n", name))
		}
		result.WriteString("\nTry hub_discover(server) to see tools from a specific server.")
		return result.String(), nil
	}
	
	// Return matched servers with recommendation
	var result strings.Builder
	result.WriteString(fmt.Sprintf("For '%s', matching servers:\n\n", query))
	
	for _, server := range matchedServers {
		result.WriteString(fmt.Sprintf("  • %s\n", server))
	}
	
	result.WriteString("\nNext step: Call hub_discover(server) to see available tools, then hub_execute to run them.")
	return result.String(), nil
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
