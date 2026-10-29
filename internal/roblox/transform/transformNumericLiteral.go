package transform

// Port of TSTransformer/nodes/expressions/transformNumericLiteral.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
	"github.com/microsoft/typescript-go/internal/scanner"
)

// transformNumericLiteral is TSTransformer/nodes/expressions/transformNumericLiteral.ts transformNumericLiteral.
func transformNumericLiteral(state *TransformState, node *ast.Node) *luau.NumberLiteral {
	return luau.Create(&luau.NumberLiteral{
		Value: scanner.GetTextOfNode(node),
	})
}
