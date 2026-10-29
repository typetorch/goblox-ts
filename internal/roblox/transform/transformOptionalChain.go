package transform

// Port of TSTransformer/nodes/transformOptionalChain.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// ChainItem is the union of the TypeScript chain item interfaces. TODO(port): shape is free, only this file uses it.
type ChainItem struct {
	Kind               int
	Node               *ast.Node
	Optional           bool
	Name               string
	Expression         *ast.Node
	ArgumentExpression *ast.Node
	CallOptional       bool
	Args               []*ast.Node
}

// flattenOptionalChain is TSTransformer/nodes/transformOptionalChain.ts flattenOptionalChain.
func flattenOptionalChain(expression *ast.Node) ([]ChainItem, *ast.Node) {
	panic(todo("nodes/transformOptionalChain.ts", "flattenOptionalChain"))
}

// transformOptionalChain is TSTransformer/nodes/transformOptionalChain.ts transformOptionalChain.
func transformOptionalChain(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/transformOptionalChain.ts", "transformOptionalChain"))
}
