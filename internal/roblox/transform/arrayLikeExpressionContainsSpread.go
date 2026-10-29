package transform

// Port of TSTransformer/util/arrayLikeExpressionContainsSpread.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// arrayLikeExpressionContainsSpread is TSTransformer/util/arrayLikeExpressionContainsSpread.ts arrayLikeExpressionContainsSpread.
func arrayLikeExpressionContainsSpread(exp *ast.Node) bool {
	for _, element := range exp.Elements() {
		if (ast.IsBindingElement(element) && element.AsBindingElement().DotDotDotToken != nil) || ast.IsSpreadElement(element) {
			return true
		}
	}
	return false
}
