# Agent 3 notes (classes, destructuring, then the stubs taken over from agent 1)

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
- Taken over from agent 1 (phase 2): `util/evaluation/*` (effects, facts, bindings, builtins, substitute,
  events, plan), `util/validateIdentifier.ts`, transformPropertyName, transformParameters,
  transformReturnStatement, transformFunctionDeclaration, transformFunctionExpression (function expressions
  and arrow functions), transformMethodDeclaration, transformVariableStatement (with transformVariable,
  transformVariableDeclaration, transformVariableDeclarationList). transformExportAssignment was already
  ported by agent 1, so it was skipped.
- Shared helpers ported because the files above needed them: `util/wrapExpressionStatement.ts`,
  `util/isMethod.ts`, `util/ensureTransformOrder.ts`, `util/arrayLikeExpressionContainsSpread.ts`,
  `util/wrapStatementsAsGenerator.ts`, `util/arrayBindingPatternContainsHoists.ts`, `nodes/transformInitializer.ts`
  and `transformIdentifierDefined` (only that function). checkVariableHoist and validateNotAny were landed by
  other agents first with the same content, so agent 3's copies dropped out on rebase.

## Shared additions

- `internal/checker/roblox_exports.go`: `(*Checker).GetNumberLiteralType(jsnum.Number)` (getAssignmentPatternType),
  and `ConditionalType.Root`, `ConditionalRoot.Node`, `ConditionalRoot.CheckType` (isMethod's
  canChangeReceiverConvention; agent 5 added the same three at the same time and dropped its copy).
- `internal/roblox/luau/node.go`: `NodeBase` has a `facts any` field with `luau.GetFacts` / `luau.SetFacts`.
  It holds the evaluation facts that TS stores in symbol-keyed node fields. A clone (including the
  clone `luau.Create` makes of an already parented child) copies them, as the TS spread does.
  `luau.ShallowClone(node)` is TS's `{ ...node }` (used by substitute and transformIdentifierDefined).
- `NO_EFFECTS` / `UNKNOWN_EFFECTS` are pointer constants; compare with `==`. `getEffects` takes a
  `luau.Node`; the TS list case is `getStatementsEffects(*luau.List)` (or `getEffectsOfListOrNode`).
  The old `agent3_shims.go` is gone.
- `EvaluationEvent.OperandIndex` / `EvaluationReference.OperandIndex` are -1 when absent.
- A `Binding` is `any`: a `*BindingKey`, a `string` (identifier name), an `int` (temp id) or the
  `USER_READS` / `USER_WRITES` sentinels.
- `ensureTransformOrder` treats a nil transformer as the TS default, `transformExpression`.

## Signature notes (not changed)

- `getSimpleAssignmentOperator` returns `luau.AssignmentOperator`. TypeScript returns `undefined` when there
  is no simple operator. transformBinaryExpression reads `""` as `undefined`, so the port must return `""`
  in that case.
- `transformIdentifierDefined` returns `luau.AnyIdentifier`. For a symbol in `state.SymbolToIdMap` it is a
  `*luau.TemporaryIdentifier` (as in TS). transformClassConstructor casts to `*luau.Identifier` for `.Name`
  (parameter properties); constructor parameters are never in SymbolToIdMap (only for-loop variables are),
  so the cast holds.
- `transformVariable` returns `luau.Expression` (TS: the identifier or the `exports` access) and takes a nil
  `right` for TS's `undefined`.
- `BindingAccessor` (createArrayBindingAccessor.go) keeps the stub's interface shape: `Read(prereqs,
  index, omitted)` / `Rest(prereqs, index)`.

## Tests

- `agent3_class_test.go` covers findConstructor, getExtendsNode, countDecorators and
  transformImplicitClassConstructor (rendered output, checked against `class.spec.luau`).
- `agent3_evaluation_test.go` covers facts surviving clones, effects, substitute and planEvaluation.
- Fixtures: class.spec, decorator.spec, assignment.spec, destructure.spec and restDestructure.spec all stop
  at agent 1's transformCallExpression stub (the `-run "class|decorator|destructure|restDestructure|assignment"`
  run: 0 match, 0 differ, 5 panic). The whole suite is at 11 match, 0 differ, 62 panic.
