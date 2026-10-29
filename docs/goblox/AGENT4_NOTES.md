# Agent 4 notes (macros)

## Signature issues

- Resolved: the fixed `getEffects(node luau.Node)` cannot take the statement list that roblox-ts's
  `ensureTransformOrder` passes (`getEffects(prereqStatements)`). Agent 1's effects port added
  `getStatementsEffects(*luau.List)` for that, and `ensureTransformOrder` (ported by agent 4) uses it.

## Port notes

- `PROPERTY_CALL_MACROS` is filled in `init()`: the shared method lists (`READONLY_SET_MAP_SHARED_METHODS`, ...)
  are functions that return fresh maps, so each class gets its own copy, like the object spreads of the TypeScript
  source, and `wrapComments` labels each with its own class name (`ReadonlySet.size`, `ReadonlyMap.size`).
- The spec of each macro is checked against roblox-ts 68648b0 output in `propertyCallMacros_test.go`,
  `constructorMacros_test.go`, and `ReadonlyArray.join` (which needs a checker for `argumentsWithDefaults` and the
  index type) in `propertyCallMacrosJoin_test.go` with an in-memory noLib program.
- Shared helpers ported by agent 4 (they were stubs): `offset`, `getLiteralNumberValue`,
  `convertToIndexableExpression`, `isUsedAsStatement`, `ensureTransformOrder`.
- `TestConstructorMacrosWithArguments` skips itself (`skipIfNotPorted`) while `transformNumericLiteral` and the
  other literal transforms are stubs, and runs once they land.
