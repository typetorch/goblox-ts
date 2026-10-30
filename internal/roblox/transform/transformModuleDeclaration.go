package transform

// Port of TSTransformer/nodes/statements/transformModuleDeclaration.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func isDeclarationOfNamespace(declaration *ast.Node) bool {
	if !ast.IsModuleDeclaration(declaration) &&
		!ast.IsFunctionDeclaration(declaration) &&
		!ast.IsClassDeclaration(declaration) &&
		!ast.IsEnumDeclaration(declaration) {
		return false
	}

	if hasDeclareModifier(declaration) {
		return false
	}

	if ast.IsModuleDeclaration(declaration) {
		return ast.IsInstantiatedModule(declaration, false)
	} else if ast.IsFunctionDeclaration(declaration) {
		return declaration.Body() != nil
	}
	return true
}

func getValueDeclarationStatement(symbol *ast.Symbol) *ast.Node {
	for _, declaration := range symbol.Declarations {
		statement := getAncestor(declaration, ast.IsStatement)
		assert(statement != nil)
		if ast.IsFunctionDeclaration(statement) && statement.Body() == nil {
			continue
		}
		if ast.IsTypeAliasDeclaration(statement) {
			continue
		}
		if ast.IsInterfaceDeclaration(statement) {
			continue
		}
		if hasDeclareModifier(statement) {
			continue
		}
		return statement
	}
	return nil
}

func transformNamespace(state *TransformState, name *ast.Node, body *ast.Node) *luau.List {
	symbol := state.TypeChecker.GetSymbolAtLocation(name)
	assert(symbol != nil)

	validateIdentifier(name)

	nameExp := transformIdentifierDefined(state, name)

	statements := luau.MakeList()
	doStatements := luau.MakeList()

	containerId := luau.TempId("container")
	state.SetModuleIdBySymbol(symbol, containerId)

	if state.IsHoisted[symbol] {
		statements.Push(luau.Create(&luau.Assignment{
			Left:     nameExp,
			Operator: "=",
			Right:    luau.NewMap(),
		}))
	} else {
		statements.Push(luau.Create(&luau.VariableDeclaration{
			Left:  nameExp,
			Right: luau.NewMap(),
		}))
	}

	moduleExports := state.GetModuleExports(symbol)
	if len(moduleExports) > 0 {
		doStatements.Push(luau.Create(&luau.VariableDeclaration{Left: containerId, Right: nameExp}))
	}

	if ast.IsModuleBlock(body) {
		exportsMap := map[*ast.Node][]string{}
		if len(moduleExports) > 0 {
			for _, exportSymbol := range moduleExports {
				originalSymbol := SkipAliases(state.TypeChecker, exportSymbol)
				if isSymbolOfValue(originalSymbol) && !isSymbolMutable(state, originalSymbol) {
					valueDeclarationStatement := getValueDeclarationStatement(exportSymbol)
					if valueDeclarationStatement != nil {
						exportsMap[valueDeclarationStatement] = append(exportsMap[valueDeclarationStatement], exportSymbol.Name)
					}
				}
			}
		}
		doStatements.PushList(transformStatementList(state, body, body.Statements(), &ExportInfo{
			Id:      containerId,
			Mapping: exportsMap,
		}))
	} else {
		innerName := body.Name()
		doStatements.PushList(transformNamespace(state, innerName, body.Body()))
		doStatements.Push(luau.Create(&luau.Assignment{
			Left:     luau.Property(containerId, innerName.Text()),
			Operator: "=",
			Right:    transformIdentifierDefined(state, innerName),
		}))
	}

	statements.Push(luau.Create(&luau.DoStatement{Statements: doStatements}))

	return statements
}

// transformModuleDeclaration is TSTransformer/nodes/statements/transformModuleDeclaration.ts transformModuleDeclaration.
func transformModuleDeclaration(state *TransformState, node *ast.Node) *luau.List {
	// type-only namespace
	if !ast.IsInstantiatedModule(node, false) {
		return luau.MakeList()
	}

	// disallow merging
	symbol := state.TypeChecker.GetSymbolAtLocation(node.Name())
	if symbol != nil && hasMultipleDefinitions(symbol, isDeclarationOfNamespace) {
		AddDiagnosticWithCache(
			symbol,
			diagnostics.Errors.NoNamespaceMerging.Create(node),
			state.MultiTransformState.IsReportedByMultipleDefinitionsCache,
		)
		return luau.MakeList()
	}

	// ts.StringLiteral is only in the case of `declare module "X" {}`? Should be filtered out above
	assert(!ast.IsStringLiteral(node.Name()))
	body := node.Body()
	assert(body != nil && !ast.IsIdentifier(body))
	// unsure how to filter out ts.JSDocNamespaceBody
	return transformNamespace(state, node.Name(), body)
}
