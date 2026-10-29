package transform

// Port of TSTransformer/nodes/transformLogicalOrCoalescingAssignmentExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformLogicalOrCoalescingAssignmentExpression is TSTransformer/nodes/transformLogicalOrCoalescingAssignmentExpression.ts transformLogicalOrCoalescingAssignmentExpression.
func transformLogicalOrCoalescingAssignmentExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.WritableExpression {
	panic(todo("nodes/transformLogicalOrCoalescingAssignmentExpression.ts", "transformLogicalOrCoalescingAssignmentExpression"))
}

// transformLogicalOrCoalescingAssignmentExpressionStatement is TSTransformer/nodes/transformLogicalOrCoalescingAssignmentExpression.ts transformLogicalOrCoalescingAssignmentExpressionStatement.
func transformLogicalOrCoalescingAssignmentExpressionStatement(state *TransformState, node *ast.Node) *luau.List {
	panic(todo("nodes/transformLogicalOrCoalescingAssignmentExpression.ts", "transformLogicalOrCoalescingAssignmentExpressionStatement"))
}
