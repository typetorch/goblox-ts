package transform

// Port of TSTransformer/util/isSymbolOfValue.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// isSymbolOfValue checks that the symbol represents something with a value at runtime,
// thus excluding types and const enums.
func isSymbolOfValue(symbol *ast.Symbol) bool {
	return symbol.Flags&ast.SymbolFlagsValue != 0 && symbol.Flags&ast.SymbolFlagsConstEnum == 0
}
