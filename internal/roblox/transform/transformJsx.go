package transform

// Port of TSTransformer/nodes/jsx/transformJsx.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformJsx is TSTransformer/nodes/jsx/transformJsx.ts transformJsx.
func transformJsx(state *TransformState, prereqs *Prereqs, node *ast.Node, tagName *ast.Node, attributes *ast.Node, children []*ast.Node) *luau.CallExpression {
	panic(todo("nodes/jsx/transformJsx.ts", "transformJsx"))
}
