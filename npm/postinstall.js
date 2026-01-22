#!/usr/bin/env node

/**
 * Postinstall script for tool-hub-mcp.
 * The binary is downloaded on-demand by cli.js when first executed.
 */

// Log installation info
console.log('tool-hub-mcp v' + require('./package.json').version + ' installed!');
console.log('Binary will be downloaded automatically on first use.');
