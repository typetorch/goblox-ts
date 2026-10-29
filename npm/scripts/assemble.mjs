#!/usr/bin/env node
// Assembles the npm packages of a goblox release from built binaries.
//
//   node npm/scripts/assemble.mjs --version 1.2.3 --binaries dist/bin --out dist/npm [--allow-missing]
//
// --binaries holds one folder per platform, named like the npm suffix: linux-x64/goblox,
// win32-x64/goblox.exe, ... --out gets one folder per package plus publish-order.txt, which lists them in the
// order they must be published: the platform packages, then goblox, then @typetorch/goblox.
//
// --allow-missing skips platforms without a binary, and leaves them out of goblox's optionalDependencies
// (for local tests with only some platforms built). Releases never pass it.

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";

const npmDir = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const repoDir = path.dirname(npmDir);

// npm suffix -> os/cpu fields (Node's process.platform / process.arch).
const PLATFORMS = [
	["win32-x64", "win32", "x64"],
	["win32-arm64", "win32", "arm64"],
	["linux-x64", "linux", "x64"],
	["linux-arm64", "linux", "arm64"],
	["darwin-x64", "darwin", "x64"],
	["darwin-arm64", "darwin", "arm64"],
];

const { values: args } = parseArgs({
	options: {
		version: { type: "string" },
		binaries: { type: "string" },
		out: { type: "string" },
		"allow-missing": { type: "boolean", default: false },
	},
});
if (!args.version || !args.binaries || !args.out) {
	console.error("usage: assemble.mjs --version <semver> --binaries <dir> --out <dir> [--allow-missing]");
	process.exit(2);
}
const version = args.version.replace(/^v/, "");
// semver 2.0.0, as npm accepts it.
if (!/^\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$/.test(version)) {
	console.error(`assemble: "${version}" is not a semver version`);
	process.exit(2);
}

const out = path.resolve(args.out);
fs.rmSync(out, { recursive: true, force: true });
fs.mkdirSync(out, { recursive: true });

const readJson = (file) => JSON.parse(fs.readFileSync(file, "utf8"));
const writeJson = (file, value) => fs.writeFileSync(file, `${JSON.stringify(value, null, "\t")}\n`);
const license = path.join(repoDir, "LICENSE");

const launcher = readJson(path.join(npmDir, "goblox", "package.json"));
const shared = {
	homepage: launcher.homepage,
	bugs: launcher.bugs,
	license: launcher.license,
};

const order = [];
const optionalDependencies = {};
for (const [suffix, os, cpu] of PLATFORMS) {
	const exe = os === "win32" ? "goblox.exe" : "goblox";
	const binary = path.join(args.binaries, suffix, exe);
	if (!fs.existsSync(binary)) {
		if (args["allow-missing"]) {
			console.warn(`assemble: no binary for ${suffix} (${binary}), skipped`);
			continue;
		}
		console.error(`assemble: missing ${binary}`);
		process.exit(1);
	}

	const name = `@typetorch/goblox-${suffix}`;
	const dir = path.join(out, `goblox-${suffix}`);
	fs.mkdirSync(path.join(dir, "bin"), { recursive: true });
	fs.copyFileSync(binary, path.join(dir, "bin", exe));
	fs.chmodSync(path.join(dir, "bin", exe), 0o755);
	fs.copyFileSync(license, path.join(dir, "LICENSE"));
	writeJson(path.join(dir, "package.json"), {
		name,
		version,
		description: `The ${suffix} binary of goblox, the Go port of the roblox-ts compiler. Install goblox instead.`,
		...shared,
		repository: { ...launcher.repository, directory: "npm" },
		os: [os],
		cpu: [cpu],
		files: ["bin"],
		// No "bin": the goblox package runs this binary through require.resolve("<name>/bin/<exe>").
		preferUnplugged: true,
	});
	fs.writeFileSync(
		path.join(dir, "README.md"),
		`# ${name}\n\nThe ${suffix} binary of [goblox](https://www.npmjs.com/package/goblox). ` +
			"Don't install this package directly: install `goblox`, which picks the right platform package.\n",
	);
	optionalDependencies[name] = version;
	order.push(dir);
}

function copyPackage(srcName, dstName, edit) {
	const src = path.join(npmDir, srcName);
	const dst = path.join(out, dstName);
	fs.cpSync(src, dst, { recursive: true });
	fs.copyFileSync(license, path.join(dst, "LICENSE"));
	const pkg = readJson(path.join(dst, "package.json"));
	pkg.version = version;
	edit(pkg);
	writeJson(path.join(dst, "package.json"), pkg);
	for (const bin of Object.values(pkg.bin ?? {})) fs.chmodSync(path.join(dst, bin), 0o755);
	order.push(dst);
}

copyPackage("goblox", "goblox", (pkg) => {
	pkg.optionalDependencies = optionalDependencies;
});
copyPackage("typetorch-goblox", "typetorch-goblox", (pkg) => {
	pkg.dependencies.goblox = version;
});

fs.writeFileSync(path.join(out, "publish-order.txt"), `${order.map((dir) => path.basename(dir)).join("\n")}\n`);
console.log(`assemble: goblox ${version}, ${order.length} packages in ${out}`);
