package transform

// Port of TSTransformer/nodes/expressions/transformIdentifier.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformIdentifierDefined is TSTransformer/nodes/expressions/transformIdentifier.ts transformIdentifierDefined.
func transformIdentifierDefined(state *TransformState, node *ast.Node) luau.AnyIdentifier {
	var symbol *ast.Symbol
	if ast.IsShorthandPropertyAssignment(node.Parent) {
		symbol = state.TypeChecker.GetShorthandAssignmentValueSymbol(node.Parent)
	} else {
		symbol = state.TypeChecker.GetSymbolAtLocation(node)
	}
	assert(symbol != nil)

	if replacementId, ok := state.SymbolToIdMap[symbol]; ok {
		identifier := luau.ShallowClone(replacementId)
		markBinding(identifier, symbol, getBindingKey(state, symbol, node))
		return identifier
	}

	identifier := luau.Create(&luau.Identifier{
		Name: node.Text(),
	})
	markBinding(identifier, symbol, getBindingKey(state, symbol, node))
	tryMarkBuiltinLibrary(state, node, identifier)
	if effects, ok := state.MultiTransformState.FunctionEffects[symbol]; ok {
		setCallEffects(identifier, effects)
	}
	return identifier
}

// transformIdentifier is TSTransformer/nodes/expressions/transformIdentifier.ts transformIdentifier.
func transformIdentifier(state *TransformState, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformIdentifier.ts", "transformIdentifier"))
}
