import { readFileSync, writeFileSync, appendFileSync, mkdirSync, copyFileSync, chmodSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { resolve } from "node:path";

const platforms = {
  "darwin-arm64": "Darwin-arm64",
  "darwin-x64": "Darwin-x86_64",
  "linux-x64": "Linux-x86_64",
  "linux-arm64": "Linux-arm64",
  "win32-x64": "Windows-x86_64.exe",
};
const root = JSON.parse(readFileSync("npm/package.json", "utf8"));
const version = root.version;
if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(version)) throw new Error("Invalid package version");
if (process.env.GITHUB_REF_TYPE === "tag" && process.env.GITHUB_REF_NAME !== `v${version}`) {
  throw new Error(`Tag ${process.env.GITHUB_REF_NAME} must match npm/package.json version v${version}`);
}
const dryRun = process.env.DRY_RUN === "true";
const distTag = version.split("+")[0].includes("-") ? "next" : "latest";
const entries = Object.keys(platforms).map(platform => ({
  platform,
  directory: `npm/platforms/${platform}`,
  name: `${root.name}-${platform}`,
}));
entries.push({ directory: "npm", name: root.name });
function exists(name) {
  const result = spawnSync("npm", ["view", `${name}@${version}`, "version", "--json", "--registry=https://registry.npmjs.org"], { encoding: "utf8" });
  if (result.error) throw result.error;
  let data;
  try { data = JSON.parse(result.stdout); } catch { throw new Error(`Invalid registry response for ${name}: ${result.stderr || result.stdout}`); }
  if (result.status === 0 && data === version) return true;
  if (result.status !== 0 && data?.error?.code === "E404") return false;
  throw new Error(`Registry lookup failed for ${name}: ${JSON.stringify(data)}`);
}
function npm(args, directory) {
  const result = spawnSync("npm", args, { cwd: directory, stdio: "inherit" });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`npm ${args.join(" ")} failed in ${directory}`);
}

switch (process.argv[2]) {
  case "check": {
    if (process.env.GITHUB_EVENT_NAME === "push" && process.env.GITHUB_REF_TYPE !== "tag" && !/^0+$/.test(process.env.BEFORE_SHA || "")) {
      const previous = spawnSync("git", ["show", `${process.env.BEFORE_SHA}:npm/package.json`], { encoding: "utf8" });
      if (previous.status !== 0) throw new Error("Cannot read npm/package.json before this push");
      if (JSON.parse(previous.stdout).version === version) {
        if (process.env.GITHUB_OUTPUT) appendFileSync(process.env.GITHUB_OUTPUT, `version=${version}\nvalidate=false\n`);
        console.log("Package version is unchanged; publishing is skipped.");
        break;
      }
    }
    const published = entries.map(entry => ({ name: entry.name, exists: exists(entry.name) }));
    const validate = dryRun || process.env.GITHUB_REF_TYPE === "tag" || published.some(entry => !entry.exists);
    if (process.env.GITHUB_OUTPUT) appendFileSync(process.env.GITHUB_OUTPUT, `version=${version}\nvalidate=${validate}\n`);
    console.log(JSON.stringify({ version, distTag, dryRun, validate, packages: published }));
    break;
  }
  case "prepare": {
    root.optionalDependencies ??= {};
    for (const entry of entries) {
      const manifestPath = `${entry.directory}/package.json`;
      const manifest = entry.platform ? JSON.parse(readFileSync(manifestPath, "utf8")) : root;
      manifest.name = entry.name;
      manifest.version = version;
      manifest.repository = { type: "git", url: "git+https://github.com/khanglvm/tool-hub-mcp.git" };
      if (entry.platform) {
        manifest.license = root.license;
        manifest.files = ["bin"];
        const binary = entry.platform === "win32-x64" ? "tool-hub-mcp.exe" : "tool-hub-mcp";
        mkdirSync(`${entry.directory}/bin`, { recursive: true });
        const destination = `${entry.directory}/bin/${binary}`;
        copyFileSync(`artifacts/npm-platform-${entry.platform}/tool-hub-mcp-${platforms[entry.platform]}`, destination);
        if (entry.platform !== "win32-x64") chmodSync(destination, 0o755);
        root.optionalDependencies[entry.name] = version;
      }
      writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
      npm(["pack", "--dry-run", "--json"], resolve(entry.directory));
    }
    break;
  }
  case "publish": {
    for (const entry of entries) {
      if (dryRun) {
        console.log(`${entry.name}@${version} passed package inspection. Dry run: nothing published.`);
      } else if (exists(entry.name)) {
        console.log(`${entry.name}@${version} already exists; skipping.`);
      } else {
        npm(["publish", "--provenance", "--access", "public", "--tag", distTag], resolve(entry.directory));
      }
    }
    break;
  }
  default: throw new Error("Use check, prepare, or publish");
}
