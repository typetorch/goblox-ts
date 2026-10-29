package transform

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// isOmittedBindingElement is TypeScript's ts.isOmittedExpression for an element of an ArrayBindingPattern.
// typescript-go parses a hole (`const [, b] = t`) as a BindingElement without a name, not as an
// OmittedExpression; ArrayLiteralExpression holes are still OmittedExpressions.
func isOmittedBindingElement(element *ast.Node) bool {
	return ast.IsOmittedExpression(element) || (ast.IsBindingElement(element) && element.Name() == nil)
}
