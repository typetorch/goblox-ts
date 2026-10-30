package transform

// Port of TSTransformer/util/getOriginalSymbolOfNode.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
)

// SkipAliases is ts.skipAlias, down to a symbol that is not an alias. The checker's SkipAlias can stop
// at an alias: TypeScript 7 flags the `export=` symbol of a module that also exports types as
// NamespaceModule (bindCommonJSTypeExports), and resolveAlias does not follow such a symbol. TypeScript 5
// follows it, so `import X from` a module with `declare const X: X; export = X; export interface Y {}`
// has to end at the variable, or the import looks like it has no value and is dropped.
func SkipAliases(typeChecker *checker.Checker, symbol *ast.Symbol) *ast.Symbol {
	for symbol.Flags&ast.SymbolFlagsAlias != 0 {
		target := typeChecker.GetAliasedSymbol(symbol)
		if target == symbol {
			break
		}
		symbol = target
	}
	return symbol
}

// getOriginalSymbolOfNode is TSTransformer/util/getOriginalSymbolOfNode.ts getOriginalSymbolOfNode.
func getOriginalSymbolOfNode(typeChecker *checker.Checker, node *ast.Node) *ast.Symbol {
	symbol := typeChecker.GetSymbolAtLocation(node)
	if symbol != nil {
		return SkipAliases(typeChecker, symbol)
	}
	return symbol
}
