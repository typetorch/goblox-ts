package transform

// Port of TSTransformer/util/convertToIndexableExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// convertToIndexableExpression is TSTransformer/util/convertToIndexableExpression.ts convertToIndexableExpression.
func convertToIndexableExpression(expression luau.Expression) luau.IndexableExpression {
	if luau.IsIndexableExpression(expression) {
		return expression
	}
	return luau.Create(&luau.ParenthesizedExpression{Expression: expression})
}
