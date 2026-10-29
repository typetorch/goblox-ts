package transform

// Port of TSTransformer/util/binding/createObjectRest.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// createObjectRest is TSTransformer/util/binding/createObjectRest.ts createObjectRest.
func createObjectRest(state *TransformState, prereqs *Prereqs, node *ast.Node, type_ *checker.Type, source luau.AnyIdentifier, keys []luau.Expression) luau.Expression {
	isUnsupported := func(t *checker.Type) bool { return isRobloxType(state)(t) && !isSharedTableType(state)(t) }
	unsupported := false
	walkTypes(type_, func(t *checker.Type) {
		unsupported = unsupported || isDefinitelyType(t, isUnsupported)
	})
	if unsupported {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoRestSpreadingOfRobloxTypes.Create(node))
		return luau.NewNone()
	}

	rest := prereqs.PushToVar(luau.NewMap(), "rest")
	key := luau.TempId("key")
	value := luau.TempId("value")
	statements := luau.MakeList(
		luau.Create(&luau.Assignment{
			Left:     luau.Create(&luau.ComputedIndexExpression{Expression: rest, Index: key}),
			Operator: "=",
			Right:    value,
		}),
	)
	if len(keys) > 0 {
		var condition luau.Expression
		if len(keys) == 1 {
			condition = luau.Binary(key, "~=", keys[0])
		} else {
			condition = luau.Unary(
				"not",
				luau.Create(&luau.ComputedIndexExpression{
					Expression: prereqs.PushToVar(luau.NewSet(keys...), "extracted"),
					Index:      key,
				}),
			)
		}
		statements = luau.MakeList(
			luau.Create(&luau.IfStatement{
				Condition:  condition,
				Statements: statements,
				ElseBody:   luau.MakeList(),
			}),
		)
	}
	prereqs.Push(
		luau.Create(&luau.ForStatement{
			Ids:        luau.MakeList(key, value),
			Expression: source,
			Statements: statements,
		}),
	)
	return rest
}
