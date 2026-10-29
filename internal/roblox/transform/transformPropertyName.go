package transform

// Port of TSTransformer/nodes/transformPropertyName.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformPropertyName is TSTransformer/nodes/transformPropertyName.ts transformPropertyName.
func transformPropertyName(state *TransformState, prereqs *Prereqs, name *ast.Node) luau.Expression {
	// identifier directly is from `{ a: value }`, so key must be "a"
	if ast.IsIdentifier(name) {
		return luau.String(name.Text())
	} else {
		// `name.expression`, if identifier, is from `{ [a]: value }`
		if ast.IsComputedPropertyName(name) {
			return transformExpression(state, prereqs, name.Expression())
		}
		return transformExpression(state, prereqs, name)
	}
}
