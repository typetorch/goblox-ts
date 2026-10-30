"use strict";

// Finds the goblox binary for this machine. The binary ships in one of the @typetorch/goblox-<os>-<arch>
// packages, which are optionalDependencies of this package: the package manager installs only the one whose
// os/cpu fields match (the esbuild/biome layout).

const fs = require("fs");

const PLATFORMS = {
    "win32-x64": "@typetorch/goblox-win32-x64",
    "win32-arm64": "@typetorch/goblox-win32-arm64",
    "linux-x64": "@typetorch/goblox-linux-x64",
    "linux-arm64": "@typetorch/goblox-linux-arm64",
    "darwin-x64": "@typetorch/goblox-darwin-x64",
    "darwin-arm64": "@typetorch/goblox-darwin-arm64",
};

function platformKey() {
    return `${process.platform}-${process.arch}`;
}

function binaryName() {
    return process.platform === "win32" ? "goblox.exe" : "goblox";
}

/** The path of the goblox binary. Throws an Error with an install hint when there is none. */
function getBinaryPath() {
    // For local goblox builds: GOBLOX_BINARY=/path/to/goblox npx goblox ...
    const override = process.env.GOBLOX_BINARY;
    if (override) {
        if (!fs.existsSync(override)) throw new Error(`GOBLOX_BINARY points to ${override}, which does not exist.`);
        return override;
    }

    const key = platformKey();
    const pkg = PLATFORMS[key];
    if (!pkg) {
        throw new Error(
            `goblox has no binary for ${key}. Supported platforms: ${Object.keys(PLATFORMS).join(", ")}.\n` +
                `Use rbxtsc instead (with typetorch: set "compiler": "rbxtsc" in typetorch.json).`,
        );
    }

    const subpath = `bin/${binaryName()}`;
    try {
        return require.resolve(`${pkg}/${subpath}`);
    }
    catch {
        // Not installed (or installed under a different node_modules than this package can see).
    }

    throw new Error(
        `The goblox binary for ${key} is missing: the package "${pkg}" is not installed.\n` +
            `goblox installs it through optionalDependencies. This happens when they were skipped\n` +
            `(npm --omit=optional / --no-optional, yarn --ignore-optional), when node_modules was copied from\n` +
            `another OS or CPU, or when the lockfile was created on another platform without it.\n` +
            `Fix: reinstall without skipping optional dependencies, or install it directly:\n` +
            `  npm i -D ${pkg}@${require("../package.json").version}`,
    );
}

module.exports = { PLATFORMS, getBinaryPath, platformKey };

if (require.main === module) console.log(getBinaryPath());
