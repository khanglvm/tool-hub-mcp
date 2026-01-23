# tool-hub-mcp

**Serverless MCP Aggregator** - Reduce AI context token consumption

## Problem

When using multiple MCP servers with AI clients (Claude Code, OpenCode, etc.), each server exposes all its tools to the AI context window. More servers = more tokens consumed before you even start working.

## Solution

`tool-hub-mcp` acts as a single MCP gateway that exposes only **2 meta-tools**:

| Tool | Description |
|------|-------------|
| `hub_search` | Semantic search for tools across servers (BM25 + bandit ranking) |
| `hub_execute` | Execute a tool from a server (with learning system) |

The AI calls these meta-tools to discover and execute tools on-demand, instead of loading all tool definitions upfront.

## Benchmark

Measured in Claude Code v2.1.6 using `--output-format json` to get exact `input_tokens` count.

**Test: 6 MCP servers** (jira, Playwright, mcp-outline, shadcn, chrome-devtools, Figma)

| Configuration | Input Tokens |
|---------------|--------------|
| 6 Individual MCPs | **48,371** |
| tool-hub only | **29,758** |
| **Tokens Saved** | **18,613** |
| **Reduction** | **38.48%** |

**Larger Test: 7 MCP servers** (98 tools total)

| Configuration | Input Tokens |
|---------------|--------------|
| 7 Individual MCPs | **15,150** |
| tool-hub only | **461** |
| **Tokens Saved** | **14,689** |
| **Reduction** | **96.9%** |

**Token Optimization (v1.2.0)**:
- Compact JSON (no indentation): ~35% reduction
- Removed redundant fields: ~40% per search result
- Result: 43.7% savings on 2 results, ~70% on 10 results

## Installation

```bash
# Zero-install (recommended) - works with npx, bunx, pnpm dlx, yarn dlx
npx @khanglvm/tool-hub-mcp setup

# Alternative: Go install
go install github.com/khanglvm/tool-hub-mcp/cmd/tool-hub-mcp@latest
```

## Quick Start

```bash
# 1. Import your existing MCP configs
tool-hub-mcp setup

# 2. Add to your AI client
# Claude Code:
claude mcp add -s user tool-hub -- npx -y @khanglvm/tool-hub-mcp serve
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

### Manage Servers

```bash
# List all servers
tool-hub-mcp list

# Remove a server
tool-hub-mcp remove jira

# Verify configuration
tool-hub-mcp verify
```

### Run MCP Server

```bash
# Start server (stdio transport)
tool-hub-mcp serve

# Via AI client (already configured)
claude mcp add tool-hub -- tool-hub-mcp serve
```

### Export Tool Index for Bash/Grep

Generate a local index file for offline tool search without MCP overhead:

```bash
# Export to default location (~/.tool-hub-mcp-index.jsonl)
tool-hub-mcp export-index

# Custom output path
tool-hub-mcp export-index --output ./my-tools.jsonl

# JSON array format (instead of JSONL)
tool-hub-mcp export-index --format json
```

**Auto-regeneration**: Index automatically updates when you run `setup`, `add`, or `remove` commands.

**Bash/Grep Usage Examples**:

```bash
# Find tools by server
grep '"jira"' ~/.tool-hub-mcp-index.jsonl

# Search tool descriptions
grep -i "search" ~/.tool-hub-mcp-index.jsonl | jq -r '.tool'

# List all tools
cat ~/.tool-hub-mcp-index.jsonl | jq -r '.tool'

# Count tools per server
cat ~/.tool-hub-mcp-index.jsonl | jq -r '.server' | sort | uniq -c

# Complex query: Find Jira tools with "issue" in description
grep '"jira"' ~/.tool-hub-mcp-index.jsonl | grep -i "issue" | jq .
```

**Why use bash/grep?**
- Zero MCP overhead (no process spawning)
- Works offline (local file)
- Standard Unix tools (no dependencies)
- Scriptable and composable

### Benchmark Performance

```bash
# Compare token consumption
tool-hub-mcp benchmark

