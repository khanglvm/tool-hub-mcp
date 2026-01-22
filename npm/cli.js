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
 * Download binary from GitHub releases.
 * @returns {Promise<string>} Path to downloaded binary
 */
async function downloadBinary() {
    const https = require('https');
    const fs = require('fs');
    const path = require('path');
    const { execSync } = require('child_process');
    const os = require('os');

    const platform = os.platform();
    const arch = os.arch();
    const platformMap = { 'darwin': 'Darwin', 'linux': 'Linux', 'win32': 'Windows' };
    const archMap = { 'arm64': 'arm64', 'x64': 'x86_64' };
    const suffix = `${platformMap[platform]}-${archMap[arch]}`;
    const binaryName = platform === 'win32' ? 'tool-hub-mcp.exe' : 'tool-hub-mcp';

    // Get version from package.json
    const packageJson = require('./package.json');
    const version = packageJson.version;

    const downloadUrl = `https://github.com/khanglvm/tool-hub-mcp/releases/download/v${version}/tool-hub-mcp-${suffix}`;
    const binDir = path.join(__dirname, 'bin');
    const destPath = path.join(binDir, binaryName);

    console.error(`tool-hub-mcp: Downloading binary from ${downloadUrl}...`);

    if (!fs.existsSync(binDir)) {
        fs.mkdirSync(binDir, { recursive: true });
    }

    // Use curl if available, otherwise https.get
    try {
        execSync(`curl -fsSL ${downloadUrl} -o ${destPath}`, { stdio: 'ignore' });
        if (fs.existsSync(destPath)) {
            if (platform !== 'win32') {
                fs.chmodSync(destPath, 0o755);
            }
            console.error(`tool-hub-mcp: Binary downloaded successfully`);
            return destPath;
        }
    } catch (e) {
        // curl failed or binary not downloaded, fall through to https.get
    }

    // Fallback to https.get
    return new Promise((resolve, reject) => {
        const file = fs.createWriteStream(destPath);
        https.get(downloadUrl, (response) => {
            if (response.statusCode === 301 || response.statusCode === 302) {
                file.close();
                https.get(response.headers.location, (resp2) => {
                    if (resp2.statusCode === 200) {
                        resp2.pipe(file);
                        file.on('finish', () => {
                            file.close();
                            if (platform !== 'win32') {
                                fs.chmodSync(destPath, 0o755);
                            }
                            console.error(`tool-hub-mcp: Binary downloaded successfully`);
                            resolve(destPath);
                        });
                    } else {
                        reject(new Error(`Failed to download: HTTP ${resp2.statusCode}`));
                    }
                });
                return;
            }
            if (response.statusCode !== 200) {
                reject(new Error(`HTTP ${response.statusCode}`));
                return;
            }
            response.pipe(file);
            file.on('finish', () => {
                file.close();
                if (platform !== 'win32') {
                    fs.chmodSync(destPath, 0o755);
                }
                console.error(`tool-hub-mcp: Binary downloaded successfully`);
                resolve(destPath);
            });
        }).on('error', reject);
    });
}

/**
 * Main entry point - find binary and spawn with all arguments.
 */
async function main() {
    // Try platform package first, then fallback to downloaded binary
    let binaryPath = findBinaryFromPackage() || findDownloadedBinary();

    if (!binaryPath) {
        console.error('tool-hub-mcp: Binary not found, downloading from GitHub...');
        try {
            binaryPath = await downloadBinary();
        } catch (err) {
            console.error('Error: Failed to download binary.');
            console.error(`  ${err.message}`);
            console.error('');
            console.error('You can manually download from: https://github.com/khanglvm/tool-hub-mcp/releases');
            process.exit(1);
        }
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
