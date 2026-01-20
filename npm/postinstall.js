#!/usr/bin/env node

/**
 * Postinstall fallback for tool-hub-mcp.
 * Downloads the Go binary from GitHub Releases if optionalDependencies were not installed.
 * 
 * This handles environments where:
 * - optionalDependencies are disabled (some CI environments)
 * - Package manager fails to resolve platform-specific packages
 */

const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Configuration
const GITHUB_REPO = 'khanglvm/tool-hub-mcp';
const BINARY_NAME = 'tool-hub-mcp';

/**
 * Get platform suffix for GitHub release assets.
 * @returns {string} Suffix like 'Darwin-arm64' or 'Linux-x86_64'
 */
function getPlatformSuffix() {
    const platform = os.platform();
    const arch = os.arch();

    const platformMap = {
        'darwin': 'Darwin',
        'linux': 'Linux',
        'win32': 'Windows'
    };

    const archMap = {
        'arm64': 'arm64',
        'x64': 'x86_64'
    };

    const mappedPlatform = platformMap[platform];
    const mappedArch = archMap[arch];

    if (!mappedPlatform || !mappedArch) {
        throw new Error(`Unsupported platform: ${platform}-${arch}`);
    }

    const suffix = `${mappedPlatform}-${mappedArch}`;
    return platform === 'win32' ? `${suffix}.exe` : suffix;
}

/**
 * Check if platform package binary already exists.
 * @returns {boolean} True if binary found in optionalDependencies
 */
function hasPlatformBinary() {
    const platform = os.platform();
    const arch = os.arch();

    const archMap = { 'arm64': 'arm64', 'x64': 'x64' };
    const platformMap = { 'darwin': 'darwin', 'linux': 'linux', 'win32': 'win32' };

    const packageName = `@khanglvm/tool-hub-mcp-${platformMap[platform]}-${archMap[arch]}`;
    const binaryName = platform === 'win32' ? `${BINARY_NAME}.exe` : BINARY_NAME;

    const searchPaths = [
        path.join(__dirname, 'node_modules', packageName, 'bin', binaryName),
        path.join(__dirname, '..', packageName, 'bin', binaryName),
        path.join(__dirname, '..', '..', packageName, 'bin', binaryName),
    ];

    return searchPaths.some(p => fs.existsSync(p));
}

/**
 * Follow redirects and download file.
 * @param {string} url - URL to download
 * @param {string} dest - Destination file path
 * @returns {Promise<void>}
 */
function downloadFile(url, dest) {
    return new Promise((resolve, reject) => {
        const file = fs.createWriteStream(dest);

        const request = (url) => {
            https.get(url, (response) => {
                // Handle redirects (GitHub releases use 302)
                if (response.statusCode === 301 || response.statusCode === 302) {
                    file.close();
                    fs.unlinkSync(dest);
                    request(response.headers.location);
                    return;
                }

                if (response.statusCode !== 200) {
                    file.close();
                    fs.unlinkSync(dest);
                    reject(new Error(`HTTP ${response.statusCode}: Failed to download ${url}`));
                    return;
                }

                response.pipe(file);
                file.on('finish', () => {
                    file.close();
                    resolve();
                });
            }).on('error', (err) => {
                file.close();
                fs.unlinkSync(dest);
                reject(err);
            });
        };

        request(url);
    });
}

/**
 * Get the latest release version from GitHub API.
 * @returns {Promise<string>} Version tag like 'v1.0.0'
 */
function getLatestVersion() {
    return new Promise((resolve, reject) => {
        const options = {
            hostname: 'api.github.com',
            path: `/repos/${GITHUB_REPO}/releases/latest`,
            headers: { 'User-Agent': 'tool-hub-mcp-installer' }
        };

        https.get(options, (response) => {
            let data = '';
            response.on('data', chunk => data += chunk);
            response.on('end', () => {
                try {
                    const release = JSON.parse(data);
                    if (release.tag_name) {
                        resolve(release.tag_name);
                    } else {
                        reject(new Error('No releases found'));
                    }
                } catch (e) {
                    reject(e);
                }
            });
        }).on('error', reject);
    });
}

/**
 * Main postinstall logic.
 */
async function main() {
    // Skip if platform binary already installed via optionalDependencies
    if (hasPlatformBinary()) {
        console.log('tool-hub-mcp: Binary found from optionalDependencies ✓');
        return;
    }

    console.log('tool-hub-mcp: optionalDependencies not available, downloading from GitHub...');

    try {
        // Get latest version
        const version = await getLatestVersion();
        console.log(`tool-hub-mcp: Downloading version ${version}...`);

        // Prepare download
        const suffix = getPlatformSuffix();
        const binaryName = os.platform() === 'win32' ? `${BINARY_NAME}.exe` : BINARY_NAME;
        const downloadUrl = `https://github.com/${GITHUB_REPO}/releases/download/${version}/${BINARY_NAME}-${suffix}`;

        // Create bin directory
        const binDir = path.join(__dirname, 'bin');
        if (!fs.existsSync(binDir)) {
            fs.mkdirSync(binDir, { recursive: true });
        }

        const destPath = path.join(binDir, binaryName);

        // Download binary
        await downloadFile(downloadUrl, destPath);

        // Make executable (Unix only)
        if (os.platform() !== 'win32') {
            fs.chmodSync(destPath, 0o755);
        }

        console.log('tool-hub-mcp: Binary downloaded and installed ✓');

    } catch (error) {
        console.error(`tool-hub-mcp: Failed to download binary: ${error.message}`);
        console.error('You can manually download from: https://github.com/khanglvm/tool-hub-mcp/releases');
        // Don't fail the install - user can still manually download
        process.exit(0);
    }
}

main();
