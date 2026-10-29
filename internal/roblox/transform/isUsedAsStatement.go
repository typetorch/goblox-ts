package transform

// Port of TSTransformer/util/isUsedAsStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// isUsedAsStatement is TSTransformer/util/isUsedAsStatement.ts isUsedAsStatement.
func isUsedAsStatement(expression *ast.Node) bool {
	child := skipUpwards(expression)
	parent := child.Parent

	if ast.IsExpressionStatement(parent) {
		return true
	}

	// if part of for statement definition, except if used as the condition
	if ast.IsForStatement(parent) && parent.AsForStatement().Condition != child {
		return true
	}

	if ast.IsDeleteExpression(parent) && isUsedAsStatement(parent) {
		return true
	}

	return false
}
