# goblox-ts benchmarks

## Phase 0: program + check (2026-10-29)

Machine: Windows 11 desktop, 12 threads, 32 GB RAM. Go 1.27.0, Node 24.22.1.

Project: gnome-game (407 `.ts`/`.tsx` files, ~63k lines; 513 files in the program with the
declaration files), built with roblox-ts 3.0.0, TypeScript 5.5.3 and @typetorch/transformer 0.3.2.
@rbxts/types 1.0.957, @rbxts/compiler-types 3.0.0-types.0.

To keep the repo clean, the benchmark ran on a copy of gnome-game without `.git`, with `node_modules`
as a junction to the real one, and with `src/shared/build.ts` written by `bun scripts/build-info.ts`.
The file is gitignored and is normally generated before every build.

Wall time and peak working set come from `measure.ps1`, which starts the process and polls
`PeakWorkingSet64` every 20 ms. The goblox phase times are its own `Program time` / `Check time` output.

### rbxtsc (cold: `out/` and `include/` deleted, so no tsbuildinfo)

`node node_modules/roblox-ts/out/CLI/cli.js --verbose`

| run | wall | peak memory |
| --- | ---: | ---: |
| 1 | 18.94 s | 888 MB |
| 2 | 18.81 s | 928 MB |
| 3 | 18.62 s | 795 MB |

Where the time goes (from `--verbose`, run 2):

| phase | time |
| --- | ---: |
| node startup, tsconfig, program creation (not logged; the remainder) | ~6.6 s |
| running transformers (@typetorch/transformer) | 1.59 s |
| re-print 407 transformed files | 0.54 s |
| re-create proxy program | 0.68 s |
| per-file compile: pre-emit diagnostics (the type check, done lazily per file) + Luau transform + render, summed | 8.64 s |
| writing compiled files | 0.20 s |

`tsc -p . --noEmit` with TypeScript 5.5.3 on the same project takes 8.2 s.

### goblox Phase 0 (program + full type check, no emit yet)

`goblox check -p .` (default of 4 checkers). The check covers every file in parallel, then collects the
diagnostics of the 407 compiled files, which is the set roblox-ts reports.

| run | wall | program | bind | check | peak memory |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1 (cold file cache) | 2.50 s | 715 ms | 23 ms | 1.04 s | 671 MB |
| 2 | 1.45 s | 106 ms | 24 ms | 1.02 s | 588 MB |
| 3 | 1.48 s | 104 ms | 24 ms | 0.99 s | 616 MB |

Result: 0 diagnostics in the compiled files, the same as rbxtsc.

Program creation plus type check is **~1.1 s against rbxtsc's ~15 s** for the same work (everything
except the transformer plugin, the Luau transform and writing). That is a 13x speedup end to end so far.
The Luau transform in roblox-ts is a small part of the 8.6 s per-file phase (most files take 0-20 ms,
and the type check dominates). So a full goblox build should land at 2-3 s, with the Go plugin and
emit on top of the 1.5 s measured here.

### @rbxts/types and @rbxts/compiler-types under TypeScript 7

Both load. With the compat layer in `internal/roblox/project/compat.go`, the compiled files have no
errors. The compat layer gives Iterable its three type parameters and drops the "removed option"
errors for `moduleResolution: Node` and `downlevelIteration`.

`goblox check -all` also checks the declaration files and reports 23 errors in `node_modules`:

- 12 in @rbxts/types `selector.d.ts`
- 2 in @rbxts/types `generated/None.d.ts`
- 9 in @rbxts/ui-labs and @rbxts/vide

`tsc` 5.5.3 reports exactly the same 23 errors. They are pre-existing library errors, not TS 7
regressions. roblox-ts never reports them, because it only asks for the diagnostics of the files it
compiles.

## First full `goblox build` of gnome-game (2026-10-30)

Same scratch copy and machine as Phase 0. goblox was built with the Go port of @typetorch/transformer
linked in (`-tags typetorch` through a `go.work`, see `cmd/goblox/plugins_typetorch.go`).
Command: `goblox build -p . -timing`, cold (`out/` deleted), 3 runs.

