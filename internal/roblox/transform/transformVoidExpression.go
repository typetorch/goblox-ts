package transform

// Port of TSTransformer/nodes/expressions/transformVoidExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformVoidExpression is TSTransformer/nodes/expressions/transformVoidExpression.ts transformVoidExpression.
func transformVoidExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.NilLiteral {
	panic(todo("nodes/expressions/transformVoidExpression.ts", "transformVoidExpression"))
}
