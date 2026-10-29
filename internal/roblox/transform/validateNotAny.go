package transform

// Port of TSTransformer/util/validateNotAny.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
)

// validateNotAnyType is TSTransformer/util/validateNotAny.ts validateNotAnyType.
func validateNotAnyType(state *TransformState, node *ast.Node) {
	if ast.IsSpreadElement(node) {
		node = skipDownwards(node.Expression())
	}

	type_ := state.GetType(node)
	isAny := getOrSetDefault(state.MultiTransformState.IsAnyOrAnyArrayCache, type_, func() bool {
		checkedType := type_
		if isDefinitelyType(type_, isArrayType(state)) {
			// Array<T> -> T
			indexType := state.TypeChecker.GetIndexTypeOfType(type_, state.TypeChecker.GetNumberType())
			if indexType != nil {
				checkedType = indexType
			}
		}
		return isDefinitelyType(checkedType, isAnyType(state))
	})

	if isAny {
		// given a type like `a: { [index: string]: any }`, `a["b"]` will not have a symbol
		symbol := getOriginalSymbolOfNode(state.TypeChecker, node)
		if symbol != nil {
			if !state.MultiTransformState.IsReportedByNoAnyCache[symbol] {
				state.MultiTransformState.IsReportedByNoAnyCache[symbol] = true
				DiagnosticService.AddDiagnostic(diagnostics.Errors.NoAny.Create(node))
			}
		} else {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoAny.Create(node))
		}
	}
}
