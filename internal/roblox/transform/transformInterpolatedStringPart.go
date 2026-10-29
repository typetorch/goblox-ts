package transform

// Port of TSTransformer/nodes/transformInterpolatedStringPart.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformInterpolatedStringPart is TSTransformer/nodes/transformInterpolatedStringPart.ts transformInterpolatedStringPart.
func transformInterpolatedStringPart(node *ast.Node) *luau.InterpolatedStringPart {
	return luau.Create(&luau.InterpolatedStringPart{Text: createStringFromLiteral(node)})
}
