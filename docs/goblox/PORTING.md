# Porting the transformer

`internal/roblox/transform` is a 1:1 port of roblox-ts's `src/TSTransformer` (roblox-ts at 68648b0,
`goblox-ts-ref/roblox-ts`). Every exported TypeScript function already has a Go stub with a fixed signature that
panics with `todo(file, name)`. Porting a function means replacing its stub body. Change a shared signature
only if you must, and record the change in `docs/goblox/AGENT3_NOTES.md` (agent 3) or here (agent 1).

## Conventions

The full list is in `internal/roblox/transform/doc.go`. In short:

- One Go file per TS file, with the same base name. Files under `util/evaluation/` are prefixed `evaluation_`,
  and `macros/types.ts` is `macros_types.go`. Function names stay the roblox-ts names.
- `ts.Node` is `*ast.Node`, `ts.SourceFile` is `*ast.SourceFile`, `ts.Type` is `*checker.Type` and `ts.Symbol` is
  `*ast.Symbol`. TS `Type` methods are the `typeX` helpers in `types.go`.
- JS numbers are `float64` and counts are `int`. `T | undefined` is `nil`, or an extra `bool` result for non-pointer
  types.
- Port `Prereqs`, temp ids, `solveTempIds` and `util/evaluation/*` exactly. Keep diagnostic messages byte-identical.
- Only code under `cmd/goblox`, `internal/roblox/...` and `pkg/goblox` may change. The one exception is
  `internal/checker/roblox_exports.go`, where you may add small exports of unexported checker functions.
- Run `go test` only on `./internal/roblox/...` and `./pkg/goblox/...`.

## Ownership

