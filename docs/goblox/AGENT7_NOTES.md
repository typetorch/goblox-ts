# Agent 7 notes: template strings and unclaimed misc expressions

## Census (2026-10-30, `goblox` at 15f1ec288)

`grep -rn 'todo("' internal/roblox` finds no stubs: every function in the skeleton is ported. The only
`todo` left is the helper itself in `internal/roblox/transform/todo.go`.

Agent 7 claims these files. All of them were already ported (agent 1, mostly in 19ec41ef2), so agent 7
reviewed them line by line against roblox-ts 68648b0 and kept them:

| File | Status |
|---|---|
| `transformTemplateExpression.go` | 1:1, `ensureTransformOrder` prereqs as in TS |
| `transformTaggedTemplateExpression.go` | 1:1 (cooked `Text()` of the head and literals, as `node.text` in TS) |
| `transformNoSubstitutionTemplateLiteral.go` | 1:1 |
| `transformInterpolatedStringPart.go` | 1:1 |
| `createStringFromLiteral.go` (used by template parts) | 1:1; the U+2028 / U+2029 checks are literal characters |
| `transformConditionalExpression.go` | 1:1 |
| `transformAwaitExpression.go` | 1:1 |
| `transformYieldExpression.go` | 1:1, including the `yield*` temp ids |
| `transformTypeExpression.go` (as, `!`, satisfies, `<T>x`, ExpressionWithTypeArguments) | 1:1 (also listed under agent 5: shared, unchanged) |
| `transformThisExpression.go`, `transformSuperKeyword.go`, `getAssignableValue.go` | agent 1's ports, kept |

The `TRANSFORMER_BY_KIND` table in `transformExpression.go` has the same 45 kinds as the TS table, so no
expression kind reaches the `Unknown expression` assert.

Owned by other agents, not touched: `transformSpreadElement`, `transformDeleteExpression`, the literals,
unary, `void` (agent 5), JSX (agent 6), calls and `new` (agent 3), macros (agent 4).

## Tests

`internal/roblox/transform/agent7_template_test.go` (package `transform_test`) builds a temporary project with
`project.Build` and compares each output byte for byte with real roblox-ts output:

- `testdata/agent7/<name>.ts` are the sources and `testdata/agent7/<name>.luau` the output of roblox-ts
  68648b0 for them, copied unchanged. Keeping them as files avoids quoting backticks and backslashes in Go.
- Cases: `templatePlain` (escapes `\t`, `` \` ``, `\uXXXX`, surrogate pairs, `\$`, braces, `\\u`, line
  continuations, multi-line), `templateSubstitution` (spans, empty head and tail, prereqs from `f()` and
  assignments, table expressions wrapped in parentheses), `templateTagged` (strings array, empty strings,
  property tags with prereqs, cooked escapes), and `misc` (conditional expression, statement and prereq forms,
  `!` / `as` / `satisfies` / `<T>`, `await`, `yield`, `yield*` as a value and as a statement).
- `MacroManager` needs `@rbxts/compiler-types` and `@rbxts/types`, which are not vendored. The test copies
  them from the oracle (`GOBLOX_ORACLE`, defaulting to the path in PORTING.md) and skips when they are missing.
- To add a case: write the snippet into a scratch roblox-ts project (not into the shared oracle's `tests/`),
  build it with the oracle CLI, then copy the `.ts` and `.luau` into `testdata/agent7/`.
- Note for agents writing snippets through tool calls: `\uXXXX` and `\\` sequences can be decoded before they
  reach the file. Check the bytes with `od -c`.

## Fixture suite

`go run ./cmd/goblox-fixtures -oracle <oracle>` on `goblox` 15f1ec288: **73 match, 0 differ, 0 panic**,
0 missing, 0 extra, 0 diagnostics.