| run | wall | goblox `Build time` |
| --- | ---: | ---: |
| 1 | 6.7 s | 5.05 s |
| 2 | 5.2 s | 5.07 s |
| 3 | 9.3 s | 8.94 s (machine busy with other agents' builds) |

rbxtsc cold build: 18.6-18.9 s (Phase 0 above).

All 407 files went through the plugin and the Luau transform. One file, `mob-hud.controller.ts`, panicked
on a class `static {}` block (`transformClassLikeDeclaration` calls `declaration.Body()`, which is nil for
`ClassStaticBlockDeclaration` in typescript-go). As in rbxtsc, any error skips the write, so these runs
wrote no `out/`. The output has not been diffed against rbxtsc's yet. These numbers include the type check
and the transform, but not the write.

## gnome-game output diff against roblox-ts (2026-10-30)

After agent 3's class `static {}` fix, goblox builds gnome-game with no errors and writes all 407 files.
Command: `goblox build --timing` (`-tags typetorch`), cold, on the same scratch copy. Build time: 3.4-4.5 s.

**Which reference.** gnome-game's `node_modules/roblox-ts` is the npm 3.0.0 release. goblox ports roblox-ts
master (68648b0, also called 3.0.0), which emits different output: it drops temporaries (evaluation
effects), prints JSDoc comments differently, and orders array-macro callbacks differently. Against the npm
build only 70 of 407 files match, so that comparison is meaningless. The reference is gnome-game compiled
by the master oracle with the JS @typetorch/transformer:

```sh
# in the scratch copy; gen/ts-hook.cjs maps require("typescript") to the oracle's TS 5.9.3 so the JS
# transformer shares the compiler's instance (node_modules is the real project's, never written)
NODE_PATH=<oracle>/out node -r gen/ts-hook.cjs <oracle>/out/CLI/cli.js     # 25 s
```

| step | identical files |
| --- | ---: |
| first diff | 398 / 407 |
| + `transformMacroCall` uses `luau.IsCall` (method call arguments get the nil-coercing parens) | 399 / 407 |
| + the plugin reprint drops typescript-go's parens around `x!` in `x!.y` / `x![i]` / `x!()` | **406 / 407** |

The last file, `shared/build.luau`, differs only in `builtAt = $compileTime()`, which is the build
timestamp. Other differences:

- goblox writes no `tsconfig.rbxtsc.tsbuildinfo`. See CLI.md, "Incremental builds".
- The plugin-transformed TS (`--writeTransformedFiles`) differed only in those `x!` parens.

## Step timings in `--verbose` (perf/incremental milestone 1, 2026-10-30)

`goblox build --verbose` now prints roblox-ts's `benchmarkIfVerbose` lines (`<step> ( <n> ms )`) for every build
step. `-timing` still prints `Build time:` on stderr. This milestone changes no output: on a gnome-game copy (407
files) `out/` is byte-identical to the previous binary's apart from `builtAt` in `shared/build.luau`, and the oracle
harness has 73/73 matches.

One cold run on the Windows desktop (12 threads; other agents were building at the same time):

```
parse tsconfig and project data ( 12 ms )
create program ( 157 ms )                      parse + bind
copy include files ( 0 ms )
collect project outputs ( 42 ms )
check rojo config and file names ( 31 ms )
load rojo configs and package paths ( 1076 ms ) includes the check above
running transformers.. ( 658 ms )               @typetorch/transformer + reprint + proxy program
407/407 files re-printed
type check 407 files ( 478 ms )
transform and render 407 files ( 533 ms )
  of which luau render: 136 ms of 533 ms (summed over the checkers' goroutines)
writing compiled files ( 183 ms )
copy non-compiled files ( 38 ms )
Build time: 3.185s
```

**New finding:** about 1.0 s, a third of the build, goes to loading the Rojo configs and the package paths
(`rojo.FromPath`, one `rojo.Synthetic` per type root, and `createNodeModulesPathMapping`), before any TypeScript
work starts. The profiles in BENCHMARKS-espresso.md didn't show it as its own step. It's the cheapest large win
left: cache it, or make the synthetic resolvers lazy.

Wall clock, medians of 3 (whole-project builds, no incremental state yet):

| scenario | median | runs |
| --- | ---: | --- |
| cold (no `out/`) | 3.44 s | 3.15 / 3.44 / 4.89 |
| warm | 4.01 s | 3.47 / 4.01 / 5.13 |
| leaf (`grenade.controller.ts`) | 3.71 s | 3.56 / 3.71 / 3.78 |
| shared (`shared/net.ts`) | 3.69 s | 3.22 / 3.69 / 3.83 |
