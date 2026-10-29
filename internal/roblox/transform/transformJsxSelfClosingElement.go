package transform

// Port of TSTransformer/nodes/expressions/transformJsxSelfClosingElement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformJsxSelfClosingElement is TSTransformer/nodes/expressions/transformJsxSelfClosingElement.ts transformJsxSelfClosingElement.
func transformJsxSelfClosingElement(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	element := node.AsJsxSelfClosingElement()
	return transformJsx(state, prereqs, node, element.TagName, element.Attributes, []*ast.Node{})
}
