package transform

// Port of TSTransformer/nodes/transformWritable.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformWritableExpression is TSTransformer/nodes/transformWritable.ts transformWritableExpression.
func transformWritableExpression(state *TransformState, prereqs *Prereqs, node *ast.Node, readAfterWrite bool) luau.WritableExpression {
	panic(todo("nodes/transformWritable.ts", "transformWritableExpression"))
}

// transformWritableAssignment is TSTransformer/nodes/transformWritable.ts transformWritableAssignment.
func transformWritableAssignment(state *TransformState, prereqs *Prereqs, writeNode *ast.Node, valueNode *ast.Node, readAfterWrite bool, readBeforeWrite bool) (writable luau.WritableExpression, readable luau.WritableExpression, value luau.Expression) {
	panic(todo("nodes/transformWritable.ts", "transformWritableAssignment"))
}

// captureWritableAssignmentTarget is TSTransformer/nodes/transformWritable.ts captureWritableAssignmentTarget.
func captureWritableAssignmentTarget(prereqs *Prereqs, writable luau.WritableExpression, prereqEffects *EvaluationEffects, valueEffects *EvaluationEffects) luau.WritableExpression {
	panic(todo("nodes/transformWritable.ts", "captureWritableAssignmentTarget"))
}
