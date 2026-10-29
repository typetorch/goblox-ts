package transform

// Port of TSTransformer/util/getOriginalSymbolOfNode.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
)

// getOriginalSymbolOfNode is TSTransformer/util/getOriginalSymbolOfNode.ts getOriginalSymbolOfNode.
func getOriginalSymbolOfNode(typeChecker *checker.Checker, node *ast.Node) *ast.Symbol {
	symbol := typeChecker.GetSymbolAtLocation(node)
	if symbol != nil {
		return typeChecker.SkipAlias(symbol)
	}
	return symbol
}
