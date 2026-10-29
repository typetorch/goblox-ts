package transform

// Port of TSTransformer/nodes/expressions/transformTypeExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformTypeExpression is TSTransformer/nodes/expressions/transformTypeExpression.ts transformTypeExpression.
func transformTypeExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformTypeExpression.ts", "transformTypeExpression"))
}
