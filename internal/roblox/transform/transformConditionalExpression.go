package transform

// Port of TSTransformer/nodes/expressions/transformConditionalExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformConditionalExpression is TSTransformer/nodes/expressions/transformConditionalExpression.ts transformConditionalExpression.
func transformConditionalExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformConditionalExpression.ts", "transformConditionalExpression"))
}
