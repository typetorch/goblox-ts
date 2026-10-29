package transform

// Port of TSTransformer/nodes/expressions/transformSuperKeyword.ts.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformSuperKeyword is TSTransformer/nodes/expressions/transformSuperKeyword.ts transformSuperKeyword.
func transformSuperKeyword() *luau.Identifier {
	return luau.Globals.Super
}
