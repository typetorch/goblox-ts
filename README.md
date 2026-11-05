# goblox

**A roblox-ts fork in Go.** goblox is the [roblox-ts](https://roblox-ts.com) compiler (TypeScript to Luau) ported
to Go on top of [typescript-go](https://github.com/microsoft/typescript-go), the native TypeScript 7 compiler. It is a
drop-in replacement for `rbxtsc`: same flags, same `tsconfig.json` and `rbxts` options, same `out/` folder, same Luau.

A 40-second rbxtsc build of a ~500-file game takes 5 seconds with goblox, and an edit rebuilds in under a second.

```sh
npm i -D goblox     # or: bun add -d goblox
npx goblox          # instead of rbxtsc
npx goblox -w       # instead of rbxtsc -w
```

The npm package ships prebuilt native binaries for Windows, macOS and Linux (x64 and arm64), so you don't need Go to
use it. [`typetorch`](https://github.com/typetorch/cli) uses goblox automatically when it is installed.

## Performance

gnome-game is a real Roblox game with 487 `.ts`/`.tsx` files (~80k lines), using Flamework-style DI and
`@typetorch/transformer`. The table shows medians of 3 runs, wall clock from process start to exit.

**Laptop**: Intel i5-4210U (2 cores / 4 threads), 3.7 GB RAM, Linux.

| Scenario | rbxtsc | goblox | Speedup |
| --- | ---: | ---: | ---: |
| Fresh build (no `out/`) | 39.10 s | 4.95 s | **7.9×** |
| Rebuild, nothing changed | 4.28 s | 0.51 s | **8.4×** |
| Comment edited in a leaf file | 5.05 s | 0.88 s | **5.7×** |
| Comment edited in a widely imported file | 28.85 s | 0.80 s | **36×** |
| New export in a widely imported file | 28.80 s | 3.93 s | **7.3×** |
| Peak memory, fresh build | 885 MB | 650 MB | 0.73× |

**Desktop**: 12 threads, 32 GB RAM, Windows. These runs used an older 407-file snapshot of the same game.

| Scenario | rbxtsc | goblox | Speedup |
| --- | ---: | ---: | ---: |
| Fresh build | 18.65 s | 2.11 s | **8.8×** |
| Rebuild, nothing changed | 5.73 s | 0.43 s | **13×** |
| Comment in a widely imported file | 14.85 s | 0.62 s | **24×** |
| New export in a leaf file | 6.35 s | 0.60 s | **11×** |
| New export in a widely imported file (160 files) | 14.74 s | 1.59 s | **9.3×** |

In watch mode (`goblox -w`), the time from saving a file to "Found 0 errors" is ~0.6 s for a leaf edit and ~1.5 s
for an edit that recompiles 160 files, polling included.

Where the speed comes from:

- **Native, parallel type checking.** typescript-go runs the checker in parallel on native code, so there is no Node
  startup or JIT warm-up.
- **The plugin runs in-process.** `@typetorch/transformer` is ported to Go and runs inside the compiler, in parallel
  across files.
- **Incremental builds that skip comment-only edits.** goblox recompiles only the files whose output can change. An
  edit that only changes comments in a shared file recompiles nothing that depends on it. rbxtsc recompiles all 183
  of that file's dependents.

Full methodology, profiles and step breakdowns:
[docs/goblox/BENCHMARKS.md](docs/goblox/BENCHMARKS.md) and
[docs/goblox/BENCHMARKS-espresso.md](docs/goblox/BENCHMARKS-espresso.md).

## Compatibility

goblox is checked against rbxtsc built from roblox-ts 3.0.0 (see [CONFORMANCE.md](docs/goblox/CONFORMANCE.md)):

| Check | Result |
| --- | --- |
| roblox-ts transform fixtures, Luau compared byte for byte | 73 / 73 |
| roblox-ts runtime specs, run under Lune | 874 / 874 |
| The same specs with `--optimizedLoops` | 874 / 874 |
| Diagnostic fixtures: code, message, span and printed text | 169 / 169 |
| CLI scenarios: exit code, stdout, stderr and files written | 64 (57 byte for byte, 7 intentional differences) |

Real games compile with the same imports and DI metadata as rbxtsc. Known differences are listed in
[CLI.md](docs/goblox/CLI.md).

## goblox-only options

| Flag | What it does |
| --- | --- |
| `--stripComments` | Leaves source comments and macro markers out of the output. The header and `--!` directives at the top of a file stay. Also available as `"rbxts": { "stripComments": true }` in tsconfig.json. Unlike `removeComments`, it keeps incremental builds |
| `--noIncremental` | Compiles every file, ignoring the incremental cache |
| `--emitFirst` | Watch mode: writes the files without errors, then reports the errors |
| `--checkers <n>` | Number of parallel type checkers (default: 4 with 8 or more threads) |
| `--timing` | Prints the build time |

Every compiled file starts with `-- goblox, a roblox-ts fork. goblox v<version>, roblox-ts v3.0.0`.

Incremental state lives next to the tsbuildinfo (`out/tsconfig.goblox.tsbuildinfo`), so set `"incremental": true`
in tsconfig.json.

## Plugins

TypeScript transformer plugins written in JS can't run inside a Go compiler. goblox compiles
[`@typetorch/transformer`](https://github.com/typetorch/transformer) into the release binary instead, ported to Go
(guards for networking, DI ids, `$print`/`$warn`/`$git` and user macros). Other JS plugins listed in tsconfig are
skipped with a warning, and `typetorch doctor` reports them. See [PLUGINS.md](docs/goblox/PLUGINS.md).

## Repository layout

goblox is a fork of typescript-go, because Go's `internal/` rule only lets code inside the module use the compiler
packages. The goblox code is:

| Path | Contents |
| --- | --- |
| `cmd/goblox` | The CLI |
| `internal/roblox/transform` | The roblox-ts transform (TypeScript AST to Luau AST) |
| `internal/roblox/luau` | Luau AST and renderer |
| `internal/roblox/project` | Builds, watch mode, incremental state, include files |
| `internal/roblox/rojo` | Rojo project resolution |
| `pkg/goblox` | The plugin API that `@typetorch/transformer` builds on |
| `cmd/goblox-fixtures` | The conformance harness against an rbxtsc oracle |
| `npm/` | The npm launcher and platform packages |

Everything else is upstream typescript-go. More docs are in [docs/goblox](docs/goblox): [CLI.md](docs/goblox/CLI.md),
[PORTING.md](docs/goblox/PORTING.md) and [RELEASING.md](docs/goblox/RELEASING.md).

## Building from source

```sh
git clone https://github.com/typetorch/goblox-ts && cd goblox-ts
go build -o goblox ./cmd/goblox                    # without the typetorch plugin
go build -tags typetorch -o goblox ./cmd/goblox    # with it (needs typetorch/transformer, see RELEASING.md)
```

Releases are built by CI when a `v*` tag is pushed. They publish `goblox`, `@typetorch/goblox` and the
`@typetorch/goblox-<os>-<arch>` binaries to npm.

## License

goblox is a fork of [typescript-go](https://github.com/microsoft/typescript-go) (Apache-2.0, see
[LICENSE](LICENSE) and [NOTICE.txt](NOTICE.txt)) and a port of [roblox-ts](https://github.com/roblox-ts/roblox-ts)
(MIT). It is not affiliated with Microsoft or the roblox-ts project.
