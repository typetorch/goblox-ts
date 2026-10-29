package transform

// Port of TSTransformer/util/isValidMethodIndexWithoutCall.ts.

import (
	"reflect"

	"github.com/microsoft/typescript-go/internal/ast"
)

// isValidMethodIndexWithoutCall is TSTransformer/util/isValidMethodIndexWithoutCall.ts isValidMethodIndexWithoutCall.
func isValidMethodIndexWithoutCall(state *TransformState, node *ast.Node) bool {
	parent := node.Parent
	// a.b !== undefined
	if ast.IsBinaryExpression(parent) {
		return true
	}

	// !a.b
	if ast.IsPrefixUnaryExpression(parent) {
		return true
	}

	// typeIs/typeOf macros
	if ast.IsCallExpression(parent) {
		expType := state.TypeChecker.GetNonOptionalType(state.GetType(parent.Expression()))
		symbol := getFirstDefinedSymbol(state, expType)
		if symbol != nil {
			macro := state.Services.MacroManager.GetCallMacro(symbol)
			if macro != nil &&
				// typeIs will be a TypeError if usage is not the first argument
				(sameCallMacro(macro, CALL_MACROS["typeIs"]) ||
					sameCallMacro(macro, CALL_MACROS["typeOf"])) {
				return true
			}
		}
	}

	return false
}

// sameCallMacro is `===` on two call macros.
func sameCallMacro(a, b CallMacro) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}
