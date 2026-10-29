package transform

// Port of TSTransformer/util/hasMultipleDefinitions.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// hasMultipleDefinitions is TSTransformer/util/hasMultipleDefinitions.ts hasMultipleDefinitions.
func hasMultipleDefinitions(symbol *ast.Symbol, filter func(*ast.Node) bool) bool {
	amtValueDefinitions := 0
	for _, declaration := range symbol.Declarations {
		if filter(declaration) {
			amtValueDefinitions++
			if amtValueDefinitions > 1 {
				return true
			}
		}
	}
	return false
}
