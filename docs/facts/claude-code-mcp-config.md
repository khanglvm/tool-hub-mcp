# Claude Code MCP Configuration

## Configuration Locations

### User-Level MCP Servers (Global)
- **File**: `~/.claude.json`
- **Section**: `mcpServers` at root level
- **Scope**: `-s user` when using CLI

### Project-Level MCP Servers
- **File**: `~/.claude.json` 
- **Section**: `projects["/path/to/project"].mcpServers`
- **Scope**: `-s project` when using CLI (uses `.mcp.json` in project root)

### Local Scope (Current Directory)
- **File**: `.mcp.json` in current directory
- **Scope**: `-s local` (default)

## CLI Commands

```bash
# Add MCP server (user scope)
claude mcp add -s user -- <name> <command> [args...]

# Add with environment variables
claude mcp add -s user -e KEY1=value1 -e KEY2=value2 -- <name> <command> [args...]

# List all MCPs
claude mcp list

# Remove MCP
claude mcp remove <name> -s user
```

## Current Active MCPs (Jan 2026)

| Name | Command | Purpose |
|------|---------|---------|
| jira | `npx -y @khanglvm/jira-mcp` | Jira Server v7.x integration |
| Playwright | `npx -y @executeautomation/playwright-mcp-server` | Browser automation |
| mcp-outline | `uvx mcp-outline` | Outline handbook integration |
| shadcn | `npx shadcn@latest mcp` | shadcn/ui components |
| chrome-devtools | `npx -y chrome-devtools-mcp@latest` | Chrome DevTools |
| Figma | `npx -y figma-developer-mcp --figma-api-key=... --stdio` | Figma design data |

## Important Notes

1. **`~/.claude/mcp.json`** is NOT read by Claude Code CLI - it's for other tools
2. **User-level MCPs** are stored in `~/.claude.json` under `mcpServers` key
3. **Project-level MCPs** can be stored in `.mcp.json` files or in `~/.claude.json` under project paths
4. **Plugin MCPs** (like `claude-mem`) are managed separately via plugin system
