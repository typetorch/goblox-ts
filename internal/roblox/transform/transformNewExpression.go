package transform

// Port of TSTransformer/nodes/expressions/transformNewExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformNewExpression is TSTransformer/nodes/expressions/transformNewExpression.ts transformNewExpression.
func transformNewExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformNewExpression.ts", "transformNewExpression"))
}
