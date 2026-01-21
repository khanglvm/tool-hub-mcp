# Claude Code MCP Configuration

## Configuration Locations

| File | Used By | Notes |
|------|---------|-------|
| `~/.claude.json` → `mcpServers` | Claude Code CLI | **USE THIS** (user-level, `-s user`) |
| `~/.claude/mcp.json` | Other AI tools (Cursor, Windsurf, etc.) | Not read by Claude Code |
| `.mcp.json` (project root) | Claude Code CLI | Project-level, **NOT USED** |

## CLI Commands

```bash
# Add MCP server (user scope - ALWAYS USE THIS)
claude mcp add -s user -- <name> <command> [args...]

# List all MCPs
claude mcp list

# Remove MCP  
claude mcp remove <name> -s user
```

## Current Active MCP (Jan 2026)

| Name | Command | Purpose |
|------|---------|---------|
| tool-hub | `npx -y @khanglvm/tool-hub-mcp serve` | Gateway to all external tools |

## Previous Individual MCPs (Now Aggregated via tool-hub)

These are accessed through `tool-hub` instead of being configured individually:
- jira, Playwright, mcp-outline, shadcn, chrome-devtools, Figma

## Important Notes

1. **ALWAYS use `-s user` scope** - We don't use project-scope MCPs
2. **`~/.claude/mcp.json`** is for OTHER tools (Cursor, Windsurf), NOT Claude Code
3. **Plugin MCPs** (like `claude-mem`) are managed separately via plugin system
