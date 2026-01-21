# @khanglvm/tool-hub-mcp

**Serverless MCP Aggregator** - Reduce AI context token consumption by 38%

## Quick Start

```bash
# Zero-install usage (recommended)
npx @khanglvm/tool-hub-mcp setup
npx @khanglvm/tool-hub-mcp serve
npx @khanglvm/tool-hub-mcp benchmark
```

Also works with other JS package runners:
```bash
bunx @khanglvm/tool-hub-mcp setup
pnpm dlx @khanglvm/tool-hub-mcp setup
yarn dlx @khanglvm/tool-hub-mcp setup
```

## What It Does

When using multiple MCP servers with AI clients (Claude Code, OpenCode, etc.), each server exposes all its tools to the AI. With 5+ servers, you can easily consume **25,000+ tokens** just for tool definitions.

`tool-hub-mcp` acts as a single MCP endpoint that exposes only **5 meta-tools**:

| Tool | Description |
|------|-------------|
| `hub_list` | List all registered MCP servers |
| `hub_discover` | Get tools from a specific server |
| `hub_search` | Search for tools across servers |
| `hub_execute` | Execute a tool from a server |
| `hub_help` | Get detailed help for a tool |

**Result:** 18,613 tokens saved = **38.48% reduction** (measured with 6 MCP servers in Claude Code)

## Commands

```bash
# Import configs from AI CLI tools (Claude Code, OpenCode, etc.)
npx @khanglvm/tool-hub-mcp setup

# Run as MCP server (for AI client integration)
npx @khanglvm/tool-hub-mcp serve

# Add MCP servers manually
npx @khanglvm/tool-hub-mcp add jira --command npx --arg -y --arg @lvmk/jira-mcp

# Run token efficiency benchmark
npx @khanglvm/tool-hub-mcp benchmark

# Run speed/latency benchmark
npx @khanglvm/tool-hub-mcp benchmark speed
```

## MCP Client Configuration

Add to your AI client config:

**Claude Code:**
```bash
claude mcp add tool-hub -- npx @khanglvm/tool-hub-mcp serve
```

**Manual JSON config:**
```json
{
  "mcpServers": {
    "tool-hub": {
      "command": "npx",
      "args": ["@khanglvm/tool-hub-mcp", "serve"]
    }
  }
}
```

## Platforms

This package automatically installs the correct binary for your platform:

- macOS (Apple Silicon & Intel)
- Linux (x64 & ARM64)
- Windows (x64)

## License

MIT © khanglvm
