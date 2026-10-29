package transform

// Port of TSTransformer/nodes/expressions/transformThisExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformThisExpression is TSTransformer/nodes/expressions/transformThisExpression.ts transformThisExpression.
func transformThisExpression(state *TransformState, node *ast.Node) luau.AnyIdentifier {
	panic(todo("nodes/expressions/transformThisExpression.ts", "transformThisExpression"))
}
