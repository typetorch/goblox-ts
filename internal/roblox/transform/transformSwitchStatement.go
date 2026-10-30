package transform

// Port of TSTransformer/nodes/statements/transformSwitchStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func canCompareIdentifierDirectly(state *TransformState, node *ast.Node, expression luau.Expression) bool {
	switchStatement := node.AsSwitchStatement()
	identifier := skipDownwards(switchStatement.Expression)
	if !ast.IsIdentifier(identifier) || !luau.IsIdentifier(expression) {
		return false
	}

	// declarations shared by case bodies can be hoisted ahead of their comparisons
	if state.TypeChecker.ResolveName(identifier.Text(), switchStatement.CaseBlock, ast.SymbolFlagsValue, false) !=
		state.TypeChecker.GetSymbolAtLocation(identifier) {
		return false
	}

	// literal labels cannot mutate the operand, and fallthrough skips later comparisons
	for _, clause := range switchStatement.CaseBlock.AsCaseBlock().Clauses.Nodes {
		if ast.IsDefaultClause(clause) {
			continue
		}

		label := skipDownwards(clause.AsCaseOrDefaultClause().Expression)
		if !(ast.IsStringLiteralLike(label) ||
			ast.IsNumericLiteral(label) ||
			label.Kind == ast.KindTrueKeyword ||
			label.Kind == ast.KindFalseKeyword) {
			return false
		}
	}
	return true
}

func transformCaseClauseExpression(
	state *TransformState,
	caseClauseExpression *ast.Node,
	switchExpression luau.Expression,
	fallThroughFlagId *luau.TemporaryIdentifier,
	canFallThroughTo bool,
) (condition luau.Expression, prereqStatements *luau.List) {
	caseValueId := luau.TempId("caseValue")
	casePrereqs := NewPrereqs()
	expression := transformExpression(state, casePrereqs, caseClauseExpression)
	prereqStatements = luau.MakeList()
	prereqStatements.PushList(casePrereqs.Statements)
	caseValueMightMutate := expressionMightMutate(state, expression, caseClauseExpression)

	if caseValueMightMutate {
		prereqStatements.Push(luau.Create(&luau.VariableDeclaration{
			Left:  caseValueId,
			Right: expression,
		}))
	}

	var caseValue luau.Expression = expression
	if caseValueMightMutate {
		caseValue = caseValueId
	}
	condition = luau.Binary(switchExpression, "==", caseValue)

	if canFallThroughTo {
		if !prereqStatements.IsEmpty() {
			noFallThroughCondition := luau.Unary("not", fallThroughFlagId)

			prereqStatements.Push(luau.Create(&luau.Assignment{
				Left:     fallThroughFlagId,
				Operator: "=",
				Right:    condition,
			}))

			prereqStatements = luau.MakeList(
				luau.Create(&luau.IfStatement{
					Condition:  noFallThroughCondition,
					Statements: prereqStatements,
					ElseBody:   luau.MakeList(),
				}),
			)

			condition = fallThroughFlagId
		} else {
			condition = luau.Binary(fallThroughFlagId, "or", condition)
		}
	}

	return condition, prereqStatements
}

func transformCaseClause(
	state *TransformState,
	node *ast.Node,
	switchExpression luau.Expression,
	fallThroughFlagId *luau.TemporaryIdentifier,
	canFallThroughTo bool,
	shouldUpdateFallThroughFlag bool,
) (canFallThroughFrom bool, prereqs *luau.List, clauseStatements *luau.List) {
	clause := node.AsCaseOrDefaultClause()
	condition, prereqStatements := transformCaseClauseExpression(
		state,
		clause.Expression,
		switchExpression,
		fallThroughFlagId,
		canFallThroughTo,
	)

	var nonEmptyStatements []*ast.Node
	for _, v := range clause.Statements.Nodes {
		if !ast.IsEmptyStatement(v) {
			nonEmptyStatements = append(nonEmptyStatements, v)
		}
	}
	var statements *luau.List
	if len(nonEmptyStatements) == 1 && ast.IsBlock(nonEmptyStatements[0]) {
		firstStatement := nonEmptyStatements[0]
		statements = transformStatementList(state, firstStatement, firstStatement.AsBlock().Statements.Nodes, nil)
	} else {
		statements = transformStatementList(state, node, clause.Statements.Nodes, nil)
	}

	canFallThroughFrom = statements.Tail == nil || !luau.IsFinalStatement(statements.Tail.Value)
	if canFallThroughFrom && shouldUpdateFallThroughFlag {
		statements.Push(luau.Create(&luau.Assignment{
			Left:     fallThroughFlagId,
			Operator: "=",
			Right:    luau.Bool(true),
		}))
	}

	clauseStatements = luau.MakeList()

	if hoistDeclaration := createHoistDeclaration(state, node); hoistDeclaration != nil {
		clauseStatements.Push(hoistDeclaration)
	}

	clauseStatements.Push(luau.Create(&luau.IfStatement{
		Condition:  condition,
		Statements: statements,
		ElseBody:   luau.MakeList(),
	}))

	return canFallThroughFrom, prereqStatements, clauseStatements
}

// transformSwitchStatement is TSTransformer/nodes/statements/transformSwitchStatement.ts transformSwitchStatement.
func transformSwitchStatement(state *TransformState, node *ast.Node) *luau.List {
	switchStatement := node.AsSwitchStatement()
	clauses := switchStatement.CaseBlock.AsCaseBlock().Clauses.Nodes

	result := luau.MakeList()
	prereqs := NewPrereqs()
	switchExpression := transformExpression(state, prereqs, switchStatement.Expression)
	needsCapture := expressionMightMutate(state, switchExpression, switchStatement.Expression) &&
		!canCompareIdentifierDirectly(state, node, switchExpression)
	var expression luau.Expression
	if needsCapture {
		expression = prereqs.PushToVar(switchExpression, "exp")
	} else {
		expression = prereqs.PushToVarIfComplex(switchExpression, "exp")
	}
	result.PushList(prereqs.Statements)
	fallThroughFlagId := luau.TempId("fallthrough")

	isFallThroughFlagNeeded := false

	statements := luau.MakeList()
	canFallThroughTo := false
	for i := range clauses {
		caseClauseNode := clauses[i]

		if ast.IsCaseClause(caseClauseNode) {
			shouldUpdateFallThroughFlag := i < len(clauses)-1 && ast.IsCaseClause(clauses[i+1])
			canFallThroughFrom, prereqStatements, clauseStatements := transformCaseClause(
				state,
				caseClauseNode,
				expression,
				fallThroughFlagId,
				canFallThroughTo,
				shouldUpdateFallThroughFlag,
			)

			statements.PushList(prereqStatements)
			statements.PushList(clauseStatements)

			canFallThroughTo = canFallThroughFrom

			if canFallThroughFrom {
				isFallThroughFlagNeeded = true
			}
		} else {
			statements.PushList(transformStatementList(state, caseClauseNode, caseClauseNode.AsCaseOrDefaultClause().Statements.Nodes, nil))
			break
		}
	}

	if isFallThroughFlagNeeded {
		statements.Unshift(luau.Create(&luau.VariableDeclaration{
			Left:  fallThroughFlagId,
			Right: luau.Bool(false),
		}))
	}

	result.Push(luau.Create(&luau.RepeatStatement{
		Condition:  luau.Bool(true),
		Statements: statements,
	}))
	return result
}
