package transform

// Port of TSTransformer/util/getDeclaredVariables.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

func getDeclaredVariablesFromBindingName(node *ast.Node, list *[]*ast.Node) {
	if ast.IsIdentifier(node) {
		*list = append(*list, node)
	} else if ast.IsObjectBindingPattern(node) {
		for _, element := range node.AsBindingPattern().Elements.Nodes {
			getDeclaredVariablesFromBindingName(element.Name(), list)
		}
	} else {
		for _, element := range node.AsBindingPattern().Elements.Nodes {
			// typescript-go represents an omitted array binding element as a BindingElement without a name
			if !ast.IsOmittedExpression(element) && element.Name() != nil {
				getDeclaredVariablesFromBindingName(element.Name(), list)
			}
		}
	}
}

// getDeclaredVariables is TSTransformer/util/getDeclaredVariables.ts getDeclaredVariables.
func getDeclaredVariables(node *ast.Node) []*ast.Node {
	list := []*ast.Node{}
	if ast.IsVariableDeclarationList(node) {
		for _, declaration := range node.AsVariableDeclarationList().Declarations.Nodes {
			getDeclaredVariablesFromBindingName(declaration.Name(), &list)
		}
	} else {
		getDeclaredVariablesFromBindingName(node.Name(), &list)
	}
	return list
}
