package transform

// Port of TSTransformer/nodes/expressions/transformConditionalExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformConditionalExpression is TSTransformer/nodes/expressions/transformConditionalExpression.ts transformConditionalExpression.
func transformConditionalExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	conditional := node.AsConditionalExpression()
	condition := transformExpression(state, prereqs, conditional.Condition)
	whenTruePrereqs := NewPrereqs()
	whenTrue := transformExpression(state, whenTruePrereqs, conditional.WhenTrue)
	whenFalsePrereqs := NewPrereqs()
	whenFalse := transformExpression(state, whenFalsePrereqs, conditional.WhenFalse)

	if isUsedAsStatement(node) {
		whenTruePrereqs.PushList(wrapExpressionStatement(whenTrue))
		whenFalsePrereqs.PushList(wrapExpressionStatement(whenFalse))
		prereqs.Push(
			luau.Create(&luau.IfStatement{
				Condition:  createTruthinessChecks(state, prereqs, condition, conditional.Condition),
				Statements: whenTruePrereqs.Statements,
				ElseBody:   whenFalsePrereqs.Statements,
			}),
		)
		return luau.NewNone()
	}

	if whenTruePrereqs.Statements.IsEmpty() && whenFalsePrereqs.Statements.IsEmpty() {
		return luau.Create(&luau.IfExpression{
			Condition:   createTruthinessChecks(state, prereqs, condition, conditional.Condition),
			Expression:  whenTrue,
			Alternative: whenFalse,
		})
	}

	tempId := luau.TempId("result")
	prereqs.Push(
		luau.Create(&luau.VariableDeclaration{
			Left:  tempId,
			Right: nil,
		}),
	)

	whenTruePrereqs.Push(
		luau.Create(&luau.Assignment{
			Left:     tempId,
			Operator: "=",
			Right:    whenTrue,
		}),
	)

	whenFalsePrereqs.Push(
		luau.Create(&luau.Assignment{
			Left:     tempId,
			Operator: "=",
			Right:    whenFalse,
		}),
	)

	prereqs.Push(
		luau.Create(&luau.IfStatement{
			Condition:  createTruthinessChecks(state, prereqs, condition, conditional.Condition),
			Statements: whenTruePrereqs.Statements,
			ElseBody:   whenFalsePrereqs.Statements,
		}),
	)

	return tempId
}
