package transform

import "github.com/microsoft/typescript-go/internal/ast"

// Port of TSTransformer/util/traversal.ts.

func isAncestorOf(ancestor *ast.Node, node *ast.Node) bool {
	for ; node != nil; node = node.Parent {
		if ancestor == node {
			return true
		}
	}
	return false
}

func isSkippedWrapper(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindNonNullExpression, ast.KindParenthesizedExpression, ast.KindAsExpression,
		ast.KindTypeAssertionExpression, ast.KindSatisfiesExpression:
		return true
	}
	return false
}

// skipDownwards skips parentheses, `!`, `as`, `<T>` and `satisfies` around an expression.
func skipDownwards(node *ast.Node) *ast.Node {
	for isSkippedWrapper(node) {
		node = node.Expression()
	}
	return node
}

// skipUpwards returns the outermost parenthesized/asserted expression that wraps node.
func skipUpwards(node *ast.Node) *ast.Node {
	parent := node.Parent
	for parent != nil && isSkippedWrapper(parent) {
		node = parent
		parent = node.Parent
	}
	return node
}

func getAncestor(node *ast.Node, check func(*ast.Node) bool) *ast.Node {
	current := node
	for current != nil && !check(current) {
		current = current.Parent
	}
	return current
}

func isSourceFileOrModuleDeclaration(node *ast.Node) bool {
	return ast.IsSourceFile(node) || ast.IsModuleDeclaration(node)
}

func getModuleAncestor(node *ast.Node) *ast.Node {
	return getAncestor(node, isSourceFileOrModuleDeclaration)
}

// isBlockLike is TSTransformer/typeGuards.ts isBlockLike.
func isBlockLike(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindSourceFile, ast.KindBlock, ast.KindModuleBlock, ast.KindCaseClause, ast.KindDefaultClause:
		return true
	}
	return false
}

// blockStatements returns the statements of a block-like node.
func blockStatements(node *ast.Node) []*ast.Node {
	switch node.Kind {
	case ast.KindSourceFile:
		return node.AsSourceFile().Statements.Nodes
	case ast.KindBlock:
		return node.AsBlock().Statements.Nodes
	case ast.KindModuleBlock:
		return node.AsModuleBlock().Statements.Nodes
	case ast.KindCaseClause, ast.KindDefaultClause:
		return node.AsCaseOrDefaultClause().Statements.Nodes
	}
	return nil
}

func hasDeclareModifier(node *ast.Node) bool {
	return ast.CanHaveModifiers(node) && ast.HasSyntacticModifier(node, ast.ModifierFlagsAmbient)
}
