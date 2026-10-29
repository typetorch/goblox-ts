package transform

// Port of TSTransformer/nodes/statements/transformBreakStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformBreakStatement is TSTransformer/nodes/statements/transformBreakStatement.ts transformBreakStatement.
func transformBreakStatement(state *TransformState, node *ast.Node) *luau.List {
	if label := node.AsBreakStatement().Label; label != nil {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoLabeledStatement.Create(label))
		return luau.MakeList()
	}

	if isBreakBlockedByTryStatement(node) {
		state.MarkTryUses("usesBreak")

		return luau.MakeList(
			luau.Create(&luau.ReturnStatement{
				Expression: state.TS(node, "TRY_BREAK"),
			}),
		)
	}

	return luau.MakeList(luau.Create(&luau.BreakStatement{}))
}
