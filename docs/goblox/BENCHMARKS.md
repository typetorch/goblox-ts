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

## Transform/luau allocation (improvement #7/#13, 2026-10-30)

Desktop (6 cores, 32 GB, Windows 11), scratch copy of gnome-game at `b6e33b3` (406 source files, 407 `.luau`
outputs, `src/shared/build.ts` generated with `bun scripts/build-info.ts --dirty false`). goblox built with
`-tags typetorch -trimpath` (no `-s -w`). Cold builds (`out/` and `include/` deleted), `goblox build -p .`,
median of 3. CPU is the process's user+sys time, RSS the peak working set (polled every 20 ms). Alloc is
`alloc_space` from a separate `-memprofile` run (sampled, so ±~5 MB). Other agents were compiling on the same
machine for part of the session, so only interleaved runs are compared.

| build | wall | CPU | peak RSS | total alloc |
| --- | ---: | ---: | ---: | ---: |
| `5b8df10e` (before) | 2.99 s | 7.78 s | 924 MB | 1,505 MB |
| `283f997f` (facts in place + effects fast path + render lines) | 2.88 s | 7.48 s | 929 MB | 1,469 MB |

A second interleaved pair ran while the machine was busy (base 3.41 s / 8.59 s, new 4.14 s / 8.67 s), so the
wall-time difference is within noise. The allocation drop is real. Per commit (alloc_space, cumulative):

| commit | what | measured |
| --- | --- | --- |
| `4054e5ef` | evaluation facts are a `*annotated` updated in place (clones copy it through `luau.FactsCloner`) instead of re-boxing the struct into `any` on every mark | `updateAnnotated` 31 MB → 12 MB |
| `41276bf0` | `getEffectsOfListOrNode` merges identifier reads and assignment/declaration target writes straight into its accumulator (no intermediate `EvaluationEffects` and maps) | 36.5 MB → 24.5 MB |
| `283f997f` | `RenderState.Line` concatenates once; indents are slices of a constant | `RenderAST` 59.3 MB → 55.7 MB |

All three keep gnome-game byte-identical (only the `builtAt` timestamp in `shared/build.luau` differs), and keep
the fixtures harness at 73/73.

**Where the allocation actually is (before).** 1,505 MB per cold build: `nodepath.Relative` 554 MB (37%,
reached from `rojo.isPathDescendantOf`, `transform.isPathDescendantOf` via `isRobloxType`/`isPossiblyType`/
`fixVoidArgumentsForRobloxFunctions`/`isStableBuiltinMember`, and `getImportParts`). That is the quick-wins item #3,
not this one. The checker's `checkSourceFile` is ~18%, and checker queries from the transform are ~10–20%. The
transform's and the renderer's own allocations are only ~210 MB (14%). The largest of those that remain:
`lsutil.GetLastToken` → `ast.NewNodeVisitor` 17.5 MB (the visitor is allocated per call outside `internal/roblox`),
the renderer's string-returning functions (~55 MB; a single-buffer renderer would need every `render*` rewritten),
and the effects accumulator's map growth (~18 MB).