| Area | Owner |
|---|---|
| `nodes/class/*`, `util/findConstructor.ts`, `util/getExtendsNode.ts` and other class-only helpers | agent 3 |
| `nodes/binding/*`, `util/binding/*`, and the destructuring-assignment paths of `transformBinaryExpression` | agent 3 |
| `macros/*`: `propertyCallMacros`, `callMacros`, `constructorMacros`, `identifierMacros`, `macros_types` (`callMacros` and `identifierMacros` already ported by agent 1; review and keep) | agent 4 |
| Loops and control flow: for..of, for, while/do, break/continue and the loop helpers; switch, try, throw; `transformOptionalChain`; `transformLogical` and the `&&=`/`\|\|=`/`??=` assignment forms | agent 5 |
| Taken over from agent 1: `util/addOneIfArrayType.ts`, `util/isSymbolMutable.ts`, `util/isValidMethodIndexWithoutCall.ts`, `util/getConstantValueLiteral.ts`, `util/valueToIdStr.ts`, `transformNumericLiteral`, `transformBooleanLiteral`, `transformOmittedExpression`, `transformParenthesizedExpression`, `transformTypeExpression`, `transformVoidExpression`, `nodes/transformWritable.ts`, `util/checkVariableHoist.ts`, `util/expressionMightMutate.ts`, `transformIdentifier`, `transformStringLiteral` (with `util/createStringFromLiteral.ts`), `transformPropertyAccessExpression`, `transformElementAccessExpression`, `transformArrayLiteralExpression`, `transformSpreadElement`, `transformDeleteExpression`, `transformObjectLiteralExpression`, `transformUnaryExpression` | agent 5 |
| Taken over from agent 1: `util/evaluation/*` (with `util/wrapExpressionStatement.ts`), `util/validateIdentifier.ts`, `transformVariableStatement` (and `transformVariableDeclaration`), `transformFunctionDeclaration`, `transformFunctionExpression` (function expressions and arrow functions), `transformParameters`, `transformMethodDeclaration`, `transformPropertyName`, `transformExportAssignment` (already ported by agent 1 with the coordinator's go-ahead: review and keep), `transformReturnStatement` | agent 3 |
| Taken over from agent 1: `transformCallExpression` and its inner functions (call, property call, element call, super calls), `transformNewExpression` (both already ported by agent 1: review and keep) | agent 3 |
| Taken over from agent 1: `transformExportDeclaration`, `transformModuleDeclaration` (namespaces), `transformImportEqualsDeclaration`, `transformEnumDeclaration` (all already ported by agent 1: review and keep) | agent 4 |
| JSX: `nodes/jsx/*`, `util/jsx*` (partly ported by agent 1) | agent 6 |
| Template expressions and tagged templates (partly ported by agent 1), and every remaining expression stub not listed elsewhere (conditional, spread, await, yield, delete, non-null/as/satisfies, ...). Agent 1 already ported `transformThisExpression`, `transformSuperKeyword`, `transformNoSubstitutionTemplateLiteral`, `transformDeleteExpression`, `util/getAssignableValue.ts`: review and keep. Templates + unclaimed misc expressions: all reviewed and kept, tests in `agent7_template_test.go`, census in `AGENT7_NOTES.md` | agent 7 |
| The driver (`internal/roblox/project`, `cmd/goblox`, `cmd/goblox-fixtures`), `MacroManager`, `transformSourceFile`, `transformStatementList`, `transformImportDeclaration` and the import/export plumbing (`createImportExpression`, module export ordering in `moduleExportsOrder.go`), and any stub nobody else owns | agent 1 |

Agent 1 does not port, or hand to subagents, the files agents 3, 4 and 5 own. Agents 3-5 work in their own
worktrees (`goblox-ts-agent3`, `-agent4`, `-agent5`) and land fast-forward pushes on `goblox`. Both agents land work on `goblox`, so
run `git pull --rebase` before you push. Never force-push.

## Oracle

These are reference outputs produced by the real roblox-ts compiler. They are not in this repository:

- Built roblox-ts:
  `C:\Users\phasenull\.t3\scratch\2026-10-29-act-as-the-implementation-sub-37576c8e\oracle`
- Its 73-file runtime test project: `oracle\tests` (sources in `tests\src`).
  - `tests\src\diagnostics` has been moved to `tests\diagnostics.off`, so the project builds with exit code 0.
- Expected Luau: `oracle\tests\out\**\*.luau`, 73 files (`main.server.luau`, `helpers/`, `tests/*.spec.luau`,
  `tests/moduleSyntax/`).
- To regenerate the expected Luau:

  ```
  cd oracle/tests && NODE_PATH=../out node ../out/CLI/cli.js --verbose --allowCommentDirectives
  ```

- To see what roblox-ts emits for one snippet, put it in a scratch `.ts` file under `oracle/tests/src` and
  rebuild. Restore the tree afterwards.

## Fixture diff harness

`go run ./cmd/goblox-fixtures -oracle <oracle dir>` (landing next, owned by agent 1) builds `oracle/tests` with
goblox into a temporary out directory. It never writes into the oracle's `out/`. It then diffs every `.luau`
file against `oracle/tests/out` and prints, per file, `match`, `differs` (with a unified diff under `-v`) or
`panic: todo(<file>, <function>)`. A file whose transform hits an unported stub fails alone; the other files
still compile.

## Signature answers (agent 1)

- `getSimpleAssignmentOperator` returns `""` when TypeScript returns `undefined` (no simple operator).
- `transformIdentifierDefined` returns `luau.AnyIdentifier` because, like TypeScript, it returns a copy of the
  `*luau.TemporaryIdentifier` in `state.SymbolToIdMap` when the symbol was renamed, and a fresh `*luau.Identifier`
  otherwise. Code that needs the name should switch on both types (both have `.Name`). Parameter properties are
  never in `SymbolToIdMap`, so the `*luau.Identifier` assertion in transformClassConstructor holds.

## Type package versions

goblox ports roblox-ts master, whose macro tables expect the newest `@rbxts/types` and `@rbxts/compiler-types`.
For example, the `vector` table needs the `add`, `sub`, `mul`, `div` and `idiv` declarations that older `@rbxts/types`
do not have in `macro_math.d.ts` (1.0.924 and 1.0.944 lack them, 1.0.957 has them).

**Deliberate difference from roblox-ts.** roblox-ts's `MacroManager` constructor throws a `ProjectError` ("MacroManager
could not find method for vector.add") for any macro or symbol the installed packages don't declare, whether or not
the project uses it. The goblox one skips it:

- A missing macro (identifier, call, constructor, or property call, down to one method) is left out of the table. Code
  can only reach a macro through a declaration, so a macro with no declaration is unused.
- A missing entry of `SYMBOL_NAMES` becomes a placeholder symbol, so identity checks such as `isArrayType` are false.
- Reading a missing symbol's declarations still throws that `ProjectError` (`RequireSymbol`), and so does a `LuaTuple`
  whose `compiler-types` has no `_nominal_LuaTuple`: compiling one as a plain value would change what the code does.
  Those errors come from the file that uses the symbol, and don't stop projects that don't use it.

roblox-ts 3.0.0 has no `vector` table, so these projects build with it. Master fails on them.

## TypeScript 7 differences that reach the transform

- **`Object` at the top level of a module.** TypeScript 7 added `checkCollisionWithGlobalObjectInGeneratedCode`, so
  with `"module": "commonjs"` it reports TS2441 for `import Object from "@rbxts/object-utils"`. `FilterCompatDiagnostics`
  drops that one case (compat.go). TS2441 for `require` and `exports` stays, since rbxtsc reports those too. Both
  compilers accept `module`, `Promise`, `Reflect`, `WeakMap`, `WeakSet` and `Map` at `target: ESNext`.
- **`export =` next to exported types.** TypeScript 7's binder flags the `export=` symbol of such a module as
  `NamespaceModule`, and `resolveAlias` does not follow it. The checker's `SkipAlias` then returns a symbol with no value
  meaning, and the import would be dropped with no diagnostic (`@rbxts/quickzone` 1.4.6 declares its types this way).
  The transform uses `skipAliases`, which goes on to the variable as TypeScript 5 does. The Go port of
  `@typetorch/transformer` still calls `SkipAlias` directly (`classes_transformState.go`).

## Running the harness

```
go run ./cmd/goblox-fixtures -oracle <oracle dir>                 # one line per file + summary
go run ./cmd/goblox-fixtures -oracle <oracle dir> -v -run object  # diffs and panic stacks for matching files
```

`goblox build` (`go run ./cmd/goblox -p <project>`) is the `rbxtsc build` equivalent; `goblox check` is the
Phase 0 type-check benchmark.
