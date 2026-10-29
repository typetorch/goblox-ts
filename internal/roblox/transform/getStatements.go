package transform

// Port of TSTransformer/util/getStatements.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// getStatements is TSTransformer/util/getStatements.ts getStatements.
func getStatements(statement *ast.Node) []*ast.Node {
	if ast.IsBlock(statement) {
		return statement.AsBlock().Statements.Nodes
	}
	return []*ast.Node{statement}
}
