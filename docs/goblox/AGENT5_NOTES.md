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
