# tool-hub-mcp

**Serverless MCP Aggregator** - Reduce AI context token consumption by 60-97%

## Problem

When using multiple MCP servers with AI clients (Claude Code, OpenCode, etc.), each server exposes all its tools to the AI. With 5+ servers averaging 10 tools each, you can easily consume **25,000+ tokens** just for tool definitions - eating into your context window.

## Solution

`tool-hub-mcp` acts as a single MCP endpoint that exposes only **5 meta-tools**:

| Tool | Description |
|------|-------------|
| `hub_list` | List all registered MCP servers |
| `hub_discover` | Get tools from a specific server |
| `hub_search` | Search for tools across servers |
| `hub_execute` | Execute a tool from a server |
| `hub_help` | Get detailed help for a tool |

**Result:** ~461 tokens instead of 1,200-25,000+ = **61-97% reduction** (varies by server count)

## Installation

```bash
# Zero-install (recommended) - works with npx, bunx, pnpm dlx, yarn dlx
npx @khanglvm/tool-hub-mcp setup

# Alternative: Go install
go install github.com/khanglvm/tool-hub-mcp/cmd/tool-hub-mcp@latest

# Alternative: Direct binary download
curl -fsSL https://github.com/khanglvm/tool-hub-mcp/releases/latest/download/tool-hub-mcp-$(uname -s)-$(uname -m) -o tool-hub-mcp
chmod +x tool-hub-mcp
```

## Quick Start

```bash
# 1. Import your existing MCP configs
tool-hub-mcp setup

# 2. Or paste any MCP config JSON
tool-hub-mcp add --json '{"mcpServers": {...}}'

# 3. Run benchmark to see savings
tool-hub-mcp benchmark

# 4. Add to your AI client
# Claude Code:
claude mcp add tool-hub -- tool-hub-mcp serve
```

## Usage

### Import Configs from AI Tools

```bash
# Auto-detect and import from Claude Code, OpenCode, etc.
tool-hub-mcp setup
```

### Add MCP Servers Manually

```bash
# Paste any MCP config format (auto-detected)
tool-hub-mcp add --json '{
  "mcpServers": {
    "jira": {"command": "npx", "args": ["-y", "@lvmk/jira-mcp"]},
    "outline": {"command": "uvx", "args": ["mcp-outline"]}
  }
}'

# Or use flags
tool-hub-mcp add jira --command npx --arg -y --arg @lvmk/jira-mcp
```

### Benchmark

```bash
# Token efficiency
tool-hub-mcp benchmark

# Speed/latency
tool-hub-mcp benchmark speed
```

**Verified results with 5 production MCPs** (Playwright, Chrome DevTools, mcp-outline, shadcn, Figma):

| Metric | Value |
|--------|-------|
| **Token Benchmark** | |
| Traditional MCP tokens | ~15,150 (98 tools) |
| tool-hub-mcp tokens | 461 (5 meta-tools) |
| **Token savings** | **95.0%** |
| **Speed Benchmark** | |
| Average latency | 307ms |
| Cold start range | 352ms - 1.5s |
| Warm latency | 0-1ms (pooled) |

**Per-server breakdown:**
| MCP Server | Tools | Avg Latency |
|------------|-------|-------------|
| Playwright | 33 | 498ms |
| Chrome DevTools | 26 | 295ms |
| mcp-outline | 30 | 202ms |
| shadcn | 7 | 424ms |
| Figma | 2 | 118ms |

## Commands

| Command | Description |
|---------|-------------|
| `setup` | Import MCP configs from AI CLI tools |
| `add` | Add MCP server(s) - paste JSON or use flags |
| `remove` | Remove an MCP server |
| `list` | List registered servers |
| `verify` | Verify configuration |
| `serve` | Run the MCP server (stdio) |
| `benchmark` | Compare token consumption |

## Supported Config Sources

- Claude Code (`~/.claude.json`, `.mcp.json`)
- OpenCode (`~/.opencode.json`)
- Google Antigravity (`~/.gemini/antigravity/mcp_config.json`)
- Gemini CLI (`~/.gemini/settings.json`)
- And more...

## How It Works

```
┌─────────────────────────────────────────────────────────┐
│                     AI Client                           │
│              (Claude Code, OpenCode, etc.)              │
└───────────────────────┬─────────────────────────────────┘
                        │ 5 meta-tools (~500 tokens)
                        ▼
┌─────────────────────────────────────────────────────────┐
│                   tool-hub-mcp                          │
│  hub_list │ hub_discover │ hub_search │ hub_execute    │
└───────────────────────┬─────────────────────────────────┘
                        │ On-demand spawning
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
   ┌─────────┐    ┌─────────┐    ┌─────────┐
   │  Jira   │    │ Outline │    │  Figma  │
   │  MCP    │    │   MCP   │    │   MCP   │
   └─────────┘    └─────────┘    └─────────┘
```

## License

MIT
