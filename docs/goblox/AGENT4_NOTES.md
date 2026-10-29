# Agent 4 notes (macros)

## Signature issues

- Resolved: the fixed `getEffects(node luau.Node)` cannot take the statement list that roblox-ts's
  `ensureTransformOrder` passes (`getEffects(prereqStatements)`). Agent 1's effects port added
  `getStatementsEffects(*luau.List)` for that, and `ensureTransformOrder` uses it.

## Port notes

- `PROPERTY_CALL_MACROS` is filled in `init()`: the shared method lists (`READONLY_SET_MAP_SHARED_METHODS`, ...)
  are functions that return fresh maps, so each class gets its own copy, like the object spreads of the TypeScript
  source, and `wrapComments` labels each with its own class name (`ReadonlySet.size`, `ReadonlyMap.size`).
- The spec of each macro is checked against roblox-ts 68648b0 output in `propertyCallMacros_test.go`,
  `constructorMacros_test.go`, and `ReadonlyArray.join` (which needs a checker for `argumentsWithDefaults` and the
  index type) in `propertyCallMacrosJoin_test.go` with an in-memory noLib program.
- Shared helpers ported by agent 4 (they were stubs): `offset`, `getLiteralNumberValue`,
  `convertToIndexableExpression`, `isUsedAsStatement`, `transformMacroCall` (nodes/transformMacroCall.ts, the entry
  point of call and property call macros). `ensureTransformOrder` was ported by agent 1 and by agent 4 at the same
  time; agent 1's version was kept.
- `callMacros`, `identifierMacros` and `macros_types` came hand-ported in the skeleton; reviewed against 68648b0, they
  match 1:1.
- `TestConstructorMacrosWithArguments` skips itself (`skipIfNotPorted`) while `transformStringLiteral` is a stub, and
  runs once it lands.
- Every property call macro class/method has an oracle-checked case, or shares its helper with one (the math operator
  methods). The cases come from `macros.server.ts` and `macros2.server.ts` in agent 4's private oracle copy
  (`.t3\scratch\2026-10-29-act-as-the-implementation-sub-cb82ac38\oracle\tests`).
- (Superseded, see below.) End-to-end runs of both macro fixtures used to panic in agent 1's
  `transformCallExpression` stub. Still stubs after it on the macro path: the `transformCallExpression` `*Inner`
  functions, `transformPropertyAccessExpression`, `transformNewExpression`, `transformStringLiteral`.

## Module syntax (taken over from agent 1)

- `transformExportDeclaration`, `transformModuleDeclaration`, `transformImportEqualsDeclaration` and
  `transformEnumDeclaration` were already ported by agent 1. Reviewed line by line against 68648b0: they match
  (`countImportExpUses` is `countExportImportExpUses` because transformImportDeclaration.ts has a function with the same
  name; `hasDeclareModifier` checks only syntactic `declare` like `ts.getModifiers`; enum constant values are
  `jsnum.Number` or `string`, as the type switch expects). Their helpers have no stubs left. The fixtures `enum`,
  `namespace`, `moduleSyntax` (and its 15 module files), `exportLet`, `robloxEnum` and `runtimeImport` match.
- The macro fixtures (`macros.server.ts`, `macros2.server.ts` in agent 4's private oracle copy) match end to end.
