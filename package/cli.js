#!/usr/bin/env node

/**
 * CLI wrapper for tool-hub-mcp Go binary.
 * Locates the platform-specific binary from optionalDependencies and spawns it.
 * 
 * Architecture: Thin wrapper that passes through all args and stdio to the Go binary.
 */

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

/**
 * Get the platform-specific package name based on current OS and architecture.
 * @returns {string} Package name like '@khanglvm/tool-hub-mcp-darwin-arm64'
 */
function getPlatformPackageName() {
    const platform = os.platform();
    const arch = os.arch();

    // Map Node.js platform/arch to our package naming convention
    const platformMap = {
        'darwin': 'darwin',
        'linux': 'linux',
        'win32': 'win32'
    };

    const archMap = {
        'arm64': 'arm64',
        'x64': 'x64',
        'x86_64': 'x64'  // Fallback
    };

    const mappedPlatform = platformMap[platform];
    const mappedArch = archMap[arch];

    if (!mappedPlatform || !mappedArch) {
        throw new Error(`Unsupported platform: ${platform}-${arch}`);
    }

    return `@khanglvm/tool-hub-mcp-${mappedPlatform}-${mappedArch}`;
}

/**
 * Find the binary path from the installed platform package.
 * Searches in node_modules relative to this script.
 * @returns {string|null} Path to binary or null if not found
 */
function findBinaryFromPackage() {
    const packageName = getPlatformPackageName();
    const binaryName = os.platform() === 'win32' ? 'tool-hub-mcp.exe' : 'tool-hub-mcp';

    // Get platform directory name (e.g., 'darwin-arm64')
    const platformDir = packageName.replace('@khanglvm/tool-hub-mcp-', '');

    // Search locations (support various node_modules layouts + local dev)
    const searchPaths = [
        // Local development (platforms subdirectory)
        path.join(__dirname, 'platforms', platformDir, 'bin', binaryName),
        // Standard node_modules
        path.join(__dirname, 'node_modules', packageName, 'bin', binaryName),
        // Hoisted (pnpm, yarn workspaces)
        path.join(__dirname, '..', packageName, 'bin', binaryName),
        // Global install
        path.join(__dirname, '..', '..', packageName, 'bin', binaryName),
    ];

    for (const searchPath of searchPaths) {
        if (fs.existsSync(searchPath)) {
            return searchPath;
        }
    }

    return null;
}

/**
 * Find binary downloaded by postinstall fallback.
 * @returns {string|null} Path to binary or null if not found
 */
function findDownloadedBinary() {
    const binaryName = os.platform() === 'win32' ? 'tool-hub-mcp.exe' : 'tool-hub-mcp';
    const downloadPath = path.join(__dirname, 'bin', binaryName);

    if (fs.existsSync(downloadPath)) {
        return downloadPath;
    }

    return null;
}

/**
 * Main entry point - find binary and spawn with all arguments.
 */
function main() {
    // Try platform package first, then fallback to downloaded binary
    let binaryPath = findBinaryFromPackage() || findDownloadedBinary();

    if (!binaryPath) {
        console.error('Error: tool-hub-mcp binary not found.');
        console.error('This may happen if:');
        console.error('  1. optionalDependencies were disabled during install');
        console.error('  2. postinstall script failed to download the binary');
        console.error('');
        console.error('Try reinstalling: npm install @khanglvm/tool-hub-mcp');
        console.error('Or download manually from: https://github.com/khanglvm/tool-hub-mcp/releases');
        process.exit(1);
    }

    // Spawn the binary with all arguments passed through
    const child = spawn(binaryPath, process.argv.slice(2), {
        stdio: 'inherit',  // Pass through stdin/stdout/stderr
        shell: false
    });

    // Forward exit code
    child.on('close', (code) => {
        process.exit(code ?? 0);
    });

    // Handle spawn errors
    child.on('error', (err) => {
        console.error(`Failed to start tool-hub-mcp: ${err.message}`);
        process.exit(1);
    });
}

main();
