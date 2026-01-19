# tool-hub-mcp Facts

## Language Performance Benchmarks (CLI Startup)

| Language | CLI Startup | Cold Start (Lambda) | MCP SDK Status |
|----------|-------------|---------------------|----------------|
| Rust | 0.64ms | ~30ms | Community SDKs (rmcp, rust-mcp-sdk) |
| **Go** | **0.88ms** | **~45ms** | **Official SDK (Google collaboration)** |
| Node.js | 41.70ms | ~1500ms | Official SDK |

**Decision: Go** - Best balance of performance + ecosystem maturity + development velocity.

## Token Consumption Problem

- MCP tool definitions consume **60k+ tokens** when using many servers
- Each request sends ALL tool definitions to LLM, even if unused
- This can consume **50%+ of context window** before any actual work

## Token Optimization Solutions

1. **Lazy Loading**: Load tools only when needed (90%+ token reduction)
2. **Dynamic Toolsets**: Filter tools based on current task
3. **Meta-tools**: Expose few aggregator tools instead of individual tools
4. **Semantic Search**: Use embedding-based tool matching

## AI Client MCP Config Locations

| Client | Global Config | Project Config | Format |
|--------|---------------|----------------|--------|
| Claude Code | `~/.claude.json` | `.mcp.json` | `mcpServers` (dash-case) |
| OpenCode | `~/.opencode.json` | `opencode.json` | `mcp` (camelCase) |
| **Google Antigravity** | `~/.gemini/antigravity/mcp_config.json` | - | `mcpServers` |
| Gemini CLI | `~/.gemini/settings.json` | `.gemini/settings.json` | `mcpServers` |
| Cursor | `~/.cursor/mcp.json` | `.cursor/mcp.json` | `mcpServers` |
| Windsurf | `~/.codeium/windsurf/mcp_config.json` | - | `mcpServers` |
| Roo Code | `~/.roo/mcp.json` | `.roo/mcp.json` | `mcpServers` |
| Zed | `~/.config/zed/settings.json` | - | `context_servers` |

## Config Format Variations

- **dash-case keys**: Claude Desktop, Claude Code (jira-mcp, JIRA_BASE_URL)
- **camelCase keys**: OpenCode, some Cursor configs (jiraMcp, jiraBaseUrl)
- **snake_case**: YAML configs in DevOps tools (jira_mcp, jira_base_url)

## MetaMCP Architecture Pattern

- Acts as **proxy/aggregator** between AI client and MCP servers
- Provides **unified endpoint** - AI connects to one MCP only
- Uses **dynamic routing** to dispatch to appropriate child server
- Supports **tool discovery** via `discover()` and `execute()` functions
- Enables **lazy schema loading** - reads tool definitions only when needed

## MCP Server Distribution

- **npx**: Standard for TypeScript/JS MCP servers (`npx @package/name`)
- **uvx**: Standard for Python MCP servers (`uvx package-name`)
- **stdio transport**: Default for CLI-based assistants
- **SSE transport**: For web-based/IDE integrations

## Related Files
- `/plans/tool-hub-mcp.md` - Implementation plan
- `/internal/mcp/server.go` - MCP server with AI-native tool descriptions
- `/internal/benchmark/benchmark.go` - Token efficiency benchmark with known tool counts

## E2E Test Results (Claude Code Headless)

**Test**: Natural language prompt → tool-hub-mcp → Figma MCP → API call

**Prompt**: "Get design info from this Figma link: https://www.figma.com/design/..."

**Result**: ✅ SUCCESS
- Claude Code understood the natural language request
- Automatically identified tool-hub-mcp as the gateway
- Called hub_discover to find Figma tools
- Called hub_execute with correct parameters
- Got proper API response (403 due to auth - expected)

**AI Discoverability Key Patterns**:
1. "USE THIS TOOL WHEN" sections with universal triggers
2. Dynamic server count and list in descriptions
3. Focus on aggregator pattern: "external tools", "integrations", "capabilities"
4. Server enum in inputSchema for parameter validation

## Speed Benchmark Results

**Command**: `tool-hub-mcp benchmark speed -n 3`

| Metric | Cold Start | Warm (Pooled) | Average |
|--------|------------|---------------|---------|
| zaiMcpServer | 845ms | 0ms | 423ms |

**Interpretation**:
- Cold start latency ~1s (spawning MCP process)
- Warm requests near-instant (process reused from pool)
- Acceptable trade-off for 97.6% token savings

## Commands Reference

```bash
# Token efficiency benchmark
tool-hub-mcp benchmark

# Speed/latency benchmark
tool-hub-mcp benchmark speed -n 3

# Setup from existing configs
tool-hub-mcp setup

# Serve as MCP
tool-hub-mcp serve
```

## Implementation Notes

- Claude Code v2.1.7+ supports "Tool Search" (lazy loading built-in)
- Most MCP SDK examples use TypeScript
- Process pooling improves performance for frequently-used servers
- Caching tool metadata reduces repeated `list_tools` calls
