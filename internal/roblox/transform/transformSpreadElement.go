package transform

// Port of TSTransformer/nodes/expressions/transformSpreadElement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformSpreadElement is TSTransformer/nodes/expressions/transformSpreadElement.ts transformSpreadElement.
func transformSpreadElement(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.CallExpression {
	validateNotAnyType(state, node.Expression())

	// array literal is caught and handled separately in transformArrayLiteralExpression.ts
	assert(!ast.IsArrayLiteralExpression(node.Parent) && node.Parent.Arguments() != nil)
	arguments := node.Parent.Arguments()
	if arguments[len(arguments)-1] != node {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoPrecedingSpreadElement.Create(node))
	}

	expression := transformExpression(state, prereqs, node.Expression())

	type_ := state.GetType(node.Expression())
	if isDefinitelyType(type_, isArrayType(state)) {
		return luau.Call(luau.Globals.Unpack, expression)
	} else {
		addIterableToArrayBuilder := getAddIterableToArrayBuilder(state, node.Expression(), type_)
		arrayId := prereqs.PushToVar(luau.NewArray(), "array")
		lengthId := prereqs.PushToVar(luau.Number(0), "length")
		prereqs.PushList(addIterableToArrayBuilder(prereqs, expression, arrayId, lengthId, 0, false))
		return luau.Call(luau.Globals.Unpack, arrayId)
	}
}
