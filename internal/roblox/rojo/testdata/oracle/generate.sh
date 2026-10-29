#!/bin/sh
# Regenerates ../expected.json with the reference TypeScript implementations.
#   ./generate.sh <dir with rojo-resolver and path-translator checkouts> <tsc> <node_modules with ajv and fs-extra>
set -e
here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
mkdir -p "$work/rojo-resolver" "$work/path-translator"
cp -r "$1/rojo-resolver/src/." "$work/rojo-resolver/"
cp "$1/rojo-resolver/rojo-schema.json" "$work/"
cp -r "$1/path-translator/src/." "$work/path-translator/"
cp "$here/oracle.ts" "$work/"
printf '{ "compilerOptions": { "module": "commonjs", "target": "es2019", "lib": ["es2019"], "types": [], "esModuleInterop": true, "outDir": "out", "rootDir": "." }, "include": ["**/*.ts"], "exclude": ["out"] }\n' > "$work/tsconfig.json"
# no @types/node here; the sources compile to working JavaScript regardless of type errors
(cd "$work" && "$2" -p . > /dev/null) || test -f "$work/out/oracle.js"
# RojoResolver reads its schema from path.join(__dirname, "..")
cp "$work/rojo-schema.json" "$work/out/"
NODE_PATH="$3" node "$work/out/oracle.js" "$(cd "$here/.." && pwd)" > "$here/../expected.json"
rm -rf "$work"
