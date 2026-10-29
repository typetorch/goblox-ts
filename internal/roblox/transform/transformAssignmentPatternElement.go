package transform

// Port of TSTransformer/nodes/binding/transformAssignmentPatternElement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformAssignmentPatternElement is TSTransformer/nodes/binding/transformAssignmentPatternElement.ts transformAssignmentPatternElement.
func transformAssignmentPatternElement(state *TransformState, prereqs *Prereqs, target *ast.Node, valuePrereqs *Prereqs, value luau.Expression, initializer *ast.Node) {
	target = skipDownwards(target)
	if ast.IsBinaryExpression(target) {
		initializer = skipDownwards(target.AsBinaryExpression().Right)
		target = skipDownwards(target.AsBinaryExpression().Left)
	}

	nested := ast.IsArrayLiteralExpression(target) || ast.IsObjectLiteralExpression(target)
	if nested || initializer != nil {
		id := valuePrereqs.PushToVar(value, "binding")
		if initializer != nil {
			valuePrereqs.Push(transformInitializer(state, id, initializer))
		}
		value = id
	}

	if nested {
		assert(luau.IsAnyIdentifier(value))
		prereqs.PushList(valuePrereqs.Statements)
		if ast.IsArrayLiteralExpression(target) {
			transformArrayAssignmentPattern(state, prereqs, target, value)
		} else {
			assert(ast.IsObjectLiteralExpression(target))
			transformObjectAssignmentPattern(state, prereqs, target, value)
		}
	} else {
		assert(
			ast.IsIdentifier(target) || ast.IsPropertyAccessExpression(target) || ast.IsElementAccessExpression(target),
			"Invalid destructuring assignment target: "+getKindName(target.Kind),
		)
		id := transformWritableExpression(state, prereqs, target, false)
		id = captureWritableAssignmentTarget(prereqs, id, getStatementsEffects(valuePrereqs.Statements), getEffects(value))
		prereqs.PushList(valuePrereqs.Statements)
		prereqs.Push(luau.Create(&luau.Assignment{Left: id, Operator: "=", Right: value}))
	}
}
