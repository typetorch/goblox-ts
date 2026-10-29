package transform

// goblox only: TypeScript's symbol tables are insertion-ordered Maps, typescript-go's are Go maps, so
// checker.getExportsOfModule comes back in random order. roblox-ts emits exports in Map order, so
// orderModuleExports rebuilds the order getExportsOfModuleWorker produces: the module's own exports
// in binder (declaration) order, then the exports of each `export * from` target in statement order.

import (
	"slices"

	"github.com/microsoft/typescript-go/internal/ast"
)

// orderModuleExports returns exports (the result of GetExportsOfModule(moduleSymbol)) in TypeScript's order.
func (s *TransformState) orderModuleExports(moduleSymbol *ast.Symbol, exports []*ast.Symbol) []*ast.Symbol {
	byName := make(map[string]*ast.Symbol, len(exports))
	for _, exportSymbol := range exports {
		byName[exportSymbol.Name] = exportSymbol
	}

	result := make([]*ast.Symbol, 0, len(exports))
	added := make(map[string]bool, len(exports))
	add := func(name string) {
		if exportSymbol, ok := byName[name]; ok && !added[name] {
			added[name] = true
			result = append(result, exportSymbol)
		}
	}

	visited := map[*ast.Symbol]bool{}
	var visit func(symbol *ast.Symbol)
	visit = func(symbol *ast.Symbol) {
		if symbol == nil || visited[symbol] {
			return
		}
		visited[symbol] = true
		for _, name := range ownExportNamesInDeclarationOrder(symbol) {
			add(name)
		}
		if exportStars := symbol.Exports[ast.InternalSymbolNameExportStar]; exportStars != nil {
			declarations := slices.Clone(exportStars.Declarations)
			slices.SortStableFunc(declarations, compareDeclarationOrder)
			for _, declaration := range declarations {
				if ast.IsExportDeclaration(declaration) && declaration.AsExportDeclaration().ModuleSpecifier != nil {
					visit(s.TypeChecker.GetSymbolAtLocation(declaration.AsExportDeclaration().ModuleSpecifier))
				}
			}
		}
	}
	visit(moduleSymbol)

	// anything else (e.g. module augmentations) keeps a stable order
	var rest []string
	for name := range byName {
		if !added[name] {
			rest = append(rest, name)
		}
	}
	slices.Sort(rest)
	for _, name := range rest {
		add(name)
	}
	return result
}

// ownExportNamesInDeclarationOrder is the key order of symbol.exports as the binder inserted them. The
// binder binds the function declarations of a statement list first (bindEachFunctionsFirst), then the
// other statements.
func ownExportNamesInDeclarationOrder(symbol *ast.Symbol) []string {
	type entry struct {
		name        string
		declaration *ast.Node
	}
	entries := make([]entry, 0, len(symbol.Exports))
	for name, exportSymbol := range symbol.Exports {
		var first *ast.Node
		for _, declaration := range exportSymbol.Declarations {
			if first == nil || compareBindOrder(declaration, first) < 0 {
				first = declaration
			}
		}
		entries = append(entries, entry{name, first})
	}
	slices.SortStableFunc(entries, func(a, b entry) int {
		if a.declaration == nil || b.declaration == nil {
			if a.declaration == b.declaration {
				return compareStrings(a.name, b.name)
			}
			if a.declaration == nil {
				return 1
			}
			return -1
		}
		if c := compareBindOrder(a.declaration, b.declaration); c != 0 {
			return c
		}
		return compareStrings(a.name, b.name)
	})
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.name
	}
	return names
}

// compareBindOrder orders declarations the way bindEachFunctionsFirst binds them.
func compareBindOrder(a, b *ast.Node) int {
	if aFirst, bFirst := isBoundFunctionsFirst(a), isBoundFunctionsFirst(b); aFirst != bFirst {
		if aFirst {
			return -1
		}
		return 1
	}
	return compareDeclarationOrder(a, b)
}

func isBoundFunctionsFirst(declaration *ast.Node) bool {
	return ast.IsFunctionDeclaration(declaration) && declaration.Parent != nil &&
		(ast.IsSourceFile(declaration.Parent) || ast.IsModuleBlock(declaration.Parent) || ast.IsBlock(declaration.Parent))
}

// compareDeclarationOrder orders declarations by file name, then by position.
func compareDeclarationOrder(a, b *ast.Node) int {
	fileA, fileB := ast.GetSourceFileOfNode(a), ast.GetSourceFileOfNode(b)
	if fileA != fileB {
		if fileA == nil || fileB == nil {
			if fileA == nil {
				return 1
			}
			return -1
		}
		return compareStrings(fileA.FileName(), fileB.FileName())
	}
	return a.Pos() - b.Pos()
}

func compareStrings(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
