package transform

// Port of TSTransformer/nodes/expressions/transformStringLiteral.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
	"github.com/microsoft/typescript-go/internal/scanner"
)

// transformStringLiteral is TSTransformer/nodes/expressions/transformStringLiteral.ts transformStringLiteral.
func transformStringLiteral(state *TransformState, node *ast.Node) *luau.StringLiteral {
	quote := `"`
	if scanner.GetTextOfNode(node)[0] == '\'' {
		quote = "'"
	}
	return luau.String(createStringFromLiteral(node), quote)
}
