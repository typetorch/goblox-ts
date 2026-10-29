package transform

// Port of TSTransformer/util/findConstructor.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// findConstructor is TSTransformer/util/findConstructor.ts findConstructor: the constructor of a
// class that has a body, or nil.
func findConstructor(node *ast.Node) *ast.Node {
	for _, member := range node.Members() {
		if ast.IsConstructorDeclaration(member) && member.Body() != nil {
			return member
		}
	}
	return nil
}
