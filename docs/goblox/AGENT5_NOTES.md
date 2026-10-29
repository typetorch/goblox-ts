# Agent 5 notes (loops, control flow, optional chaining, logical operators)

## Shared additions

- `internal/checker/roblox_exports_references.go` (new file, kept apart from `roblox_exports.go` only so
  that agents do not conflict on its import block):
  - `(*Checker).EachSymbolReferenceInFile(definition, sourceFile, cb, searchContainer)` is TypeScript's
    `ts.FindAllReferences.Core.eachSymbolReferenceInFile` (cb returns bool; nil container = whole file).
    `util/checkVariableHoist.ts` needs the same function.
  - `(*Checker).IsSymbolReferencedInFileWithin(definition, sourceFile, searchContainer)` is
    `ts.FindAllReferences.Core.isSymbolReferencedInFile` with its container argument. typescript-go's
    own `IsSymbolReferencedInFile` has no container.
  - typescript-go's `getPossibleSymbolReferencePositions` returns positions relative to the container
    for a container that is not the source file, so these use their own position search.
- `internal/roblox/luau/setparent.go`: `luau.SetParent(node, parent)`, needed by
  transformForStatement's `addFinalizers` (`node.parent = statement.parent`).
- Shared helpers ported because they were needed: `util/createTruthinessChecks.ts` (both functions),
  `binaryExpressionChain` in `util/expressionChain.ts`.

## Port status

All agent 5 files are ported 1:1 (no `todo` left): `transformForOfStatement.ts`, `transformForStatement.ts`,
`transformWhileStatement.ts`, `transformDoStatement.ts`, `transformBreakStatement.ts`,
`transformContinueStatement.ts`, `transformSwitchStatement.ts`, `transformTryStatement.ts`,
`transformThrowStatement.ts`, `transformOptionalChain.ts`, `transformLogical.ts`,
`transformLogicalOrCoalescingAssignmentExpression.ts`, `util/getStatements.ts` and
`util/isBlockedByTryStatement.ts`.

Port notes:

- `TupleType.combinedFlags` is unexported in typescript-go; transformForOfStatement ORs `ElementFlags()`
  instead (`tupleCombinedFlags`).
- The LuaTuple element names come from `TupleType.ElementInfos()[i].LabeledDeclaration()` (TypeScript's
  `labeledElementDeclarations`).

## Test status

- Unit tests (`agent5_controlflow_test.go`) cover the luau-only helpers `collapseFlowControlCases` and
  `binaryExpressionChain` against roblox-ts 68648b0 output.
- The statement transforms are not tested end to end yet. Every oracle spec panics on
  `transformExportAssignment`, and a top-level rewrite still hits `transformFunctionDeclaration`,
  `transformIdentifier`, `transformVariableStatement`, `transformExpressionStatement`, and so on (agent 1).
- A ready private fixture: copy `oracle/tests/{node_modules,include,package.json}` to
  `<dir>/tests`, add a tsconfig without `references`, a `default.project.json` that maps `include`,
  `node_modules/@rbxts` and `out/tests`, a `src/globals.d.ts` with
  `/// <reference types="@rbxts/testez/globals" />`, and copies of `loop`, `switch`, `try`, `optional` and
  `truthiness` `.spec.ts` with `export = () => {` rewritten to `export function run() {`. Build the expected
  output with `NODE_PATH=<oracle>/out node <oracle>/out/CLI/cli.js build -p <dir>/tests`, then run
  `go run ./cmd/goblox-fixtures -oracle <dir>`. Rerun it once agent 1's statement and expression
  transforms land.
