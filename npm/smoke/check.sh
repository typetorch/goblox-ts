#!/usr/bin/env bash
# Compiles this fixture with the goblox given as $1 (default: goblox on PATH) and checks the output.
set -euo pipefail
cd "$(dirname "$0")"
goblox="${1:-goblox}"
[ -d node_modules/@rbxts/compiler-types ] || npm install --no-audit --no-fund --no-package-lock
rm -rf out include
"$goblox" --verbose
for f in out/shared/greet.luau out/server/main.server.luau include/RuntimeLib.luau; do
	[ -s "$f" ] || { echo "smoke: $f was not written" >&2; exit 1; }
done
grep -q 'TS.import' out/server/main.server.luau || { echo "smoke: main.server.luau has no TS.import" >&2; exit 1; }
echo "smoke: ok"
