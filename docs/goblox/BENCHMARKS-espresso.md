# goblox vs rbxtsc on espresso (2026-10-30)

A detailed gnome-game build benchmark on a small Linux server, step by step, with profiles, an output diff
and a ranked list of improvements. The desktop numbers are in [BENCHMARKS.md](BENCHMARKS.md).

**Headline (medians of 3 runs, wall clock):**

| scenario | rbxtsc | goblox | speedup |
| --- | ---: | ---: | ---: |
| cold (no `out/`, no tsbuildinfo) | 40.48 s | 6.79 s | **6.0×** |
| warm, no changes | 4.79 s | 7.33 s | **0.65×** (goblox slower) |
| one leaf file edited | 5.18 s | 7.05 s | **0.73×** (goblox slower) |
| one widely imported file edited (`shared/net.ts`, 183 files recompiled by rbxtsc) | 28.70 s | 6.79 s | **4.2×** |
| peak RSS, cold | 884 MB | 1,104 MB | 1.25× more |

goblox always compiles the whole project, so its warm and one-file builds cost as much as a cold one.
rbxtsc's incremental build skips almost all the work for a leaf edit. On this machine that makes rbxtsc
faster for the most common edit. Improvement #1 below fixes that.

## Machine and commits

| | |
| --- | --- |
| host | espresso: Intel Core i5-4210U (Haswell, **2 cores / 4 threads**, 1.7–2.7 GHz, 3 MB L3), 3.7 GiB RAM + 3.7 GiB swap, SATA SSD (SanDisk SD6SB1M1), Linux 6.8.0 |
| toolchain | Node 26.12.1, Go 1.26.10 (user-local `~/.local/go`) |
| gnome-game | `28332a7` plus the user's uncommitted edits to `src/client/gun/gun-fx.ts` and `gun-hud.ts` (copied, not committed). **487** `.ts`/`.tsx` files, 79.8k lines; 594 files in the program with the declaration files |
| rbxtsc | roblox-ts 3.0.0 (npm, plus gnome-game's `patches/roblox-ts@3.0.0.patch`), TypeScript 5.5.3, @typetorch/transformer 0.3.2 |
| goblox | `799d58d5` (`goblox`), typescript-go 7.1.0-dev, typetorch/transformer `go-port` `5c33da7`, built with `-tags typetorch -trimpath -ldflags "-s -w"` (the release task's linux-x64 binary). `origin/goblox` is now at `c6adc6da`, but the 4 newer commits touch only `npm/`, `docs/` and `.github/`, so the measured code is the latest |
| profiling binary | the same commits plus the `-cpuprofile`/`-memprofile` flags in `cmd/goblox` (committed separately), built without `-s -w`. Used only for profiles |

**Method.** Everything ran on copies under `~/bench` (`gnome-game` for rbxtsc, `gnome-game-gb` for goblox, with
`node_modules` symlinked). The user's checkout was never touched. Builds ran one at a time. Before every timed run a
gate waited until no `go`/`compile`/`link` process had run for 2 minutes and the 1-minute load was under 4.
Free memory, swap and load were logged next to each run. Wall time, CPU and peak RSS come from `/usr/bin/time -v`.
Both tools ran on the game's own `tsconfig.json`, which is `bun run build` without `scripts/build-info.ts`. The
prod-only `debugMacros: false` config that `typetorch build` generates was not used. Commands:

```sh
# rbxtsc: a -r preload times program creation, the roblox-ts transform and render by wrapping module exports
node -r ~/bench/scripts/rbxts-phases.js node_modules/roblox-ts/out/CLI/cli.js --verbose
# goblox
goblox build -p . -timing --verbose
```

- **cold**: `rm -rf out include` (that removes `out/tsconfig.tsbuildinfo`)
- **warm**: the same command straight after
- **leaf**: append a comment line to `src/client/controllers/grenade.controller.ts`, a 344-line `@Controller` that
  nothing imports
- **shared**: append a comment line to `src/shared/net.ts`. 63 files import it directly, and rbxtsc recompiles all
  183 files that import it directly or indirectly

**Background load (read before comparing numbers).** espresso was not idle:

- The user's own `typetorch serve --remote` dev server ran `rbxtsc -w` (PID 21135) in the real checkout the whole
  time. It uses about 670–830 MB when resident. It was idle, mostly swapped out, during the timed runs.
- 4 other Claude Code sessions, the t3 server and its bun workers held about 1.7 GB. Swap use stayed at
  2.0–2.4 GB throughout.
- Before the timed runs, a release task compiled goblox in `/tmp/gbx` at the same time as my first `go build`. The
  host thrashed (load average about 145, kswapd at 30 CPU-minutes), and an earlier OOM likely crashed the server.
  No timed run overlapped any Go compile.
- No run tripped the quiet gate. But the **first goblox batch** (cold/warm/leaf/shared/GOMAXPROCS sweep, raw
  labels `goblox-*-[123]`) had 100–400 major page faults per run, with swap use climbing from 2.1 to 2.4 GB. goblox's
  1.3 GB peak was pushing other processes to swap, and it ran **~20% slower** than the two later batches: cold
  8.40 s vs 6.79 s. Those numbers are kept below, marked *noisy*. The headline uses the later batches. rbxtsc
  runs had ~0 major faults, and their background inflation is estimated at under 5%.

## Results per scenario

Median of 3 runs, with the range in brackets.

### rbxtsc

| scenario | wall | CPU (user+sys) | peak RSS | files compiled |
| --- | ---: | ---: | ---: | ---: |
| cold | **40.48 s** [38.49–41.63] | 61.1 s | 884 MB | 487 |
| warm | **4.79 s** [4.40–6.63] | 9.1 s | 401 MB | 0 |
| leaf | **5.18 s** [5.09–6.42] | 9.8 s | 416 MB | 1 |
| shared | **28.70 s** [28.52–29.76] | 46.9 s | 746 MB | 183 |
| `tsc --noEmit` (type check only, no plugin) | 22.53 s [22.30–23.81] | 37.4 s | 762 MB | – |

CPU exceeds wall because V8's GC and compiler threads run alongside the single JS thread.

### goblox

| scenario | wall | CPU | peak RSS |
| --- | ---: | ---: | ---: |
| cold | **6.79 s** [6.62–7.39] | 20.4 s | 1,104 MB |
| warm | **7.33 s** [7.07–8.55] | 20.6 s | 1,110 MB |
| warm `--writeOnlyChanged` | 6.98 s [6.90–7.62] | 20.7 s | 1,290 MB |
| leaf | **7.05 s** [6.99–7.07] | 20.5 s | 1,116 MB |
| shared | **6.79 s** [6.75–7.05] | 20.4 s | 1,112 MB |
| `goblox check` (type check only) | 4.03 s [3.80–4.08] | 12.8 s | 682 MB |
| *noisy batch 1:* cold / warm / woc / leaf / shared | 8.40 / 8.43 / 8.24 / 8.69 / 9.04 s | ~21.5 s | 1.1–1.3 GB |

goblox wrote all 487 files with no diagnostics and no transformer panics in any run. The class `static {}` crash
in `mob-hud.controller.ts` is fixed.

### GOMAXPROCS and `-checkers` (cold)

| config | wall | CPU | peak RSS | batch |
| --- | ---: | ---: | ---: | --- |
| `GOMAXPROCS=1` | 12.75 s [12.54–13.39] | 12.5 s | 1,306 MB | 1 (noisy) |
| `GOMAXPROCS=2` | 8.53 s [7.96–9.13] | 14.9 s | 1,228 MB | 1 (noisy) |
| `GOMAXPROCS=4` (default) | 8.40 s [8.19–9.42] | 21.6 s | 1,262 MB | 1 (noisy) |
| default (4 checkers, GOMAXPROCS=4) | 6.79 s [6.62–7.39] | 20.4 s | 1,104 MB | 3 |
| `-checkers 2` | **6.42 s** [6.38–6.55] | **14.7 s** | **921 MB** | 3 (interleaved with the row above) |
| `-checkers 1` | 8.23 s [8.03–9.58] | 13.0 s | 695 MB | 2 |
| `GOMAXPROCS=2 -checkers 2` | 7.05 s [6.97–7.79] | 12.5 s | 1,105 MB | 2 |
| `GOGC=200` | 6.62 s [6.58–6.66] | 18.8 s | 1,171 MB | 2 |

On 2 physical cores, going from 1 to 2 threads gives 1.5×. Going from 2 to 4 (Hyper-Threading) gives
nothing more, and burns 45% more CPU. typescript-go's 4 checkers each build their own type caches, so the
extra checkers duplicate work and memory. `-checkers 2` is the best setting here: 5% faster than the default,
with 28% less CPU and 17% less memory.

## Step breakdown

### rbxtsc (cold, from `--verbose` plus the preload hook; medians)

| step | cold | leaf | shared | warm |
| --- | ---: | ---: | ---: | ---: |
| node boot + load typescript | 0.30 s | 0.29 s | 0.30 s | 0.30 s |
| **program creation** (parse 1.7 s + bind 1.0 s + builder state) | **3.41 s** | 3.34 s | 3.41 s | 3.66 s |
| copy include + non-compiled files | 0.03 s | 0.03 s | 0.03 s | 0.03 s |
| plugin transformers: `plugins` (@typetorch/transformer) | 4.76 s | 0.22 s | 3.15 s | – |
| plugin transformers: `print` (reprint of all 487 files) | 1.16 s | 0.03 s | 0.71 s | – |
| plugin transformers: `program` (second, proxy program) | 1.21 s | 0.07 s | 0.80 s | – |
| per-file loop: **type check** (`getPreEmitDiagnostics`, derived†) | **12.37 s** | 0.23 s | 9.10 s | – |
| per-file loop: **roblox-ts transform** (`transformSourceFile`) | **14.81 s** | 0.20 s | 9.69 s | – |
| per-file loop: luau render (`renderAST`) | 0.64 s | 0.01 s | 0.40 s | – |
| write compiled files | 0.06 s | 0.00 s | 0.04 s | – |
| other (cleanup, project data, exit) | 0.42 s | 0.42 s | 0.44 s | 0.47 s |
| **total wall** | **40.48 s** | 5.18 s | 28.70 s | 4.79 s |

† TypeScript 5.5's exports are non-configurable getters, so `getPreEmitDiagnostics` can't be wrapped. The value is the
sum of the per-file `compile` lines minus the measured transform and render.

### goblox (cold, from CPU profiles; goblox has no step timings of its own)

Samples are bucketed by the most specific frame on their stack (`~/bench/scripts/pprof-phases.py`). At
`GOMAXPROCS=1`, CPU time ≈ wall time (12.4 s of samples vs 13.1 s wall), so that column reads as wall time.

| step | GOMAXPROCS=1 (≈ wall) | GOMAXPROCS=4 (CPU) | rbxtsc counterpart (wall) |
| --- | ---: | ---: | ---: |
| parse/scan | 0.54 s | 1.23 s | ⎫ 3.41 s |
| bind | 0.30 s | 0.68 s | ⎬ |
| module resolution / config | 0.08 s | 0.19 s | ⎭ |
| plugin: typetorch transform | 0.83 s (0.65 s of it checker) | 0.80 s | 4.76 s |
| plugin: reprint + reparse + second program | 0.41 s | 0.40 s | 2.37 s |
| **type check** (checker, outside plugin/transform) | **3.43 s** | 6.10 s | 12.37 s |
| **roblox-ts transform** | **4.91 s** (checker 1.64, own code 0.94, alloc/maps 1.77) | 7.36 s | 14.81 s |
| luau render | 0.30 s | 0.53 s | 0.64 s |
| write / copy / fs | 0.07 s | 0.08 s | 0.09 s |
| GC (background mark) | 1.44 s | 3.17 s | (V8, mostly off-thread) |
| other | 0.07 s | 0.13 s | 0.72 s |
| **total** | **12.38 s** | **20.67 s** (6.9 s wall) | 40.48 s |

The rbxtsc column is wall time with one JS thread. Single-threaded, goblox is 3.3× faster overall (12.4 s vs
40.5 s): diagnostics 3.6× (3.43 s vs 12.37 s), all checker work 5.3× (5.7 s vs ~30 s, counting the checker
queries made by the transform and the plugin), the roblox-ts transform 3.0×, and the plugin pipeline 5.8×. Parallelism on 2 cores
adds the rest of the cold speedup, 1.8× (12.4 s → 6.8 s).

## Profiles

Raw profiles are in `~/bench/results/prof/` on espresso: `rbxtsc-cold.cpuprofile`, `goblox-cpu-p{1,4}.pprof`,
`goblox-mem.pprof`.

### rbxtsc (`node --cpu-prof`, cold, 47.6 s of samples)

Stack-based split: roblox-ts transform 17.5 s (37%), type check 15.1 s (32%), @typetorch/transformer 5.0 s
(11%), program creation 3.4 s (7%), GC 1.8 s (4%), plugin reprint and second program 2.2 s (5%), render 0.8 s.

- **64% of all JS time is the TypeScript checker.** That is 15.1 s in `getPreEmitDiagnostics`, plus **10.9 s of checker work
  called from the roblox-ts transform** (type queries), plus **4.5 s called from the plugin**. roblox-ts's own
  transform code is only 6.6 s, and the plugin's own code is 0.5 s.
- The plugin's checker work runs on the *original* program. The per-file loop then type-checks the *proxy* program
  with a fresh checker, so the 4.5 s of plugin type work is done twice.
- Top self time: `(garbage collector)` 1.78 s, then a long flat tail of checker functions (`checkIdentifier`,
  `resolveNameHelper`, `resolveEntityName`, `checkExpressionWorker`, `getFlowTypeOfReference`,
  `recursiveTypeRelatedTo`, 0.4–0.65 s each). luau-ast `create` takes 0.55 s.
- `node:path` `normalizeString` + `relative` take 0.75 s, called almost entirely from `RojoResolver.isPathDescendantOf`
  (0.57 s) and roblox-ts's `isPathDescendantOf` (0.19 s).
- Startup: `node -e 0` takes 0.04 s, `require("typescript")` 0.31 s, roblox-ts + plugin modules 0.57 s. That is
  under 2% of a cold build, but about 12% of a warm one.

### goblox CPU (`go tool pprof -top`, GOMAXPROCS=4, 20.7 s of samples over 6.9 s)

- The top flat entries are all runtime: `tryDeferToSpanScan`, `scanObject`, `findObject`, map probing
  (`matchH2`), `memclr`, `memmove`, `mallocgc` (14% cumulative). Background GC is 15% of CPU at GOMAXPROCS=4 and
  12% at 1.
- **`nodepath.Relative` is 12% of all CPU** (1.48 s of 12.4 s at GOMAXPROCS=1, 2.23 s at 4). It is called from
  `rojo.isPathDescendantOf` via `(*RojoResolver).GetRbxPathFromFilePath` (1.0 s) and from `transform.isPathDescendantOf`
  (0.4 s). This is the Go port of the same rbxtsc hotspot, but here it weighs 3× more, because everything else got
  faster.
- Checker hot spots (`getTypeAtFlowNode` 1.48 s cumulative, `NameResolver.Resolve` 0.6 s) look as expected.

### goblox heap (`-memprofile`, `alloc_space`, 1.45 GB allocated in one cold build)

- **`nodepath.Relative` allocates 465 MB, 31% of everything**: `normalizeString` 0.24 GB, `strings.Split` 0.13 GB,
  `splitNonEmpty` 0.15 GB, `Resolve` 0.27 GB cumulative. 72% of that comes from `rojo.isPathDescendantOf`
  and 23% from `transform.isPathDescendantOf`.
- `transformFiles` (the roblox-ts transform, including its checker queries) is 52% of allocation. The checker's
  `checkSourceFile` is 19%, `parser.ParseSourceFile` 9.4%, and `RunPlugins` 7.4% (typetorch `captureDiagnostic`
  is 7.1% of it).
- Peak RSS is 1.10–1.30 GB with 4 checkers, 0.92 GB with 2, and 0.70 GB with 1. rbxtsc peaks at 0.88 GB.

## Output parity

The reference here is gnome-game's own `node_modules/roblox-ts`: the npm 3.0.0 release. goblox ports roblox-ts
**master** (68648b0). BENCHMARKS.md ("gnome-game output diff against roblox-ts") explains why that reference
differs and measured 406/407 identical against a master oracle on the desktop. That oracle isn't built on
espresso (only the source is in `~/projects/ref/roblox-ts`), so this section measures against what gnome-game
actually ships with.

| comparison | identical | differ |
| --- | ---: | ---: |
| byte for byte, all `.luau` files | **82 / 487** | 405 |
| code only (Luau comments and blank lines removed) | **213 / 487** | 274 |
| only in rbxtsc | `tsconfig.tsbuildinfo` (goblox writes none, by design) | |

Every difference sampled matches the known differences between master and 3.0.0. None looks like a goblox
bug. Grouped by root cause, with the transformer area:

1. **JSDoc comment rendering** (comments, about 350 of 405 files differ in comments). 3.0.0 copies `/** … */` as
   `--[[ * … * … ]]` and `--* text `. Master strips the `*` decoration (`--[[ text ]]`, `-- text`). This is the
   only difference in 131 files.
2. **Temporaries dropped for side-effect-free expressions** (expressions/identifiers, 169 files, 747 lines).
   `local _typeId = typeId; MobTypeById[_typeId]` becomes `MobTypeById[typeId]`. This is master's
   evaluation-effects analysis.
3. **Destructuring without `_binding`** (destructuring, 41 files). `local _binding = throwData; local origin =
   _binding[1]` becomes `local origin = throwData[1]`.
4. **Optional chaining / nil-check temporaries** (optional chain, 69 files). `local _exp_2 = origin; local
   _result_2 = def; … Fx.shot(_exp_2, …)` keeps `origin` inline.
5. **Template and string captures** (template literals, 36 files). `` `[VFX] {_exp} v…` `` becomes `` `[VFX] {name} v…` ``,
   and `_ctx:reply` becomes `ctx:reply`.
6. **Object literals built inline** (object literals, 16 files, 206 lines). In `shared/gun/gun-config.luau`, 3.0.0
   builds `_object_1` field by field, because `CFrame.new(…) * CFrame.Angles(…)` produces a prerequisite. Master
   emits one `table.freeze({ … holdC0 = CFrame.new(…) * CFrame.Angles(…), … })`.
7. **Method calls with prerequisites** (calls, 10 files). `local _self = self; local _exp = player; …
   _self:track(_exp, …)` becomes `local _fn = self.track; … _fn(self, player, …)`.
8. **JSX children pushed inline** (JSX, 12 files). `local _arg0 = React.createElement(…); table.insert(pieces,
   _arg0)` becomes `table.insert(pieces, React.createElement(…))`.
9. **Array slice/spread macros** (array macros, 4 files). `table.move(_lines, _arg0 + 1, …)` and its temporaries
   become inline arguments.
10. **Debug macro output** (`$print`/`$warn` from @typetorch/transformer, 24 files). The macro output is the
    same, apart from captured temporaries as in #5 (`{_exp}` becomes `{name}`).

To get a meaningful parity number on espresso, build the master oracle (`~/projects/ref/roblox-ts` at 68648b0)
and repeat BENCHMARKS.md's oracle command. That needs `npm install` plus a tsc build, roughly 1 GB more RAM.

## Improvements, ranked

Savings are for gnome-game on espresso unless noted. "goblox" items need changes in `internal/roblox/**` or
the transformer, which this benchmark did not touch.

| # | improvement | where | estimated saving | effort |
| ---: | --- | --- | --- | --- |
| 1 | **Incremental goblox builds / a warm watch daemon.** goblox's `-w` rebuilds the whole project on every change (7 s here). Keep the `compiler.Program` and per-file results between rebuilds. Re-parse only changed files, re-check and re-transform only the changed files and their importers (rbxtsc's `getChangedFilePaths` set), and reuse cached outputs and diagnostics for the rest | goblox `watch` / `project.Build` | leaf edit 7.05 s → ~1 s (rbxtsc: 5.2 s), shared edit 6.8 s → ~3 s. It also removes 0.85 s of parse/bind per rebuild | L |
| 2 | **Cache diagnostics and outputs on disk (a tsbuildinfo equivalent)** for one-shot `goblox build` (CI, `typetorch build`). Key by file text hash + config + plugin hash, and skip type check and transform for files whose hash and dependency hashes are unchanged | goblox | warm 7.3 s → ~1.5 s (load + hash + write). Leaf like #1 | M–L |
| 3 | **Make `isPathDescendantOf` a prefix check and memoize `GetRbxPathFromFilePath` per file.** Today each call allocates through `nodepath.Relative` (`Resolve` + `Split`). Normalize paths once, compare with `strings.HasPrefix` plus a separator check, and cache the rbx path per source and import path | goblox `internal/roblox/rojo`, `transform` | ~1.5 s CPU (12%) and 465 MB (31%) of allocation, so ~0.7–1.0 s wall at default settings plus less GC. The same fix in `@roblox-ts/rojo-resolver` saves rbxtsc ~0.75 s | S |
| 4 | **Choose the checker count from physical cores and memory** (or default `-checkers 2` when ≤ 4 threads or < 8 GB). 4 checkers on 2 cores duplicate type caches | goblox `cmd/goblox` / `project` | 5% wall (6.79 → 6.42 s), 28% CPU, 180 MB RSS. On espresso, the RSS saving also avoids the swap churn that cost batch 1 ~20% | XS |
| 5 | **Don't type-check twice around plugins.** rbxtsc's plugin queries the original program's checker (4.5 s), then type-checks the proxy program again from scratch. Reuse the original checker's diagnostics for files the plugin left untouched, and check only transformed files on the proxy | rbxtsc patch / goblox `RunPlugins` | rbxtsc ~4–5 s cold. goblox ~0.6 s CPU (0.65 s plugin checker + part of the 0.4 s reparse) | M |
| 6 | **Return untouched source files unchanged from @typetorch/transformer.** rbxtsc reports `487/487 files re-printed`, so the plugin returns a new node for every file. Files it doesn't change could skip the reprint and reparse, and keep their bound/checked state | @typetorch/transformer (JS and Go) | rbxtsc up to ~2.4 s cold (print 1.16 + program 1.21 s), proportionally on edits. goblox up to 0.4 s | S |
| 7 | **Cut allocation in the roblox-ts transform.** GC is 12–15% of goblox CPU, and the transform's own map and allocation work (1.77 s) is twice its logic (0.94 s). Preallocate slices, intern strings, reuse visitor buffers. `GOGC=200` buys only 2.5% wall for 6% more RSS, so fix the allocations rather than tune GC | goblox `internal/roblox/transform` | ~0.5–1 s CPU (~0.3–0.5 s wall) | M |
| 8 | **Type-check cost: skip or defer it?** It can't be dropped: emit is blocked on errors, and the transform needs types (its checker queries alone are 1.6 s). For the dev loop, run full diagnostics in the background after emitting, and only block when errors appear. Diagnostics alone are 3.4 s of goblox's 12.4 s single-threaded | goblox watch | ~1.5–2 s wall per watch rebuild before #1/#2. Less after | M |
| 9 | **Step timings in `goblox --verbose`**, like roblox-ts's `benchmarkIfVerbose` (program, plugins/print/program, per-file compile, write). This benchmark had to derive goblox's steps from profiles | goblox `project` | – (makes regressions visible) | XS |
| 10 | **Parallel transform across files:** goblox already does it (one goroutine per checker). Going from 1 to 2 threads gives 1.5×, and 2 to 4 HT threads gives 1.0×, so on small machines it's memory-bound, not core-bound. For rbxtsc, in-process parallelism isn't possible (single JS thread), and worker threads would need one program each (~0.9 GB each). Adopting goblox is the parallel option | – | – | – |
| 11 | **`--writeOnlyChanged`**: writing is 0.06–0.07 s in both tools, so it doesn't change build time (6.98 s vs 7.33 s warm, within noise). Its value is fewer file events for Rojo, which is why it's on by default in watch mode | – | ~0 s build time | – |
| 12 | **Startup:** goblox starts in < 10 ms. rbxtsc spends ~0.6 s on boot and module load (12% of a warm build). A daemon (#1) removes it. Node's `NODE_COMPILE_CACHE` would cut module compile time, but only by a few hundred ms | rbxtsc | ≤ 0.6 s per rbxtsc build | XS |
| 13 | **Memory on small hosts.** goblox peaks at 1.1–1.3 GB, which is 25–45% above rbxtsc. With the user's `rbxtsc -w` (~0.7–0.8 GB) and other processes, espresso swaps, and an OOM probably caused the crash that interrupted the first attempt at this benchmark. #3 and #4 together should bring goblox to about rbxtsc's 0.9 GB. Setting `GOMEMLIMIT` (for example 1 GiB) in the npm launcher on small machines adds a soft cap | goblox | avoids ~20% swap slowdowns and OOM risk | XS–S |

The **top 5 by expected payoff for the user's edit loop** are #1, #2, #3, #4 and #5/#6.

## Reproducing

Scripts are in `~/bench/scripts` on espresso; raw logs, `summary.txt`, `medians.txt` and the parity reports are in
`~/bench/results`.

- `run.sh <label> <cmd…>`: quiet gate, memory/load log, `/usr/bin/time -v`
- `quiet.sh` + `gowatch.sh`: wait until no Go compile has run for 2 minutes and the load is under 4
- `bench-rbxtsc.sh`, `bench-goblox.sh`: the scenarios above
- `rbxts-phases.js`: `node -r` preload that times rbxtsc phases without editing `node_modules`
- `cpuprofile-summary.py`, `pprof-phases.py`: profile summaries and step buckets
- `parity.py`, `parity-code.py`: byte-for-byte and code-only `out/` diffs
