package transform

// Port of TSTransformer/util/isBlockedByTryStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// isReturnBlockedByTryStatement is TSTransformer/util/isBlockedByTryStatement.ts isReturnBlockedByTryStatement.
func isReturnBlockedByTryStatement(node *ast.Node) bool {
	ancestor := ast.FindAncestor(node, func(ancestor *ast.Node) bool {
		return ast.IsTryStatement(ancestor) || ast.IsFunctionLikeDeclaration(ancestor)
	})
	return ancestor != nil && ast.IsTryStatement(ancestor)
}

// isBreakBlockedByTryStatement is TSTransformer/util/isBlockedByTryStatement.ts isBreakBlockedByTryStatement.
func isBreakBlockedByTryStatement(node *ast.Node) bool {
	ancestor := ast.FindAncestor(node, func(ancestor *ast.Node) bool {
		return ast.IsTryStatement(ancestor) || ast.IsIterationStatement(ancestor, false) || ast.IsSwitchStatement(ancestor)
	})
	return ancestor != nil && ast.IsTryStatement(ancestor)
}
