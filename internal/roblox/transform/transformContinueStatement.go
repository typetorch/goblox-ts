package transform

// Port of TSTransformer/nodes/statements/transformContinueStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformContinueStatement is TSTransformer/nodes/statements/transformContinueStatement.ts transformContinueStatement.
func transformContinueStatement(state *TransformState, node *ast.Node) *luau.List {
	if label := node.AsContinueStatement().Label; label != nil {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoLabeledStatement.Create(label))
		return luau.MakeList()
	}

	if isBreakBlockedByTryStatement(node) {
		state.MarkTryUses("usesContinue")

		return luau.MakeList(
			luau.Create(&luau.ReturnStatement{
				Expression: state.TS(node, "TRY_CONTINUE"),
			}),
		)
	}

	return luau.MakeList(luau.Create(&luau.ContinueStatement{}))
}
