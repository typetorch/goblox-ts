# The goblox CLI

`goblox` is a drop-in replacement for `rbxtsc` (roblox-ts 3.0.0). It takes the same flags, reads the same
tsconfig.json and `rbxts` options, writes the same `out/` and include folder, prints diagnostics in the same
format and exits with the same codes. `typetorch build` (typetorch/cli#36) uses it automatically when it is
installed.

## Packages

| npm package | What it is |
| --- | --- |
| `goblox` | The primary package (unscoped). Its `bin` is `goblox`. It installs the right platform package through `optionalDependencies`. |
| `@typetorch/goblox` | An alias that depends on `goblox` and declares the same `bin`, so `node_modules/.bin/goblox` exists with Bun and npm whichever one you install. |
| `@typetorch/goblox-<os>-<arch>` | The native binary for one platform: `win32-x64`, `win32-arm64`, `darwin-x64`, `darwin-arm64`, `linux-x64` and `linux-arm64`. Each has `os` and `cpu` fields, so only one is installed. |

The binary is called `goblox` (`goblox.exe` on Windows) in every package.

```sh
bun add -d goblox   # or: npm i -D goblox
```

### How typetorch finds goblox

From TypeTorch/cli `src/compiler.ts`. The first match wins:

1. `TYPETORCH_COMPILER`, or `"compiler"` in typetorch.json (`"goblox"` or `"rbxtsc"`).
2. goblox, if `node_modules/.bin/goblox` exists (`goblox.exe` or `goblox.cmd` on Windows).
3. Otherwise rbxtsc.

The version in the build cache key comes from `node_modules/goblox/package.json` (or `@typetorch/goblox`'s).
Failing that, it is the first line of `goblox --version`.

## Invocation

```
goblox [build] [flags]     build the project (the default command, like `rbxtsc` / `rbxtsc build`)
goblox watch [flags]       same as `goblox build -w`
goblox check [flags]       type check only, with timings (benchmarking, not part of the rbxtsc contract)
goblox -v | --version      goblox 0.1.0 (roblox-ts 3.0.0 compatible, typescript 7.1.0)
goblox -h | --help         flags
```

Flags are parsed like yargs parses them for rbxtsc: values follow either `=` or a space (`--type=game`,
`--type game`), booleans also take `--flag false` and `--no-flag`, `--kebab-case` names work, and a string
flag without a value is the empty string. One difference: a single dash name that is a whole flag
(`-verbose`) is that flag, where yargs reads the letters `-v -e -r ...`. `build` has to be the first argument.

### What typetorch runs

- **`typetorch build`** runs `bun run goblox`. If the package.json `build` script ends in goblox, it runs
  `bun run build` instead. It adds `-p .typetorch/<channel>.tsconfig.json` when it generated a prod
  tsconfig, and passes no other flags.
  - The environment includes `TYPETORCH_DEBUG_MACROS=0|1`. The built-in `@typetorch/transformer` reads it:
    `0` strips `$print`/`$warn`/`$dbg`.
  - It also sets `TYPETORCH_SKIP_BUILD_INFO=1` and `TYPETORCH_CHANNEL`. Those are for the repo's own scripts;
    goblox ignores them.
- **`typetorch serve`** runs the game's `watch` script, or `bun run rbxtsc -w` when there is none. To use
  goblox there, set `"watch": "goblox -w"`. serve parses the TypeScript watch status lines (see Watch mode
  below), and goblox prints them identically.
- **`typetorch doctor`** warns when tsconfig lists JS plugins that goblox will skip.

### Flags

Every `rbxtsc build` flag is accepted. Only flags given on the command line override the tsconfig's `rbxts`
options, as in roblox-ts.

| Flag | rbxtsc | goblox |
| --- | --- | --- |
| `-p`, `--project <path>` | tsconfig.json, or a folder at or below the one that has it | same |
| `-w`, `--watch` | watch mode | same (see Watch mode) |
| `--usePolling` | chokidar polling | accepted; goblox always polls |
| `--verbose` | verbose logs | same, plus a `<step> ( <n> ms )` line for every build step (parse tsconfig, create program, project outputs, rojo, transformers, type check, transform and render, write, copy) |
| `--noInclude` | don't copy the runtime library | same |
| `--logTruthyChanges` | log truthiness differences | same |
| `--writeOnlyChanged` | (hidden) write only outputs whose content changed | same; on by default in watch mode |
| `--writeTransformedFiles` | (hidden) write the plugin-transformed TypeScript | same |
| `--optimizedLoops` | (hidden) | same |
| `--type game\|model\|package` | override the project type | same, including the "Invalid values" error |
| `-i`, `--includePath <dir>` | runtime library folder | same |
| `--rojo <file>` | Rojo project file | same |
| `--allowCommentDirectives` | (hidden) allow `@ts-ignore` | same |
| `--noIncremental` | no | goblox only: compile every file, ignoring what the last build left |
| `--emitFirst` | no | goblox only, watch mode only: errors don't block the emit (see Watch mode) |
| `--luau` | `.luau` extension | same |
| `--logTruncatedPaths` | not in roblox-ts 3.0.0 | accepted and ignored (goblox never truncates paths) |
| `--checkers <n>` | – | goblox only: type checker count. Default: 4 with 8 or more threads, otherwise min(4, physical cores). Physical cores are read on Linux and macOS; elsewhere half the threads are assumed. `GOMAXPROCS` counts as the thread count |
| `--timing` | – | goblox only: print `Build time:` on stderr |

### Output and exit codes

- Diagnostics are written to stdout through `ts.createDiagnosticReporter(ts.sys, true)`. That means the
  colored, pretty format with source context, always colored, as in rbxtsc. roblox-ts errors print as
  `error TS roblox-ts: <message>`, and their messages are byte-identical.
- **0**: the build had no error diagnostics. Warnings don't change the exit code.
- **1**: one of the following.
  - An error diagnostic (TypeScript or roblox-ts).
  - A CLI or project error, such as `Unable to find tsconfig.json!`, or a Rojo project file that can't be read.
  - A flag check that yargs would fail, printed on stdout as yargs prints it: `Unknown argument: x`, `Invalid
    values:` for `--type`, and `Missing dependent arguments:` for `--usePolling` without `--watch`.
  - An internal error.
- When a file's transform panics (an unported construct), the file gets an error diagnostic, the panic is
  printed on stderr as `goblox: <file>: <panic>`, and the exit code is 1. As in roblox-ts, nothing is written
  when there are errors.
- `--version` prints a single line, `goblox <version> (roblox-ts 3.0.0 compatible, typescript <version>)`.
  rbxtsc prints only its version number. typetorch takes the first line.

`go run ./cmd/goblox-fixtures -oracle <oracle> -cli` compares both compilers on these invocations: exit code,
stdout, stderr and every file written. It is byte-for-byte, except for these intentional differences, where only
the exit code is compared:

- `--version`, `--help` and `--verbose` print goblox's own text (see above).
- A tsconfig.json that isn't valid JSON is reported by typescript-go's parser, which can pick another code than
  TypeScript 5.9's (TS1005 for TS1136).
- A Rojo project file that isn't valid JSON gets the same `Unable to read Rojo project` error, but the text after
  `SyntaxError:` is Go's, not V8's.
- Project references are not built: rbxtsc's `ProjectGraph.ts` builds each referenced project first, goblox reports
  TS6305 until their outputs exist. Build them with their own `goblox -p <project>` first.

The contents of a `.d.ts.map` aren't compared, as typescript-go's source map generator picks other mappings. See
[CONFORMANCE.md](CONFORMANCE.md) for the results.

### Watch mode

roblox-ts watches with chokidar and rebuilds only the changed files. goblox polls file modification times
under the tsconfig's folder every 250 ms. It skips `node_modules`, `.git` and the build's own outputs (outDir,
declarationDir and the include folder). It waits 100 ms for changes to settle and then compiles the files the
changes can reach (see Incremental builds).

- Polling needs no file watcher dependency. fsnotify would be a new direct module requirement in go.mod.
- Files whose modification time and size did not change are not read, parsed or bound again: the watcher
  keeps them between rebuilds, like `tsc -w` (gnome-game: create program 138 ms → ~47 ms per rebuild).
- The first build uses the cache file of the last one-shot or watch build, and every build writes it, when the
  tsconfig has `incremental`. Without it, the state stays in memory and the first build is a full one.
- `--writeOnlyChanged` is on by default in watch mode, so Rojo sees only outputs whose content changed.
  `--writeOnlyChanged=false` turns it off.
- `--emitFirst` (goblox only, watch mode only) doesn't let errors block the emit. Files without errors are written,
  a file with errors keeps its last output, and the errors are reported after the writes. The affected files are
  still type-checked first, because the transform needs the checker's results (import elision), so this saves the
  wait for you to fix an unrelated error, not the check. Without the flag, errors block the emit, as in roblox-ts.

The status lines are `ts.createWatchStatusReporter(ts.sys, true)`'s (time in grey, en-US locale format,
followed by a blank line):

```
[12:52:20 AM] Starting compilation in watch mode...
[12:52:21 AM] File change detected. Starting incremental compilation...
src/bad.ts:1:7 - error TS2322: Type 'number' is not assignable to type 'string'.
[12:52:22 AM] Found 1 error. Watching for file changes.
```

A build that fails before compiling, such as a broken rojo project, counts as 1 error, and watching
continues. In rbxtsc that kills the watcher. When there is no tsconfig.json, goblox exits with 1.

### Incremental builds and tsbuildinfo

roblox-ts writes `tsBuildInfoFile` and recompiles the changed files plus their direct importers. goblox recompiles
the changed files plus the importers of files whose d.ts signature changed, using typescript-go's own incremental engine
(`internal/roblox/project/incremental.go`). It leaves the tsbuildinfo path itself alone, and keeps its state in
`<tsBuildInfoFile minus extension>.goblox.tsbuildinfo` (`out/tsconfig.goblox.tsbuildinfo`). This needs `incremental` or
`composite` in the tsconfig, like tsc. `--noIncremental` (goblox only) compiles every file and doesn't touch the cache.

**What is compiled.** A file is compiled again when its text changed, when the d.ts signature of something it depends on
changed (transitively, through importers), when a global-scope file changed, when its last build reported a warning for
it, or when its output in `out/` is missing or isn't what goblox wrote. So a comment or a function body in
`shared/net.ts` compiles one file, not its 183 importers. There are two states: the original program decides
which files the tsconfig `plugins` run on, because their output depends on types, and the program of the plugins' output
decides what is checked and transformed. The other files reuse their printed text and their output.

**Errors.** A build with an error writes nothing and changes no state, so the next build compares with the same last good
build, compiles the same files and reports the same errors until they are fixed, also in files you didn't edit.

**Full builds.** Everything is compiled again when any of these changed: goblox (version, revision, executable size and
time), the compiler options, tsconfig and what it extends, the CLI options, `package.json` and lockfiles from the
project up to the root, the Rojo project (and `*.project.json` files copied to the output), the plugin entries,
`typetorch.json`, `TYPETORCH_DEBUG_MACROS`. Also when the Rojo resolvers answer differently. A missing, corrupt or old
cache file means a full build, never an error. These always compile everything: `--logTruthyChanges`,
`--writeTransformedFiles`, and the `removeComments`, `declaration` and `assumeChangesOnlyAffectDirectDependencies`
compiler options. With tsconfig `plugins`, a diagnostic (a warning too) in a file the plugins didn't print also means a
full compile of that build, because rbxtsc reports positions in the printed text. That compile saves no state, so the
next build compares with the last good one.

**Output.** The result is byte-identical to a clean build, apart from volatile macros: `$git()`, `$compileTime()` and UUID
user macros keep the value of the build that last compiled their file, as in rbxtsc. Delete the cache file, or use
`--noIncremental`, to refresh all of them. A generated file that changes each time (`shared/build.ts` in typetorch
projects) is compiled each time, and its importers only when its types change.

typetorch's build cache restores `out/` and then runs goblox; the cache file is restored with it. If the outputs don't
match what it says, those files are compiled again. typetorch adds `compiler goblox <version>` to the cache key, so
switching compilers never reuses the other compiler's state.

### tsconfig `plugins`

goblox can't run JavaScript transformers. Plugins are Go packages registered with
`goblox.RegisterPlugin(name, factory)`, where `name` is the npm package of the JS transformer they port (see
PLUGINS.md).

- **`@typetorch/transformer`** maps to the built-in Go port (`github.com/typetorch/transformer/go`). It gets
  the tsconfig entry's options, like the JS plugin. If the binary was built without the port, this is an
  error: building without the transformer would emit calls to macros that don't exist.
- **Any other `transform` entry** (`rbxts-transform-debug`, `rbxts-transformer-flamework`, ...) gets one
  warning per process (not one per watch rebuild), `goblox skips the tsconfig plugin "<name>": ...`, and is
  skipped. To keep such plugins, set `"compiler": "rbxtsc"` in typetorch.json.
- Entries without `transform`, which are language service plugins, are ignored, as in tsc.

## Linking @typetorch/transformer into the release binary

Today the port is linked only by `-tags typetorch` in a go.work that has both modules
(`cmd/goblox/plugins_typetorch.go`):

```sh
go work init ./goblox-ts ./typetorch-transformer/go
go build -tags typetorch -o goblox ./goblox-ts/cmd/goblox
```

A plain `go build ./cmd/goblox` in goblox-ts has no transformer. Every gnome-game build needs it, because 87
files use `$print`/`$warn`. The release binary must therefore always link it. There are three ways to do
that.

### A. Release CI builds in a go.work (recommended now: no go.mod change in either repo)

The release workflow checks out `typetorch/goblox-ts` and `typetorch/transformer` side by side.

- It pins the transformer to a tag (`v0.3.x`), recorded in the workflow.
- It runs `go work init ./goblox-ts ./transformer/go`, then for each platform:
  `GOOS=… GOARCH=… go build -tags typetorch -trimpath -ldflags "-s -w -X main.version=$NPM_VERSION"`.
- It packs each binary into `@typetorch/goblox-<os>-<arch>`.

Notes:
- In workspace mode, the transformer's `replace github.com/microsoft/typescript-go => ../../goblox-ts` is
  ignored and the workspace's goblox-ts is used. This is exactly the setup that produced the gnome-game numbers.
- The build tag stays: contributors who build without the transformer checkout still get a working goblox.
- `--version` could append `+typetorch <tag>`.

Cost:
- `go install github.com/microsoft/typescript-go/cmd/goblox` does not give you the plugin. This is fine,
  since goblox ships through npm.
- The CI needs read access to both repos.

### B. goblox-ts requires the transformer module (needs go.mod changes: ask first)

- Add `require github.com/typetorch/transformer/go v0.3.x` to goblox-ts's go.mod and go.sum.
- Drop the build tag from `plugins_typetorch.go`.

The module cycle (goblox-ts → transformer → typescript-go) is legal. In goblox-ts the main module is
typescript-go itself, so the transformer's own requirement on `github.com/microsoft/typescript-go v0.0.0`
(and its replace) is ignored.

Costs:
- The transformer must be published as a Go module. That means `go/vX.Y.Z` tags in typetorch/transformer, or
  GOPRIVATE plus auth if the repo is private.
- goblox-ts's go.mod (an upstream file) gets a non-upstream requirement. Every upstream merge has to keep it.
- A transformer change needs a tag and a go.mod bump in goblox-ts before a release sees it.

### C. A separate release module (largest change)

- Make the CLI importable: move `cmd/goblox`'s body into `pkg/goblox/cli` with `func Main(args []string) int`.
- Add a tiny `main` module, e.g. `typetorch/transformer/cmd/goblox`, that imports both and calls `cli.Main`.

Costs:
- This needs a go.mod for the new module, plus `replace` for the fork, since its module path is
  `github.com/microsoft/typescript-go`.
- The binary is no longer built from goblox-ts.

**Proposal:** A for the first releases. It works today, it is what was benchmarked, and it touches no go.mod.
B only if `go install` with the plugin becomes a requirement, and only after the user approves the go.mod
change in both repos.

### npm packaging (for the release workflow)

- **Platform packages:** each `@typetorch/goblox-<os>-<arch>` has `bin/goblox[.exe]`, plus `os`/`cpu` in its
  package.json.
- **`goblox`:**
  - `optionalDependencies` on all six platform packages, at the same version.
  - `"bin": { "goblox": "bin/goblox.js" }`, a shim that resolves the installed platform package and runs its
    binary with inherited stdio, forwarding the exit code and signals.
  - A `postinstall` may replace the shim with the binary to skip Node startup, as esbuild does. The shim has
    to stay as the fallback for `--ignore-scripts`.
- **`@typetorch/goblox`:** `"dependencies": { "goblox": "<same version>" }` and the same `bin` (a one-line
  shim that requires `goblox/bin/goblox.js`). Bun links bins only for direct dependencies, so the alias needs
  its own.
