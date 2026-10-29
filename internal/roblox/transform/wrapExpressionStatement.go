package transform

// Port of TSTransformer/util/wrapExpressionStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// wrapExpressionStatement is TSTransformer/util/wrapExpressionStatement.ts wrapExpressionStatement.
func wrapExpressionStatement(node luau.Expression) *luau.List {
	if luau.IsTemporaryIdentifier(node) || luau.IsNone(node) {
		return luau.MakeList()
	} else if luau.IsCall(node) {
		return luau.MakeList(luau.Create(&luau.CallStatement{Expression: node}))
	} else {
		return luau.MakeList(
			luau.Create(&luau.VariableDeclaration{
				Left:  luau.TempId(),
				Right: node,
			}),
		)
	}
}
