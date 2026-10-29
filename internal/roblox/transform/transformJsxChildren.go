package transform

// Port of TSTransformer/nodes/jsx/transformJsxChildren.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformJsxChildren is TSTransformer/nodes/jsx/transformJsxChildren.ts transformJsxChildren.
func transformJsxChildren(state *TransformState, prereqs *Prereqs, children []*ast.Node) []luau.Expression {
	panic(todo("nodes/jsx/transformJsxChildren.ts", "transformJsxChildren"))
}
