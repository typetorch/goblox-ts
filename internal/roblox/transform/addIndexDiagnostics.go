package transform

// Port of TSTransformer/util/addIndexDiagnostics.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
)

func isLengthStringLiteralType(t *checker.Type) bool {
	return t.IsStringLiteral() && typeStringValue(t) == "length"
}

func isLengthPropertyName(state *TransformState, node *ast.Node) bool {
	if ast.IsComputedPropertyName(node) {
		argumentType := state.GetType(node.Expression())
		return isPossiblyType(argumentType, isLengthStringLiteralType)
	}

	return (ast.IsIdentifier(node) || ast.IsStringLiteral(node)) && node.Text() == "length"
}

// addIndexDiagnostics is TSTransformer/util/addIndexDiagnostics.ts addIndexDiagnostics.
func addIndexDiagnostics(state *TransformState, node *ast.Node, memberType *checker.Type, receiverType *checker.Type) {
	symbol := getFirstDefinedSymbol(state, memberType)
	if (symbol != nil && state.Services.MacroManager.GetPropertyCallMacro(symbol) != nil) ||
		(!isValidMethodIndexWithoutCall(state, skipUpwards(node)) && isMethod(state, node)) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoIndexWithoutCall.Create(node))
	}

	if ast.IsPrototypeAccess(node) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoPrototype.Create(node))
	}

	if isPossiblyType(receiverType, checker.IsTupleType) {
		if ast.IsPropertyAccessExpression(node) && node.Name().Text() == "length" {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoLengthIndexInTuples.Create(node))
		}

		if ast.IsPropertyName(node) && isLengthPropertyName(state, node) {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoLengthIndexInTuples.Create(node))
		}

		if ast.IsElementAccessExpression(node) {
			argumentType := state.GetType(node.AsElementAccessExpression().ArgumentExpression)
			if isPossiblyType(argumentType, isLengthStringLiteralType) {
				DiagnosticService.AddDiagnostic(diagnostics.Errors.NoLengthIndexInTuples.Create(node))
			}
		}
	}
}
