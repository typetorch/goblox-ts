package transform

// Port of TSTransformer/nodes/expressions/transformNoSubstitutionTemplateLiteral.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// backtick string literals without interpolation expressions should be preserved
// as they still are valid in luau
func transformNoSubstitutionTemplateLiteral(state *TransformState, node *ast.Node) *luau.InterpolatedString {
	return luau.Create(&luau.InterpolatedString{
		Parts: luau.MakeList(transformInterpolatedStringPart(node)),
	})
}
