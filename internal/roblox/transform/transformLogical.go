package transform

// Port of TSTransformer/nodes/transformLogical.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

type logicalChainItem struct {
	node       *ast.Node
	type_      *checker.Type
	expression luau.Expression
	statements *luau.List
	inline     bool
}

type logicalConditionBuilder func(prereqs *Prereqs, conditionId *luau.TemporaryIdentifier, node *ast.Node) luau.Expression

// flattenByOperator splits `node` recursively by binary operator `operatorKind` into a []*ast.Node
//
// i.e. using `&&` as our operator, `a && b && c` -> `[a, b, c]`
func flattenByOperator(node *ast.Node, operatorKind ast.Kind) []*ast.Node {
	var result []*ast.Node
	for ast.IsBinaryExpression(node) && node.AsBinaryExpression().OperatorToken.Kind == operatorKind {
		result = append([]*ast.Node{node.AsBinaryExpression().Right}, result...)
		node = node.AsBinaryExpression().Left
	}
	result = append([]*ast.Node{node}, result...)
	return result
}

// getLogicalChain builds a []*logicalChainItem using `splitBinaryChain` and
// captures all of the prerequisite statements for each expression
func getLogicalChain(
	state *TransformState,
	binaryExp *ast.Node,
	binaryOperatorKind ast.Kind,
	enableInlining bool,
) []*logicalChainItem {
	array := flattenByOperator(binaryExp, binaryOperatorKind)
	result := make([]*logicalChainItem, len(array))
	for index, node := range array {
		type_ := state.GetType(node)
		expressionPrereqs := NewPrereqs()
		expression := transformExpression(state, expressionPrereqs, node)
		inline := false
		if enableInlining {
			willWrap := index < len(array)-1 && willCreateTruthinessChecks(type_)
			inline = expressionPrereqs.Statements.IsEmpty() && !willWrap
		}
		result[index] = &logicalChainItem{
			node:       node,
			type_:      type_,
			expression: expression,
			statements: expressionPrereqs.Statements,
			inline:     inline,
		}
	}
	return result
}

// buildLogicalChainPrereqs recursively creates prerequisite statements for non-inlined expressions in a
// logical chain
func buildLogicalChainPrereqs(
	state *TransformState,
	prereqs *Prereqs,
	chain []*logicalChainItem,
	conditionId *luau.TemporaryIdentifier,
	buildCondition logicalConditionBuilder,
	index int,
) {
	expInfo := chain[index]
	prereqs.PushList(expInfo.statements)
	if index == 0 {
		prereqs.Push(luau.Create(&luau.VariableDeclaration{
			Left:  conditionId,
			Right: expInfo.expression,
		}))
	} else {
		prereqs.Push(luau.Create(&luau.Assignment{
			Left:     conditionId,
			Operator: "=",
			Right:    expInfo.expression,
		}))
	}
	if index+1 < len(chain) {
		condition := buildCondition(prereqs, conditionId, expInfo.node)
		branchPrereqs := NewPrereqs()
		buildLogicalChainPrereqs(state, branchPrereqs, chain, conditionId, buildCondition, index+1)
		prereqs.Push(luau.Create(&luau.IfStatement{
			Condition:  condition,
			Statements: branchPrereqs.Statements,
			ElseBody:   luau.MakeList(),
		}))
	}
}

// mergeInlineExpressions merges any neighboring inline logicalChainItems together, builds new
// expressions using `binaryOperator`. It returns the shortened chain (TypeScript splices in place).
//
// i.e. using `and` as our operator,
//
//	[{ expression: a, inline: true }, { expression: b, inline: true }, { expression: c, inline: false }]
//
// turns into
//
//	[{ expression: a and b, inline: true }, { expression: c, inline: false }]
func mergeInlineExpressions(chain []*logicalChainItem, binaryOperator luau.BinaryOperator) []*logicalChainItem {
	for i := 0; i < len(chain); i++ {
		info := chain[i]
		if info.inline {
			exps := []luau.Expression{info.expression}
			j := i + 1
			for j < len(chain) && chain[j].inline {
				exps = append(exps, chain[j].expression)
				chain = append(chain[:j], chain[j+1:]...)
			}
			info.expression = binaryExpressionChain(exps, binaryOperator)
		}
	}
	return chain
}

// buildInlineConditionExpression solves the simple case for `&&` / `||` where items can be inlined into
// `and` / `or` expressions.
func buildInlineConditionExpression(
	state *TransformState,
	prereqs *Prereqs,
	node *ast.Node,
	tsBinaryOperator ast.Kind,
	luaBinaryOperator luau.BinaryOperator,
	buildCondition logicalConditionBuilder,
) luau.Expression {
	chain := getLogicalChain(state, node, tsBinaryOperator, true)

	chain = mergeInlineExpressions(chain, luaBinaryOperator)

	// single inline at the end, no temp variable needed
	if len(chain) == 1 && chain[0].inline {
		return chain[0].expression
	}

	conditionId := luau.TempId("condition")
	buildLogicalChainPrereqs(state, prereqs, chain, conditionId, buildCondition, 0)
	return conditionId
}

// transformLogical is TSTransformer/nodes/transformLogical.ts transformLogical.
func transformLogical(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	operatorKind := node.AsBinaryExpression().OperatorToken.Kind
	if operatorKind == ast.KindAmpersandAmpersandToken {
		return buildInlineConditionExpression(
			state,
			prereqs,
			node,
			operatorKind,
			"and",
			func(prereqs *Prereqs, conditionId *luau.TemporaryIdentifier, node *ast.Node) luau.Expression {
				return createTruthinessChecks(state, prereqs, conditionId, node)
			},
		)
	} else if operatorKind == ast.KindBarBarToken {
		return buildInlineConditionExpression(
			state,
			prereqs,
			node,
			operatorKind,
			"or",
			func(prereqs *Prereqs, conditionId *luau.TemporaryIdentifier, node *ast.Node) luau.Expression {
				return luau.Unary("not", createTruthinessChecks(state, prereqs, conditionId, node))
			},
		)
	} else if operatorKind == ast.KindQuestionQuestionToken {
		conditionBuilder := func(prereqs *Prereqs, conditionId *luau.TemporaryIdentifier, node *ast.Node) luau.Expression {
			return luau.Binary(conditionId, "==", luau.Nil())
		}
		if !isPossiblyType(state.GetType(node), isBooleanLiteralType(state, false)) {
			return buildInlineConditionExpression(
				state,
				prereqs,
				node,
				operatorKind,
				"or",
				conditionBuilder,
			)
		}
		chain := getLogicalChain(state, node, ast.KindQuestionQuestionToken, false)
		conditionId := luau.TempId("condition")
		buildLogicalChainPrereqs(state, prereqs, chain, conditionId, conditionBuilder, 0)
		return conditionId
	}
	assert(false, "Operator not implemented: "+getKindName(operatorKind))
	return nil
}
