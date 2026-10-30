package transform

// Port of TSTransformer/nodes/statements/transformExportAssignment.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func transformExportEquals(state *TransformState, node *ast.Node) *luau.List {
	state.HasExportEquals = true

	statements := luau.MakeList()
	prereqs := NewPrereqs()
	var statement luau.Statement

	sourceFile := ast.GetSourceFileOfNode(node)
	sourceStatements := sourceFile.Statements.Nodes
	finalStatement := sourceStatements[len(sourceStatements)-1]
	if finalStatement == node {
		statement = luau.Create(&luau.ReturnStatement{
			Expression: transformExpression(state, prereqs, node.Expression()),
		})
	} else {
		statement = luau.Create(&luau.VariableDeclaration{
			Left:  state.GetModuleIdFromNode(node),
			Right: transformExpression(state, prereqs, node.Expression()),
		})
	}

	statements.PushList(prereqs.Statements)
	statements.Push(statement)
	return statements
}

func transformExportDefault(state *TransformState, node *ast.Node) *luau.List {
	statements := luau.MakeList()

	expressionPrereqs := NewPrereqs()
	expression := transformExpression(state, expressionPrereqs, node.Expression())
	statements.PushList(expressionPrereqs.Statements)
	statements.Push(luau.Create(&luau.VariableDeclaration{
		Left:  luau.Id("default"),
		Right: expression,
	}))

	return statements
}

// transformExportAssignment is TSTransformer/nodes/statements/transformExportAssignment.ts transformExportAssignment.
func transformExportAssignment(state *TransformState, node *ast.Node) *luau.List {
	symbol := state.TypeChecker.GetSymbolAtLocation(node.Expression())
	if symbol != nil && isSymbolMutable(state, symbol) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoExportAssignmentLet.Create(node))
	}

	if symbol != nil &&
		(!isSymbolOfValue(SkipAliases(state.TypeChecker, symbol)) ||
			(!state.CompilerOptions.VerbatimModuleSyntax.IsTrue() && state.TypeChecker.GetTypeOnlyAliasDeclaration(symbol) != nil)) {
		return luau.MakeList()
	}

	if node.AsExportAssignment().IsExportEquals {
		return transformExportEquals(state, node)
	} else {
		return transformExportDefault(state, node)
	}
}
