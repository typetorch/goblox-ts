# Agent 4 notes (macros)

## Signature issues

- `getEffects(node luau.Node)` in `evaluation_effects.go` cannot take a `luau.List`, but roblox-ts's
  `getEffects(node: luau.Node | luau.List<luau.Statement>)` is called with a statement list by
  `ensureTransformOrder` (`getEffects(prereqStatements)`), so `ensureTransformOrder` cannot be ported 1:1 against
  the fixed signature. I left `ensureTransformOrder` as a stub for agent 1, who owns `util/evaluation/*`.
  Suggested fix: widen it to `getEffects(node luau.ListOrNode)` and return `NO_EFFECTS` for an empty list, as the
  TypeScript version does. `NO_EFFECTS` is not declared in Go yet either.
  Until that lands, `new Array(n)`, `new Set([...])` (literal without spreads) and all non-macro paths that use
  `ensureTransformOrder` panic with `todo(util/ensureTransformOrder.ts, ...)`.

## Port notes

- `PROPERTY_CALL_MACROS` is filled in `init()`: the shared method lists (`READONLY_SET_MAP_SHARED_METHODS`, ...)
  are functions that return fresh maps, so each class gets its own copy, like the object spreads of the TypeScript
  source, and `wrapComments` labels each with its own class name (`ReadonlySet.size`, `ReadonlyMap.size`).
- The spec of each macro is checked against roblox-ts 68648b0 output in `propertyCallMacros_test.go`,
  `constructorMacros_test.go`, and `ReadonlyArray.join` (which needs a checker for `argumentsWithDefaults` and the
  index type) in `propertyCallMacrosJoin_test.go` with an in-memory noLib program.
- Shared helpers ported by agent 4 (they were stubs): `offset`, `getLiteralNumberValue`,
  `convertToIndexableExpression`, `isUsedAsStatement`.
