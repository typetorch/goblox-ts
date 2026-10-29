package transform

// Port of TSTransformer/util/expressionChain.ts.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// binaryExpressionChain is TSTransformer/util/expressionChain.ts binaryExpressionChain.
func binaryExpressionChain(expressions []luau.Expression, operator luau.BinaryOperator) luau.Expression {
	acc := expressions[0]
	for _, current := range expressions[1:] {
		acc = luau.Binary(acc, operator, current)
	}
	return acc
}

// propertyAccessExpressionChain is TSTransformer/util/expressionChain.ts propertyAccessExpressionChain.
func propertyAccessExpressionChain(expression luau.Expression, names []string) luau.IndexableExpression {
	acc := convertToIndexableExpression(expression)
	for _, current := range names {
		acc = luau.Property(acc, current)
	}
	return acc
}
