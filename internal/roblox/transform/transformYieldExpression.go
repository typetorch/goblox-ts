package transform

// Port of TSTransformer/nodes/expressions/transformYieldExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformYieldExpression is TSTransformer/nodes/expressions/transformYieldExpression.ts transformYieldExpression.
func transformYieldExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformYieldExpression.ts", "transformYieldExpression"))
}