# Measure latency
tool-hub-mcp benchmark speed
```

## Commands

| Command | Description |
|---------|-------------|
| `setup` | Import MCP configs from AI CLI tools |
| `add` | Add MCP server(s) - paste JSON or use flags |
| `remove` | Remove an MCP server |
| `list` | List registered servers |
| `verify` | Verify configuration |
| `serve` | Run the MCP server (stdio) |
| `export-index` | Export tool index for bash/grep search (offline) |
| `benchmark` | Compare token consumption |
| `benchmark speed` | Measure latency per server |
| `learning` | Manage learning system (status, export, clear, enable, disable) |

## Supported Config Sources

- Claude Code (`~/.claude.json`, `.mcp.json`)
- OpenCode (`~/.opencode.json`)
- Google Antigravity (`~/.gemini/antigravity/mcp_config.json`)
- Gemini CLI (`~/.gemini/settings.json`)
- Cursor (`~/.cursor/mcp.json`)
- Windsurf (`~/.codeium/windsurf/mcp_config.json`)

## How It Works

```
┌─────────────────────────────────────────────────────────┐
│                     AI Client                           │
│              (Claude Code, OpenCode, etc.)              │
└───────────────────────┬─────────────────────────────────┘
                        │ 2 meta-tools
                        ▼
┌─────────────────────────────────────────────────────────┐
│                   tool-hub-mcp                          │
│           hub_search │ hub_execute                      │
└───────────────────────┬─────────────────────────────────┘
                        │ On-demand spawning
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
   ┌─────────┐    ┌─────────┐    ┌─────────┐
   │  Jira   │    │ Outline │    │  Figma  │
   │  MCP    │    │   MCP   │    │   MCP   │
   └─────────┘    └─────────┘    └─────────┘
```

**AI Workflow:**
1. Calls `hub_search("what I need")` to find tools with ranked results
2. Calls `hub_execute(server, tool, args, searchId)` to execute (learning tracks usage)

**Result:** Tool definitions loaded on-demand, intelligent ranking improves over time.

## Architecture

**Technology Stack:**
- **Language:** Go 1.22+ (0.88ms startup)
- **Distribution:** Zero-install npm + Go binary
- **Transport:** JSON-RPC 2.0 over stdio
- **Protocol:** MCP 2024-11-05

**Key Design Decisions:**
- **Lazy Spawning:** Processes start only when tools accessed
- **Process Pool:** Reuse spawned processes (default: 3)
- **Safe Request IDs:** Atomic counter (not UnixNano) for JS compatibility
- **Stderr Draining:** Prevents pipe buffer deadlock

## Performance

**Token Efficiency:**
- 38-97% reduction vs traditional approach
- v1.2.0: Additional 43-70% savings per search
- Scales better with more servers
- Bash/grep alternative: Zero tokens (offline)

**Speed:**
- Cold start: ~845ms (first tool call)
- Warm start: ~50ms (process reuse)
- Average: 307ms across 5 servers

**Memory:**
- Config: ~1KB per server
- Process: ~5-10MB per active server
- Pool: ~15-30MB (3 processes)

## Configuration

**Config Location:** `~/.tool-hub-mcp.json`

**Format:**
```json
{
  "servers": {
    "serverName": {
      "command": "npx",
      "args": ["-y", "@package/name"],
      "env": {"KEY": "value"},
      "source": "claude-code"
    }
  },
  "settings": {
    "cacheToolMetadata": true,
    "processPoolSize": 3,
    "timeoutSeconds": 30
  }
}
```

## Development Workflow

### Setup

Install git hooks for automatic testing:

```bash
make setup-hooks
```

### Testing

```bash
# Run all tests
make test

# Run tests with race detector
make test-race

# Run fast tests (pre-commit)
make test-fast

# Run full suite with coverage check (pre-push)
make test-coverage
```

### Git Hooks

- **Pre-commit:** Runs fast tests on changed packages (~10s)
- **Pre-push:** Runs full suite with coverage check (~60s, requires 80% coverage)
- **Bypass:** Use `git commit --no-verify` or `git push --no-verify` for emergencies

The hooks prevent failing code from reaching the remote repository. Coverage threshold is enforced at 80%.

## Documentation

Comprehensive documentation available in `/docs/`:

- **Project Overview & PDR:** `/docs/project-overview-pdr.md`
- **Code Standards:** `/docs/code-standards.md`
- **Codebase Summary:** `/docs/codebase-summary.md`
- **System Architecture:** `/docs/system-architecture.md`
- **Design Guidelines:** `/docs/design-guidelines.md`
- **Deployment Guide:** `/docs/deployment-guide.md`
- **Project Roadmap:** `/docs/project-roadmap.md`

## Contributing

Contributions welcome! Please see:
- `/docs/` for architecture and design guidelines
- `/CLAUDE.md` for development workflows
- GitHub Issues for bug reports and feature requests

## License

MIT

## Links

- **npm:** https://www.npmjs.com/package/@khanglvm/tool-hub-mcp
- **GitHub:** https://github.com/khanglvm/tool-hub-mcp
- **MCP Protocol:** https://modelcontextprotocol.io/
