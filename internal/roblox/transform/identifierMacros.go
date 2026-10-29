package transform

// Port of TSTransformer/macros/identifierMacros.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

var IDENTIFIER_MACROS = MacroList[IdentifierMacro]{
	"Promise": func(state *TransformState, node *ast.Node) luau.Expression { return state.TS(node, "Promise") },
}
