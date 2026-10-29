package transform

// Port of TSTransformer/util/valueToIdStr.ts.

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func expressionToStr(expression luau.Expression) (string, bool) {
	// X -> "X"
	if identifier, ok := expression.(*luau.Identifier); ok {
		return identifier.Name, true
	}
	// A.B -> "B"
	if propertyAccess, ok := expression.(*luau.PropertyAccessExpression); ok {
		return propertyAccess.Name, true
	}
	if call, ok := expression.(*luau.CallExpression); ok {
		// X.new() -> "X"
		if propertyAccess, ok := call.Expression.(*luau.PropertyAccessExpression); ok && propertyAccess.Name == "new" {
			return expressionToStr(propertyAccess.Expression)
		}
	}
	return "", false
}

func uncapitalizeFirstLetter(str string) string {
	if str == "" {
		return str
	}
	return strings.ToLower(str[:1]) + str[1:]
}

// valueToIdStr is TSTransformer/util/valueToIdStr.ts valueToIdStr.
func valueToIdStr(value luau.Expression) string {
	valueStr, ok := expressionToStr(value)
	if ok && luau.IsValidIdentifier(valueStr) {
		return uncapitalizeFirstLetter(valueStr)
	}
	return ""
}
