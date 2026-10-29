package transform

// Port of TSTransformer/nodes/expressions/transformClassExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformClassExpression is TSTransformer/nodes/expressions/transformClassExpression.ts transformClassExpression.
func transformClassExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformClassExpression.ts", "transformClassExpression"))
}
