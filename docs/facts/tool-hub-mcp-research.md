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

## Implementation Notes

- Claude Code v2.1.7+ supports "Tool Search" (lazy loading built-in)
- Most MCP SDK examples use TypeScript
- Process pooling improves performance for frequently-used servers
- Caching tool metadata reduces repeated `list_tools` calls
