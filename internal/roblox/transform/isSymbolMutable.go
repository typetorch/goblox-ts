package transform

// Port of TSTransformer/util/isSymbolMutable.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// isSymbolMutable is TSTransformer/util/isSymbolMutable.ts isSymbolMutable.
func isSymbolMutable(state *TransformState, idSymbol *ast.Symbol) bool {
	return getOrSetDefault(state.MultiTransformState.IsDefinedAsLetCache, idSymbol, func() bool {
		if idSymbol.ValueDeclaration != nil {
			if ast.IsParameterDeclaration(idSymbol.ValueDeclaration) {
				return true
			}
			varDecList := getAncestor(idSymbol.ValueDeclaration, ast.IsVariableDeclarationList)
			if varDecList != nil {
				return varDecList.Flags&ast.NodeFlagsLet != 0
			}
		}
		return false
	})
}
