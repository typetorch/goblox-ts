package transform

// Port of TSTransformer/util/checkVariableHoist.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// checkVariableHoist is TSTransformer/util/checkVariableHoist.ts checkVariableHoist.
func checkVariableHoist(state *TransformState, node *ast.Node, symbol *ast.Symbol) {
	if _, ok := state.IsHoisted[symbol]; ok {
		return
	}

	statement := getAncestor(node, ast.IsStatement)
	assert(statement != nil)

	caseClause := statement.Parent
	if !ast.IsCaseClause(caseClause) {
		return
	}
	caseBlock := caseClause.Parent

	isUsedOutsideOfCaseClause := state.TypeChecker.EachSymbolReferenceInFile(
		node,
		ast.GetSourceFileOfNode(node),
		func(token *ast.Node) bool {
			return !isAncestorOf(caseClause, token)
		},
		caseBlock,
	)

	if isUsedOutsideOfCaseClause {
		state.HoistsByStatement[statement.Parent] = append(state.HoistsByStatement[statement.Parent], node)
		state.IsHoisted[symbol] = true
	}
}
