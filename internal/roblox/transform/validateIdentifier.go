package transform

// Port of TSTransformer/util/validateIdentifier.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// validateIdentifier is TSTransformer/util/validateIdentifier.ts validateIdentifier.
func validateIdentifier(node *ast.Node) {
	if !luau.IsValidIdentifier(node.Text()) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoInvalidIdentifier.Create(node))
	} else if luau.IsReservedIdentifier(node.Text()) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoReservedIdentifier.Create(node))
	}
}
