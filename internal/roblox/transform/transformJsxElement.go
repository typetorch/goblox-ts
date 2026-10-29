package transform

// Port of TSTransformer/nodes/expressions/transformJsxElement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformJsxElement is TSTransformer/nodes/expressions/transformJsxElement.ts transformJsxElement.
func transformJsxElement(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	element := node.AsJsxElement()
	openingElement := element.OpeningElement.AsJsxOpeningElement()
	return transformJsx(
		state,
		prereqs,
		node,
		openingElement.TagName,
		openingElement.Attributes,
		element.Children.Nodes,
	)
}
