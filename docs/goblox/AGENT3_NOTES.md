# Agent 3 notes (classes, destructuring)

## Ported

- `nodes/binding/*` and `util/binding/*`: transformBindingName, transformArrayBindingPattern,
  transformObjectBindingPattern, transformArrayAssignmentPattern, transformObjectAssignmentPattern,
  transformAssignmentPatternElement, transformBindingPattern, objectAccessor, createObjectRest,
  createArrayBindingAccessor, getAssignmentPatternType.
- `nodes/expressions/transformBinaryExpression.ts`, the whole file. The non-destructuring branches only
  dispatch to agent 1's helpers.
- `nodes/class/*`: transformClassLikeDeclaration, transformClassConstructor (both functions),
  transformPropertyDeclaration, transformDecorators. Also transformClassDeclaration,
  transformClassExpression, `util/findConstructor.ts` and `util/getExtendsNode.ts`.

## Shared additions

- `internal/checker/roblox_exports.go`: `(*Checker).GetNumberLiteralType(jsnum.Number)`, needed by
  getAssignmentPatternType (TypeScript's `checker.getNumberLiteralType`).
- `agent3_shims.go`: `getStatementsEffects(*luau.List)`. In TS, `getEffects` also accepts a
  `luau.List` and returns the joined effects of its statements. The Go `getEffects` only takes a
  `luau.Node`, so the list case lives here. If `getEffects` is widened to `luau.ListOrNode`, replace the
  callers and delete the shim.

## Signature notes (not changed)

- `getSimpleAssignmentOperator` returns `luau.AssignmentOperator`. TypeScript returns `undefined` when there
  is no simple operator. transformBinaryExpression reads `""` as `undefined`, so the port must return `""`
  in that case.
- `transformIdentifierDefined` returns `luau.AnyIdentifier`. TypeScript returns `luau.Identifier`.
  transformClassConstructor casts to `*luau.Identifier` for `.Name` (parameter properties). That matches
  TS as long as the port really returns an `*luau.Identifier`.
- `NO_EFFECTS` is not defined in Go. transformBinaryExpression's optimized array assignment joins the
  effects of each write target. It skips the property-access index (NO_EFFECTS in TS), because joining
  it is the identity.
- `BindingAccessor` (createArrayBindingAccessor.go) keeps the stub's interface shape: `Read(prereqs,
  index, omitted)` / `Rest(prereqs, index)`.

## Tests

- `agent3_class_test.go` covers findConstructor, getExtendsNode, countDecorators and
  transformImplicitClassConstructor (rendered output, checked against `class.spec.luau`).
- Everything else waits for end-to-end fixtures. class.spec, decorator.spec, destructure.spec,
  restDestructure.spec and assignment.spec need agent 1's transformIdentifier, transformVariableStatement,
  transformParameters, transformMethodDeclaration, transformBlock, transformPropertyName, transformWritable,
  transformInitializer, validateIdentifier, validateMethodAssignment, expressionMightMutate,
  getSimpleAssignmentOperator and evaluation/*.
