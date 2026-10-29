package transform

// Port of TSTransformer/typeGuards.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
)

func isBlockLike(node *ast.Node) bool {
	return node.Kind == ast.KindSourceFile ||
		node.Kind == ast.KindBlock ||
		node.Kind == ast.KindModuleBlock ||
		node.Kind == ast.KindCaseClause ||
		node.Kind == ast.KindDefaultClause
}

func isUnaryAssignmentOperator(operator ast.Kind) bool {
	return operator == ast.KindPlusPlusToken || operator == ast.KindMinusMinusToken
}

func isTemplateLiteralType(t *checker.Type) bool {
	return t.Flags()&checker.TypeFlagsTemplateLiteral != 0
}

func isNamespace(node *ast.Node) bool {
	// typescript-go has no NodeFlags.Namespace; the keyword is kept on nested declarations too
	return ast.IsModuleDeclaration(node) && node.AsModuleDeclaration().Keyword == ast.KindNamespaceKeyword
}
