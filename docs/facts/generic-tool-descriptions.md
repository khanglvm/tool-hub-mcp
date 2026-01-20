# Generic Tool Descriptions

## Summary
Tool descriptions in `server.go` are now **fully generic** - no hardcoded server/service names (playwright, figma, jira, etc.).

## Key Changes (Jan 2026)

### Removed Hardcoding
- `serverDescriptions` map - removed all hardcoded `"playwright": "Browser automation..."` entries
- `keywordMap` in `execHubSearch` - removed 30+ hardcoded keyword→server mappings
- Static examples like `"create a ticket"`, `"take screenshot"` removed from descriptions

### Dynamic Behavior Retained
- `getServerNames()` and `getServerNamesList()` still provide **runtime** server lists from config
- `hub_discover` and `hub_execute` still show `CURRENTLY REGISTERED: %s` with dynamic server list
- Search now matches query directly against actual configured server names

### Tool Description Philosophy
The hub acts as a **gateway** that doesn't know what tools are available until runtime:

```
IMPORTANT: This tool hub does NOT know what tools are available until you call hub_list.
You MUST call this tool first to discover currently configured servers and their capabilities.
```

### Search Behavior Change
**Before**: Hardcoded keyword→server map (e.g., "ticket" → jira, "screenshot" → playwright)
**After**: Simple substring match against actual registered server names

AI must now:
1. Call `hub_list` first to see available servers
2. Call `hub_discover(server)` to see tools and their descriptions
3. Use server descriptions (from the servers themselves) to understand capabilities

---

**Reminder for AI**: Always read `/docs/facts/*.md` before making changes. Update this file if tool description logic changes.
