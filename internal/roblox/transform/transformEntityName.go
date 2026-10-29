package transform

// Port of TSTransformer/nodes/transformEntityName.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformEntityName is TSTransformer/nodes/transformEntityName.ts transformEntityName.
func transformEntityName(state *TransformState, node *ast.Node) luau.Expression {
	if ast.IsIdentifier(node) {
		validateIdentifier(node)
		return transformIdentifier(state, node)
	}
	return transformQualifiedName(state, node)
}

func transformQualifiedName(state *TransformState, node *ast.Node) *luau.PropertyAccessExpression {
	qualifiedName := node.AsQualifiedName()
	return luau.Property(convertToIndexableExpression(transformEntityName(state, qualifiedName.Left)), qualifiedName.Right.Text())
}
