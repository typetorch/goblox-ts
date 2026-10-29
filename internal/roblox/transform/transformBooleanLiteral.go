package transform

// Port of TSTransformer/nodes/expressions/transformBooleanLiteral.ts.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformTrueKeyword is TSTransformer/nodes/expressions/transformBooleanLiteral.ts transformTrueKeyword.
func transformTrueKeyword() *luau.TrueLiteral {
	return luau.Create(&luau.TrueLiteral{})
}

// transformFalseKeyword is TSTransformer/nodes/expressions/transformBooleanLiteral.ts transformFalseKeyword.
func transformFalseKeyword() *luau.FalseLiteral {
	return luau.Create(&luau.FalseLiteral{})
}
