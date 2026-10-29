package transform

// Port of TSTransformer/nodes/statements/transformTryStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func transformCatchClause(state *TransformState, node *ast.Node) *luau.FunctionExpression {
	catchClause := node.AsCatchClause()
	parameters := luau.MakeList()
	statements := luau.MakeList()

	if catchClause.VariableDeclaration != nil {
		parameters.Push(transformBindingName(state, catchClause.VariableDeclaration.Name(), statements))
	}

	statements.PushList(transformStatementList(state, catchClause.Block, catchClause.Block.AsBlock().Statements.Nodes, nil))

	catchFunction := luau.Create(&luau.FunctionExpression{
		Parameters:   parameters,
		HasDotDotDot: false,
		Statements:   statements,
	})

	return catchFunction
}

func transformIntoTryCall(
	state *TransformState,
	node *ast.Node,
	exitTypeId *luau.TemporaryIdentifier,
	returnsId *luau.TemporaryIdentifier,
	tryUses *TryUses,
) luau.Statement {
	tryStatement := node.AsTryStatement()
	var tryCallArgs []luau.Expression

	tryCallArgs = append(tryCallArgs, luau.Create(&luau.FunctionExpression{
		Parameters:   luau.MakeList(),
		HasDotDotDot: false,
		Statements:   transformStatementList(state, tryStatement.TryBlock, tryStatement.TryBlock.AsBlock().Statements.Nodes, nil),
	}))

	if tryStatement.CatchClause != nil {
		tryCallArgs = append(tryCallArgs, transformCatchClause(state, tryStatement.CatchClause))
	} else {
		assert(tryStatement.FinallyBlock != nil)
		tryCallArgs = append(tryCallArgs, luau.Nil())
	}

	if tryStatement.FinallyBlock != nil {
		tryCallArgs = append(tryCallArgs, luau.Create(&luau.FunctionExpression{
			Parameters:   luau.MakeList(),
			HasDotDotDot: false,
			Statements:   transformStatementList(state, tryStatement.FinallyBlock, tryStatement.FinallyBlock.AsBlock().Statements.Nodes, nil),
		}))
	}

	if !tryUses.UsesReturn && !tryUses.UsesBreak && !tryUses.UsesContinue {
		return luau.Create(&luau.CallStatement{
			Expression: luau.Call(state.TS(node, "try"), tryCallArgs...),
		})
	}

	return luau.Create(&luau.VariableDeclaration{
		Left:  luau.MakeList(exitTypeId, returnsId),
		Right: luau.Call(state.TS(node, "try"), tryCallArgs...),
	})
}

func createFlowControlCondition(
	state *TransformState,
	node *ast.Node,
	exitTypeId *luau.TemporaryIdentifier,
	flowControlConstant string,
) luau.Expression {
	return luau.Binary(exitTypeId, "==", state.TS(node, flowControlConstant))
}

type flowControlCase struct {
	condition  luau.Expression
	statements *luau.List
}

func collapseFlowControlCases(exitTypeId *luau.TemporaryIdentifier, cases []flowControlCase) *luau.List {
	assert(len(cases) > 0)

	nextStatements := luau.Create(&luau.IfStatement{
		Condition:  exitTypeId,
		Statements: cases[len(cases)-1].statements,
		ElseBody:   luau.MakeList(),
	})

	for i := len(cases) - 2; i >= 0; i-- {
		nextStatements = luau.Create(&luau.IfStatement{
			Condition:  cases[i].condition,
			Statements: cases[i].statements,
			ElseBody:   nextStatements,
		})
	}

	return luau.MakeList(nextStatements)
}

func transformFlowControl(
	state *TransformState,
	node *ast.Node,
	exitTypeId *luau.TemporaryIdentifier,
	returnsId *luau.TemporaryIdentifier,
	tryUses *TryUses,
) *luau.List {
	var flowControlCases []flowControlCase

	if !tryUses.UsesReturn && !tryUses.UsesBreak && !tryUses.UsesContinue {
		return luau.MakeList()
	}

	returnBlocked := isReturnBlockedByTryStatement(node.Parent)
	breakBlocked := isBreakBlockedByTryStatement(node.Parent)

	if tryUses.UsesReturn && returnBlocked {
		state.MarkTryUses("usesReturn")
	}
	if tryUses.UsesBreak && breakBlocked {
		state.MarkTryUses("usesBreak")
	}
	if tryUses.UsesContinue && breakBlocked {
		state.MarkTryUses("usesContinue")
	}

	if tryUses.UsesReturn {
		if returnBlocked {
			flowControlCases = append(flowControlCases, flowControlCase{
				condition: createFlowControlCondition(state, node, exitTypeId, "TRY_RETURN"),
				statements: luau.MakeList(
					luau.Create(&luau.ReturnStatement{
						Expression: luau.MakeList(exitTypeId, returnsId),
					}),
				),
			})
			if breakBlocked {
				return collapseFlowControlCases(exitTypeId, flowControlCases)
			}
		} else {
			flowControlCases = append(flowControlCases, flowControlCase{
				condition: createFlowControlCondition(state, node, exitTypeId, "TRY_RETURN"),
				statements: luau.MakeList(
					luau.Create(&luau.ReturnStatement{
						Expression: luau.Call(luau.Globals.Unpack, returnsId),
					}),
				),
			})
		}
	}

	if tryUses.UsesBreak || tryUses.UsesContinue {
		if breakBlocked {
			flowControlCases = append(flowControlCases, flowControlCase{
				condition: exitTypeId,
				statements: luau.MakeList(
					luau.Create(&luau.ReturnStatement{
						Expression: exitTypeId,
					}),
				),
			})
		} else {
			if tryUses.UsesBreak {
				flowControlCases = append(flowControlCases, flowControlCase{
					condition:  createFlowControlCondition(state, node, exitTypeId, "TRY_BREAK"),
					statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
				})
			}
			if tryUses.UsesContinue {
				flowControlCases = append(flowControlCases, flowControlCase{
					condition:  createFlowControlCondition(state, node, exitTypeId, "TRY_CONTINUE"),
					statements: luau.MakeList(luau.Create(&luau.ContinueStatement{})),
				})
			}
		}
	}

	return collapseFlowControlCases(exitTypeId, flowControlCases)
}

// transformTryStatement is TSTransformer/nodes/statements/transformTryStatement.ts transformTryStatement.
func transformTryStatement(state *TransformState, node *ast.Node) *luau.List {
	statements := luau.MakeList()
	exitTypeId := luau.TempId("exitType")
	returnsId := luau.TempId("returns")

	tryUses := state.PushTryUsesStack()
	statements.Push(transformIntoTryCall(state, node, exitTypeId, returnsId, tryUses))
	state.PopTryUsesStack()

	statements.PushList(transformFlowControl(state, node, exitTypeId, returnsId, tryUses))

	return statements
}
