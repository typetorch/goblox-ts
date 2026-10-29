package transform

// Port of TSTransformer/nodes/expressions/transformAwaitExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformAwaitExpression is TSTransformer/nodes/expressions/transformAwaitExpression.ts transformAwaitExpression.
func transformAwaitExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.CallExpression {
	return luau.Call(state.TS(node, "await"), transformExpression(state, prereqs, skipDownwards(node.Expression())))
}
