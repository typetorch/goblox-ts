package transform

// Port of TSTransformer/nodes/expressions/transformArrayLiteralExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformArrayLiteralExpression is TSTransformer/nodes/expressions/transformArrayLiteralExpression.ts transformArrayLiteralExpression.
func transformArrayLiteralExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformArrayLiteralExpression.ts", "transformArrayLiteralExpression"))
}
