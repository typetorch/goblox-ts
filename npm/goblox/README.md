# goblox

A Go port of the [roblox-ts](https://roblox-ts.com) compiler (TypeScript to Luau), built on
[typescript-go](https://github.com/microsoft/typescript-go). `goblox` is a drop-in replacement for `rbxtsc`:
same flags, same tsconfig options, same output.

```sh
npm i -D goblox     # or: bun add -d goblox
npx goblox          # like rbxtsc
npx goblox -w       # like rbxtsc -w
```

`typetorch build` uses goblox automatically when it is installed.

The native binary comes from one of the `@typetorch/goblox-<os>-<arch>` packages (Windows, macOS and Linux,
x64 and arm64), installed through `optionalDependencies`. Don't install with optional dependencies turned
off. `GOBLOX_BINARY=/path/to/goblox` makes the `goblox` bin run a local build instead.

Docs: [docs/goblox](https://github.com/typetorch/goblox-ts/tree/goblox/docs/goblox).
