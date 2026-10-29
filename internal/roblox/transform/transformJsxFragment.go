package transform

// Port of TSTransformer/nodes/expressions/transformJsxFragment.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformJsxFragment is TSTransformer/nodes/expressions/transformJsxFragment.ts transformJsxFragment.
func transformJsxFragment(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.CallExpression {
	resolver := state.TypeChecker.GetEmitResolver()
	jsxFactoryEntity := resolver.GetJsxFactoryEntity(node)
	assert(jsxFactoryEntity != nil, "Expected jsxFactoryEntity to be defined")

	createElementExpression := convertToIndexableExpression(transformEntityName(state, jsxFactoryEntity))

	// getJsxFragmentFactoryEntity() doesn't seem to default to "Fragment"..
	// but the typechecker does, so we should follow that behavior
	jsxFragmentFactoryEntity := resolver.GetJsxFragmentFactoryEntity(node)
	if jsxFragmentFactoryEntity == nil {
		jsxFragmentFactoryEntity = parser.ParseIsolatedEntityName("Fragment")
	}
	assert(jsxFragmentFactoryEntity != nil, "Unable to find valid jsxFragmentFactoryEntity")

	args := []luau.Expression{transformEntityName(state, jsxFragmentFactoryEntity)}

	transformedChildren := transformJsxChildren(state, prereqs, node.AsJsxFragment().Children.Nodes)

	// props parameter
	if len(transformedChildren) > 0 {
		args = append(args, luau.Nil())
	}

	args = append(args, transformedChildren...)

	return luau.Call(createElementExpression, args...)
}
