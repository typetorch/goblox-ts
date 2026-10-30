#!/usr/bin/env node
"use strict";

// The goblox bin: runs the platform binary with the same arguments and stdio, and exits like it did.

const { spawnSync } = require("child_process");
const { getBinaryPath } = require("../lib/index.js");

let binary;
try {
    binary = getBinaryPath();
}
catch (error) {
    console.error(`goblox: ${error.message}`);
    process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit", windowsHide: true });
if (result.error) {
    console.error(`goblox: could not run ${binary}: ${result.error.message}`);
    process.exit(1);
}
if (result.signal) {
    // Die from the same signal, so the parent sees what the binary saw (e.g. Ctrl+C in watch mode).
    process.kill(process.pid, result.signal);
}
process.exit(result.status ?? 1);
