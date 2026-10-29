package transform

// Port of TSTransformer/nodes/statements/transformReturnStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func isTupleReturningCall(state *TransformState, tsExpression *ast.Node, luaExpression luau.Expression) bool {
	// intentionally NOT using state.getType() here, because that uses skipUpwards
	return luau.IsCall(luaExpression) &&
		isLuaTupleType(state)(state.TypeChecker.GetTypeAtLocation(skipDownwards(tsExpression)))
}

func isTupleMacro(state *TransformState, expression *ast.Node) bool {
	symbol := getFirstDefinedSymbol(state, state.GetType(expression.Expression()))
	if symbol != nil && symbol == state.Services.MacroManager.GetSymbolOrThrow(SYMBOL_NAMES.Tuple) {
		return true
	}
	return false
}

// transformReturnStatementInner is TSTransformer/nodes/statements/transformReturnStatement.ts transformReturnStatementInner.
func transformReturnStatementInner(state *TransformState, returnExp *ast.Node) *luau.List {
	result := luau.MakeList()
	prereqs := NewPrereqs()

	var expression luau.ListOrNode

	innerReturnExp := skipDownwards(returnExp)
	if ast.IsCallExpression(innerReturnExp) && isTupleMacro(state, innerReturnExp) {
		args := ensureTransformOrder(state, prereqs, innerReturnExp.Arguments(), nil)
		expression = luau.MakeList(args...)
	} else {
		transformed := transformExpression(state, prereqs, innerReturnExp)
		expression = transformed
		if isLuaTupleType(state)(state.GetType(returnExp)) && !isTupleReturningCall(state, returnExp, transformed) {
			if array, ok := transformed.(*luau.Array); ok {
				expression = array.Members
			} else {
				expression = luau.Call(luau.Globals.Unpack, transformed)
			}
		}
	}

	result.PushList(prereqs.Statements)

	if isReturnBlockedByTryStatement(returnExp) {
		state.MarkTryUses("usesReturn")
		var members *luau.List
		if list, ok := expression.(*luau.List); ok {
			members = list
		} else {
			members = luau.MakeList(expression.(luau.Node))
		}
		result.Push(luau.Create(&luau.ReturnStatement{
			Expression: luau.MakeList(
				state.TS(returnExp, "TRY_RETURN"),
				luau.Create(&luau.Array{
					Members: members,
				}),
			),
		}))
	} else {
		result.Push(luau.Create(&luau.ReturnStatement{Expression: expression}))
	}

	return result
}

// transformReturnStatement is TSTransformer/nodes/statements/transformReturnStatement.ts transformReturnStatement.
func transformReturnStatement(state *TransformState, node *ast.Node) *luau.List {
	if node.Expression() == nil {
		if isReturnBlockedByTryStatement(node) {
			state.MarkTryUses("usesReturn")
			return luau.MakeList(
				luau.Create(&luau.ReturnStatement{
					Expression: luau.MakeList(state.TS(node, "TRY_RETURN"), luau.NewArray()),
				}),
			)
		}
		return luau.MakeList(luau.Create(&luau.ReturnStatement{Expression: luau.Nil()}))
	}
	return transformReturnStatementInner(state, node.Expression())
}
