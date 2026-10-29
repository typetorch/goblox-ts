package transform

// Port of TSTransformer/nodes/jsx/transformJsx.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformJsx is TSTransformer/nodes/jsx/transformJsx.ts transformJsx.
func transformJsx(state *TransformState, prereqs *Prereqs, node *ast.Node, tagName *ast.Node, attributes *ast.Node, children []*ast.Node) *luau.CallExpression {
	// jsxFactoryEntity seems to always be defined and will default to `React.createElement`
	jsxFactoryEntity := state.TypeChecker.GetJsxFactoryEntity(node)
	assert(jsxFactoryEntity != nil, "Expected jsxFactoryEntity to be defined")

	createElementExpression := convertToIndexableExpression(transformEntityName(state, jsxFactoryEntity))

	tagNameExp := transformJsxTagName(state, tagName)

	var attributesPtr *MapPointer
	if len(attributes.AsJsxAttributes().Properties.Nodes) > 0 {
		attributesPtr = createMapPointer("attributes")
		transformJsxAttributes(state, prereqs, attributes, attributesPtr)
	}

	transformedChildren := transformJsxChildren(state, prereqs, children)

	args := []luau.Expression{tagNameExp}

	if attributesPtr != nil {
		args = append(args, attributesPtr.Value)
	} else if len(transformedChildren) > 0 {
		args = append(args, luau.Nil())
	}

	args = append(args, transformedChildren...)

	return luau.Call(createElementExpression, args...)
}
