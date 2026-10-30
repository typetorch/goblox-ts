package transform

// Port of TSTransformer/nodes/expressions/transformNumericLiteral.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
	"github.com/microsoft/typescript-go/internal/scanner"
)

// transformNumericLiteral is TSTransformer/nodes/expressions/transformNumericLiteral.ts transformNumericLiteral.
func transformNumericLiteral(state *TransformState, node *ast.Node) *luau.NumberLiteral {
	// see ProjectData.TransformersRan: TypeScript's printer replaces a literal with separators by its value
	if state.transformersRan() && node.AsNumericLiteral().TokenFlags&ast.TokenFlagsContainsSeparator != 0 {
		return luau.Create(&luau.NumberLiteral{
			Value: node.Text(),
		})
	}
	return luau.Create(&luau.NumberLiteral{
		Value: scanner.GetTextOfNode(node),
	})
}