**GC and peak memory.** `GODEBUG=gctrace=1` shows only 15 GCs per cold build. The live heap reaches ~500 MB and the
heap peaks at ~860 MB just before the end (transform + 4 checkers' caches), so peak RSS is about 2× the live heap.
The Luau AST and `TransformState` are already per-file locals in `project.transformFiles`, dropped after
`RenderAST`. Only the rendered string is kept until write. Soft memory limits (env `GOMEMLIMIT`, same binary):

| `GOMEMLIMIT` | wall | CPU | peak RSS |
| --- | ---: | ---: | ---: |
| unset | 2.94 s | 7.72 s | 947 MB |
| 896 MiB | 3.21 s | 8.38 s | 819 MB |
| 768 MiB | 3.54 s | 9.45 s | 743 MB |
| 512 MiB | 5.31 s | 30.6 s | 601 MB |

(The 896 MiB row was measured on `283f997f`, during the busy period.) A limit trades ~10–20% wall per 15–20%
less RSS, and thrashes once it gets near the ~500 MB live heap. On a 32 GB machine that's a pure loss. On
espresso it only pays off if the build would otherwise swap. So no default was added to `cmd/goblox`. Users on
small machines can set `GOMEMLIMIT` themselves. A RAM-scaled default (for example total RAM / 4) belongs in the
npm launcher, with a measurement on espresso. `-checkers 2` (#4) cuts more RSS for less cost.

**Also seen (not in this item's scope).** On Windows, `rojo.(*RojoResolver).parseTree` spends 0.81 s on the main
goroutine in `Realpath`/`Stat` syscalls before the transform starts (~28% of the 2.9 s wall). It is worth a
look on Windows.

### Milestone 2: cached Rojo setup + watch source-file reuse (2026-10-30, `54284a92`)

`load rojo configs and package paths` went from 994 ms to 480 ms (a cachedvfs while the resolvers are built).
Watch rebuilds reuse unchanged parsed/bound files (create program 138 ms to ~47 ms). Medians of 3, same desktop and
copy, quieter machine. The binary includes everything on main at the time, including the allocation agent's
transform/luau commits, so these numbers aren't this branch's work alone:

| scenario | median | runs |
| --- | ---: | --- |
| cold | 2.33 s | 2.32 / 2.33 / 2.35 |
| warm | 2.37 s | 2.36 / 2.37 / 2.39 |
| leaf | 2.38 s | 2.37 / 2.38 / 2.40 |
| shared | 2.37 s | 2.37 / 2.37 / 2.38 |

Every build still compiled the whole project at this point. Items #1 (warm watch) and #2 (on-disk cache) came next:
see "Incremental builds" below.

## Incremental builds (2026-10-30, `d1e6e4cbe`)

Same desktop (12 threads, 32 GB, Windows), on a copy of gnome-game (407 compiled files; `shared/build.ts` is fixed, so
it doesn't count as changed), with nothing else running. Times in ms from process start to exit, median of 3.
goblox-base is `910f3438c`, before this work. rbxtsc is the project's `node_modules/.bin/rbxtsc` with `incremental`.

- **cold:** `out/` deleted. **warm:** no change. **comment:** a comment appended to `shared/net.ts`.
- **leaf:** an export appended to `client/ui/app.tsx` (3 files compiled).
- **shared:** an export appended to `shared/net.ts` (160 files compiled).

| one-shot `goblox build` | cold | warm | comment | leaf | shared |
| --- | ---: | ---: | ---: | ---: | ---: |
| rbxtsc | 21957 | 6372 | 17701 | 7365 | 18725 |
| goblox-base | 3723 | 3991 | 2823 | 3294 | 2826 |
| goblox | 2477 | 323 | 506 | 797 | 2758 |
| goblox, second pass | 2889 | 357 | 651 | 648 | 1976 |

Runs (first pass): cold 3892 / 2436 / 2477, warm 323 / 334 / 302, comment 485 / 506 / 566, leaf 797 / 795 / 862, shared
3435 / 2758 / 2191. The first run of a series is often slower, so the spread on cold and shared is large.

`goblox -w`, from saving the file to the "Found N errors" line. This includes up to 250 ms of polling and the 100 ms
debounce:

| `goblox -w` | runs (ms) |
| --- | --- |
| first build, cache present | 352 |
| leaf | 774 / 507 / 553 |
| comment in `shared/net.ts` | 595 / 566 / 563 |
| shared | 2053 / 1921 / 1895 |
| adding a type error (160 files) | 2489 |
| another edit, error still reported | 2288 |
| fixing it | 1166 |

The user's laptop was not measured. These numbers only say that goblox beats rbxtsc on this desktop.

Correctness on the same copy (`gnome_check.sh`, 61 steps: 20 scripted edits plus 30 sampled ones and restores): exit code,
diagnostics and output of each incremental build matched a `--noIncremental` build, apart from `builtAt` in
`shared/build.luau`. After the watch runs above, the 407 outputs matched a clean build too.

### Again on main (2026-11-01, `5f08c9a77`)

The same scenarios after the plugin-per-checker, compile order, declaration emit and transform commits on main.
Same desktop, a fresh copy of gnome-game (`b6e33b3`, 407 compiled files), goblox built with `-tags typetorch`
against typetorch-transformer `go-port` (`d3e0c4c`). Medians of 3, ms, process start to exit:

| one-shot `goblox build` | cold | warm | comment | leaf | shared |
| --- | ---: | ---: | ---: | ---: | ---: |
| rbxtsc | 18653 | 5726 | 14851 | 6353 | 14741 |
| goblox | 2114 | 426 | 615 | 603 | 1585 |
| goblox, second pass | 2120 | 426 | 584 | 610 | 1602 |

Runs: rbxtsc cold 19030 / 18653 / 17789, warm 5753 / 5726 / 5706, comment 14851 / 15311 / 14686, leaf 6282 / 6353 /
6470, shared 15472 / 14343 / 14741. goblox cold 2233 / 2114 / 2072, warm 426 / 409 / 454, comment 595 / 615 / 757,
leaf 601 / 742 / 603, shared 1675 / 1585 / 1559.

A warm build is ~100 ms slower than on Oct 30, spread over steps that existed then (parse tsconfig 57 → 70-88 ms,
create program, copy non-compiled files 35 ms), so it's likely the machine. Shared got faster (2758 → 1585) with the
transform and plugin work on main. A trailing comment in `shared/net.ts` now compiles 0 files: the plugin prints the
same text, so the program of the plugins' output doesn't change.

| `goblox -w` (save to "Found N errors", incl. polling and debounce) | runs (ms) |
| --- | --- |
| first build, cache present | 413 |
| leaf | 813 / 584 / 572 |
| comment in `shared/net.ts` | 566 / 566 / 571 |
| shared | 1463 / 1466 / 1496 |
| adding a type error (160 files) | 1387 |
| another edit, error still reported | 1457 |
| fixing it | 595 |

Correctness: `gnome_check.sh` (seed 7, 61 steps) matched a `--noIncremental` build on every step, and the watch
outputs matched a clean build (407 of 407 files). `go test ./internal/roblox/... ./pkg/goblox/...` and the oracle
harness (73/73) pass on the same commit.

## Incremental design (implemented, see CLI.md "Incremental builds")

The notes this work followed, for items #1 and #2 of BENCHMARKS-espresso.md. `internal/roblox/project/incremental.go` has
the code. Where it differs from these notes: a build whose diagnostic lands in a file the plugins didn't print is
compiled again in full and saves no state (see CLI.md), and the JSON files' signature is their version.

- **Affected files.** Use `internal/execute/incremental` and don't write a new graph. `incremental.NewProgram(program,
  old, ...)` hashes every file's text, diffs references, and `collectAllAffectedFiles` removes the cached semantic
  diagnostics of every affected file. "No cached semantic diagnostics" is then exactly the set to re-check *and*
  re-transform. Affected means: the changed file, plus all transitive importers when its d.ts signature changed
  (re-exports make the transitive walk necessary), plus everything when a global-scope file changed. A comment edit
  in `shared/net.ts` re-checks 1 file, where rbxtsc's `getChangedFilePaths` re-checks 183.
- **Missing exports to add** (in a new `internal/execute/incremental/roblox_exports.go`): a batch "collect semantic
  diagnostics for these files" (today `GetSemanticDiagnostics(ctx, file)` checks one file at a time, so the parallel
  check is lost), the list of files without cached diagnostics, and `snapshotToBuildInfo` / `buildInfoToSnapshot`
  wrappers for goblox's own cache file.
- **Plugins need two incremental programs.** @typetorch/transformer is type-driven (`buildGuardFromType`, user
  macros), so its output for a file depends on the types it imports. Keep one incremental state on the *original*
  program (its affected set = the files to re-run the plugin on; store "plugin done" markers as empty semantic
  diagnostics), cache each file's printed plugin output, rebuild the proxy program from those texts (reusing parsed
  proxy files in watch mode), and keep a second incremental state on the *proxy* program, whose affected set is what
  gets type-checked and transformed.
- **Per-file cache policy.** Reuse a file's Luau only if its transform reported no diagnostics at all and no
  panic. Files with warnings or errors run again on every build, so their diagnostics come out identical. If a
  transform reports a diagnostic for another file, or one without a file, don't cache any transform result from that
  build. One-shot builds must also check that the `out/` file still hashes to the cached output, and re-transform it
  when it doesn't. A build with errors writes nothing (as today), so its new results must not be marked as written.
- **Cache key (invalidate everything on change):** goblox version + VCS revision + executable size/mtime, the parsed
  compiler options, the tsconfig/extends texts, the resolved project options (including CLI overrides), the
  package.json, every lockfile (bun.lock/bun.lockb/package-lock.json/yarn.lock/pnpm-lock.yaml), the Rojo project
  text, the tsconfig `plugins` entries and `TYPETORCH_DEBUG_MACROS`. The @rbxts `.d.ts` files are program files, so
  the incremental state already tracks them. But `createNodeModulesPathMapping` and the synthetic Rojo resolvers read
  node_modules directly, so the lockfile has to be in the key.
- **Cache location:** beside `tsBuildInfoFile` (`out/tsconfig.goblox.tsbuildinfo`), registered as an owned output
  so `syncProjectOutputs` keeps it. Only when `incremental`/`composite` is on, as with tsc.
- **Volatile macros:** `$git()`, `$compileTime()` and UUID user macros keep their old value in unaffected files, as
  in rbxtsc. gnome-game rewrites `shared/build.ts` before every build, so that file always counts as changed.
- **Checkers can't be kept across programs.** A change makes a new Program and a new checker pool, as in `tsc -w`.
  Skipping unaffected files is where the savings come from.

## Quick wins: path checks and the checker default (2026-10-30)

Items #3 and #4 of [BENCHMARKS-espresso.md](BENCHMARKS-espresso.md), on gnome-game: cold `goblox build`, Ryzen 5 5600
(6 cores / 12 threads), Windows, 4 checkers. Before is `5b8df10e`; after is the same commit plus the path changes.
Median of 5 interleaved runs, each started when total CPU was under 15% (other builds were running on the machine):

| | before | after | change |
| --- | ---: | ---: | ---: |
| wall | 3.25 s | 2.84 s | −12% |
| CPU | 8.47 s | 6.69 s | −21% |
| peak working set | 953 MB | 705 MB | −26% |
| allocated (`-memprofile`, one run each) | 1,553 MB | 922 MB | −41% |
| allocated in `nodepath.Relative` (same profiles) | 582 MB | 17 MB | −97% |

`nodepath.IsPathDescendantOf` allocates nothing, against 28 allocations and 1,684 B per call before, and takes 137 ns
against 1,283 ns. Output is byte-identical apart from the `builtAt` timestamp in `shared/build.luau`.

**Checker count on this machine** (median of 3, same conditions, after the path changes):

| `-checkers` | wall | CPU | peak working set |
| ---: | ---: | ---: | ---: |
| 2 | 3.08 s | 5.61 s | 695 MB |
| 3 | 2.81 s | 6.11 s | 658 MB |
| **4** | **2.75 s** | 6.50 s | 712 MB |
| 6 | 2.89 s | 8.20 s | 929 MB |
| 8 | 2.74 s | 8.34 s | 979 MB |

Above 4 checkers, wall time stops improving while CPU and memory rise, so hosts with 8 or more threads keep 4. Below
that the default is min(4, physical cores), based on the espresso numbers. That lower branch was not re-measured
here: a `GOMAXPROCS=4` comparison on this desktop was drowned out by other load.

## Plugin step (2026-10-30)

gnome-game (407 files) with the Go @typetorch/transformer, cold builds on the 12-thread desktop, alternating the two
binaries of each comparison. `--verbose` step timings, median of the rounds. Each row adds one change to the row above.

| transformer | running transformers | type check | transform and render | wall |
| --- | ---: | ---: | ---: | ---: |
| keeps untouched files (goblox and go-port c8c6e54) | 624 ms | 418 ms | 390 ms | 2.19 s |
| + skips callees that can't be macros | 431 ms | 539 ms | 448 ms | 2.31 s |
| + one goroutine per checker | 192 ms | 468 ms | 420 ms | 1.79 s |

The rows come from two 3-round comparisons against the last row, so wall and the steps other than the plugin step are
only comparable within a comparison: the second row's run was noisier (minimum wall 2.00 s). The first and last rows
were measured together. Before untouched files were kept, the plugin step took 715 ms (5 rounds, same conditions except
that binary also lacks the other agents' changes to goblox).

The first cold build after the plugin stops warming the checkers pays a little of it back: type check +49 ms and
transform +24 ms in the direct comparison, against −434 ms in the plugin step.

Output: identical to rbxtsc with the JS transformer apart from `builtAt` and the build info files, at 1, 2, 4 and 8
checkers and across repeated builds. Clean under the race detector (Linux build in WSL: gnome-game, and the transformer
tests twice).

**Tried and dropped:**

- *Not allocating in `captureDiagnostic`.* Its closures barely allocate: about 0.3 MB of 1,388 MB in a whole build. Its
  large cumulative numbers in profiles are the recursive transform under it. Skipping it for nodes with nothing to catch
  saved 0.4% of the plugin step's allocations, and no time.
- *Reusing the original program's diagnostics for untouched files.* The default build doesn't check twice: the original
  program only answers the plugin's queries. Reuse would move checking to the original program rather than remove it.
  Only 158 of the 407 files (22% of the bytes) have an import closure free of changed files, and that ignores global
  declarations. In the best case the check took 1.44 s against 1.59 s (median of 7, under load, within the noise),
  without counting the warm checkers transform would lose.

## Transform CPU (2026-10-30)

Cold gnome-game builds on the 12-thread desktop, interleaved. "Without" is the same `main` with these commits reverted,
so the other agents' work is in both columns. `-checkers 4` is the median of 7, `-checkers 2` the median of 3.

| | transform and render | render sum | type check | total |
| --- | --- | --- | --- | --- |
| 4 checkers, without | 408 ms (min 363) | 134 ms | 443 ms | 2.30 s |
| 4 checkers, with | 368 ms (min 342) | 135 ms | 417 ms | 2.20 s |
| 2 checkers, without | 714 ms (min 711) | 119 ms | 518 ms | 2.61 s |
| 2 checkers, with | 629 ms (min 626) | 124 ms | 523 ms | 2.49 s |

CPU in `transformFiles`, from 4 merged profiles at 4 checkers: 2.40 s per build before these commits and the quick wins,
1.40 s after both (nodepath 0.86 s to 0.06 s, mostly the quick wins; checker 0.77 s to 0.64 s).

What helped, in the order they landed. The machine was shared with other builds, so read the minimums (5 interleaved
rounds, each row includes the ones above it):

| commit | transform min | transform median |
| --- | --- | --- |
| `isRobloxType` cached by symbol, `hasCallSignatures` by type | 639 ms | 930 ms |
| + no checker call for types known from the syntax | 531 ms | 604 ms |
| + `ensureTransformOrder` skips the effects of the last expressions | 491 ms | 665 ms |
| + closing brace of a block without `lsutil.GetLastToken` | 488 ms | 703 ms |

The last row is about allocation, 23 MB per build at the time, rather than time. The medians are noise: use the table at
the top of this section for the total effect.

Output: identical to the commit before apart from `builtAt` and the build info file, harness 73/73. Race detector:
gnome-game at 4 checkers and the `transform` and `project` tests are clean. On Windows this took a linked import library
for `WaitOnAddress` and clearing `DYNAMIC_BASE` and `HIGH_ENTROPY_VA` in the binary's header.

The `GetLastToken` commit broke comments in constructors with a statement after `super()`: it took the position of the
closing brace from the last statement it was given, not the last one in the block. gnome-game and the fixtures have no
such comment, so identity checks passed. Fixed in the next commit, with a test that compares every split of the statement
lists against the search.

**Tried and dropped:**

- *Skipping `validateNotAnyType` for syntactic primitives, and memoizing `guessVirtualPath`.* Transform median 462 ms
  against 391 ms, minimum 377 ms against 370 ms (5 rounds), CPU within noise. They saved about 9 MB of allocation per
  build and no measurable time.

**Measured and not tried:**

- *Single-buffer renderer.* Render is about 0.135 s of the 1.40 s of CPU and 55 MB of 234 MB of allocation. `SolveTempIds`
  is 0.04 s of that and wouldn't change. Rewriting about 25 render functions could save about 0.03 s of CPU at best.
- *Macro symbol fields instead of the string map.* `GetSymbolOrThrow` is about 2.5 ms of CPU per build.
- *Lazy binding sets in `getEffectsOfListOrNode`.* About 4 MB and 10 ms of CPU per build.
- *Memoizing `GetSymbolAtLocation`.* 335k calls for 161k identifiers at about 420 ns each; the map would take back about
  half of the 70 ms of CPU it could save. This estimate comes from counts, not a run.

**What's left is the checker.** It is 0.64 s of the 1.40 s, nearly all `GetTypeAtLocation`, which checks nested expressions
again each time, so its cost is quadratic in chain depth. That can't be cached from the transform: the fix belongs in
`checker.go`. Checker groups also finish unevenly (0.93 to 1.13 s of wall with 85 to 115 files each, measured with
instrumentation), which is for the driver to balance.
