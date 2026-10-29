package transform

// Small private helpers (no roblox-ts counterpart) for the call, access and literal transforms.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// getAllSuperTypeNodes is TypeScript's ts.getAllSuperTypeNodes: the nodes in the `extends` and
// `implements` clauses of a class or interface.
func getAllSuperTypeNodes(node *ast.Node) []*ast.Node {
	if ast.IsInterfaceDeclaration(node) {
		return ast.GetHeritageElements(node, ast.KindExtendsKeyword)
	}
	if ast.IsClassLike(node) {
		var result []*ast.Node
		if base := ast.GetClassExtendsHeritageElement(node); base != nil {
			result = append(result, base)
		}
		return append(result, ast.GetImplementsHeritageClauseElements(node)...)
	}
	return nil
}
