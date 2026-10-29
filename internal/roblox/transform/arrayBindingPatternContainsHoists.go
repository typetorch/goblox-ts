package transform

// Port of TSTransformer/util/arrayBindingPatternContainsHoists.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// arrayBindingPatternContainsHoists is TSTransformer/util/arrayBindingPatternContainsHoists.ts arrayBindingPatternContainsHoists.
func arrayBindingPatternContainsHoists(state *TransformState, arrayBindingPattern *ast.Node) bool {
	for _, element := range arrayBindingPattern.Elements() {
		// If it's not a ts.BindingElement, it must be a ts.OmittedExpression. In that case, doesn't create a variable.
		// For non-identifiers, element.name must be a nested array/object binding pattern.
		// For those cases, the hoisting logic is handled elsewhere and the variable here will be a tempId.
		if ast.IsBindingElement(element) && ast.IsIdentifier(element.Name()) {
			symbol := state.TypeChecker.GetSymbolAtLocation(element.Name())
			assert(symbol != nil)
			// isHoisted is marked inside checkVariableHoist
			checkVariableHoist(state, element.Name(), symbol)
			if state.IsHoisted[symbol] {
				return true
			}
		}
	}
	return false
}
