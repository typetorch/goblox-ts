#!/usr/bin/env node
"use strict";

// The goblox bin: runs the platform binary with the same arguments and stdio, and exits like it did.

const { spawn } = require("child_process");
const { getBinaryPath } = require("../lib/index.js");

let binary;
try {
    binary = getBinaryPath();
}
catch (error) {
    console.error(`goblox: ${error.message}`);
    process.exit(1);
}

const child = spawn(binary, process.argv.slice(2), { stdio: "inherit", windowsHide: true });

// Pass signals on: a killed launcher must not leave the binary running (a second watcher writing to out/).
for (const signal of ["SIGINT", "SIGTERM", "SIGHUP"]) {
    process.on(signal, () => {
        child.kill(signal);
    });
}

child.on("error", error => {
    console.error(`goblox: could not run ${binary}: ${error.message}`);
    process.exit(1);
});

child.on("exit", (code, signal) => {
    if (signal) {
        // Die from the same signal, so the parent sees what the binary saw (e.g. Ctrl+C in watch mode).
        process.removeAllListeners(signal);
        process.kill(process.pid, signal);
        return;
    }
    process.exit(code ?? 1);
});
