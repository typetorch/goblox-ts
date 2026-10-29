package transform

// Port of TSTransformer/util/createGetService.ts.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// createGetService returns a LuauAST node equivalent for `game:GetService(serviceName)`.
func createGetService(serviceName string) *luau.MethodCallExpression {
	return luau.Create(&luau.MethodCallExpression{
		Expression: luau.Globals.Game,
		Name:       "GetService",
		Args:       luau.MakeList(luau.String(serviceName)),
	})
}
