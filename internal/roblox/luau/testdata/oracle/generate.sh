#!/bin/sh
# Regenerates ../render.json with the reference TypeScript renderer.
#   ./generate.sh <luau-ast checkout (2.1.2)> <path to a TypeScript 5.x tsc>
# luau-ast's modules import each other in a cycle that only works as CommonJS, so the sources are
# compiled to CommonJS and run with node (bare "LuauAST/..." imports resolve through NODE_PATH).
set -e
here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
cp -r "$1/src/." "$work/"
cp "$here/cases.ts" "$here/render.ts" "$work/"
printf '{ "compilerOptions": { "module": "commonjs", "target": "es2019", "lib": ["es2019"], "types": [], "strict": true, "esModuleInterop": true, "baseUrl": ".", "paths": { "*": ["*"] }, "outDir": "out", "rootDir": "." }, "include": ["**/*.ts"], "exclude": ["out"] }\n' > "$work/tsconfig.json"
# The cases are loosely typed (luau.list.make infers from its first element); emit regardless.
(cd "$work" && "$2" -p . --declaration false --sourceMap false > /dev/null) || test -f "$work/out/render.js"
NODE_PATH="$work/out" node "$work/out/render.js" > "$here/../render.json"
rm -rf "$work"
