# Conformance with roblox-ts

How goblox compares to rbxtsc built from roblox-ts at 68648b0, and how to check it again. `<oracle>` is that checkout,
built (see [PORTING.md](PORTING.md)). All commands run from the repository root.

## Results

| Check | Command | Result |
| --- | --- | --- |
| Transform fixtures, Luau byte for byte | `go run ./cmd/goblox-fixtures -oracle <oracle>` | 73 of 73 |
| Runtime specs under Lune | `... -lune -lune-bin <lune> -rojo-bin <rojo>` | goblox 874 of 874, rbxtsc 874 of 874 |
| The same, with `--optimizedLoops` | `... -lune -optimizedLoops ...` | goblox 874 of 874, rbxtsc 874 of 874 |
| Diagnostic fixtures: code, message, span, printed text | `... -diagnostic-fixtures` | 169 of 169 |
| CLI behaviour: exit code, stdout, stderr, files written | `... -cli` | 64 scenarios: 57 byte for byte, 7 intentional |

The tools are lune 0.8.8 and rojo 7.4.4. `-compiler rbxtsc` runs the specs with rbxtsc for the baseline. Add `-v` for
diffs and `-run <regexp>` to pick cases.

## What each check does

- `-lune` is `npm test` without jest: it compiles a temporary copy of `<oracle>/tests`, runs `rojo build`, then
  `lune run runTestsWithLune.luau`. The tests project references `projects/shared` and `projects/common`, which keep
  the oracle's outputs (see the gaps below).
- `-diagnostic-fixtures` compiles every file of `tests/diagnostics.off` with both compilers and compares what they
  print.
- `-cli` runs both compilers in temporary projects. A scenario can name a file rbxtsc must write or text it must
  print, so a scenario can't pass because both compilers failed early. It covers missing and invalid tsconfigs, flag
  errors, type errors, `--noInclude`, `--type`, `--rojo`, declaration emit, a project that uses an `@rbxts` package and
  the compile order.

## Declaration emit

`--type package` and the tsconfig's `declaration` write `.d.ts` files, byte-identical to rbxtsc's in the `-cli`
scenarios. That includes `declarationDir`, `paths`, `baseUrl` imports, `import()` types, `.d.ts` inputs (printed again
as assets) and declaration errors, which skip every write.

typescript-go's `Emit` has no `afterDeclarations` hook, so `internal/roblox/project/declarations.go` edits the printed
text instead. That has two limits, listed below.

## Intentional differences

For these, `-cli` compares only the exit code:

- `--version`, `--help` and `--verbose` print goblox's own text.
- A tsconfig.json that isn't valid JSON can get another code from typescript-go's parser (TS1005 for TS1136).
- A Rojo project file that isn't valid JSON has Go's text after `SyntaxError:`, not V8's.

`.d.ts.map` files must exist on both sides, but their contents aren't compared, as typescript-go's source map generator
picks other mappings.

## Known gaps

- **Project references aren't built.** rbxtsc's `ProjectGraph.ts` (379 lines, using TypeScript's solution builder)
  builds referenced projects first. goblox reports TS6305 until their outputs exist. Run `goblox -p <project>` for each
  one first. Because of this `-lune` doesn't compile `projects/common` and `projects/shared` with goblox, so those two
  projects are not covered by the spec run.
- **Plugins can't change declarations.** rbxtsc runs a plugin's `afterDeclarations` transformers, goblox has no such
  step.
- **`.d.ts.map` offsets.** A mapping after a rewritten module specifier on the same line is off by the difference in
  length. It only happens with `baseUrl` or `paths` imports.
- **Compile order in an incremental rebuild.** rbxtsc starts from the changed files; goblox replays the files of the
  rebuild in the order of a fresh build. Only which errors are reported can differ, and only with several errors.
