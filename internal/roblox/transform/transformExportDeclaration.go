package transform

// Port of TSTransformer/nodes/statements/transformExportDeclaration.ts.

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func isExportSpecifierValue(state *TransformState, element *ast.Node) bool {
	if element.IsTypeOnly() {
		return false
	}

	if state.isReferencedAliasDeclaration(element) {
		return true
	}

	aliasSymbol := state.TypeChecker.GetSymbolAtLocation(element.Name())
	if aliasSymbol != nil && isSymbolOfValue(skipAliases(state.TypeChecker, aliasSymbol)) {
		return true
	}

	return false
}

// countExportImportExpUses is transformExportDeclaration.ts countImportExpUses (renamed: the package
// already has transformImportDeclaration.ts countImportExpUses).
func countExportImportExpUses(state *TransformState, exportClause *ast.Node) int {
	uses := 0
	if exportClause != nil && ast.IsNamedExports(exportClause) {
		for _, element := range exportClause.Elements() {
			if isExportSpecifierValue(state, element) {
				uses++
			}
		}
	} else {
		uses++
	}
	return uses
}

func transformExportFrom(state *TransformState, node *ast.Node) *luau.List {
	exportDeclaration := node.AsExportDeclaration()
	moduleSpecifier := exportDeclaration.ModuleSpecifier
	assert(moduleSpecifier != nil && ast.IsStringLiteral(moduleSpecifier))

	statements := luau.MakeList()
	var importExp luau.IndexableExpression

	exportClause := exportDeclaration.ExportClause

	// detect if we need to push to a new var or not
	uses := countExportImportExpUses(state, exportClause)
	if uses == 1 {
		importExp = createImportExpression(state, ast.GetSourceFileOfNode(node), moduleSpecifier)
	} else if uses > 1 {
		moduleName := strings.Split(moduleSpecifier.Text(), "/")
		id := luau.TempId(cleanModuleName(moduleName[len(moduleName)-1]))
		importExp = id
		statements.Push(luau.Create(&luau.VariableDeclaration{
			Left:  id,
			Right: createImportExpression(state, ast.GetSourceFileOfNode(node), moduleSpecifier),
		}))
	}

	if importExp == nil {
		return statements
	}

	moduleId := state.GetModuleIdFromNode(node)
	if exportClause != nil {
		if ast.IsNamedExports(exportClause) {
			// export { a, b, c } from "./module";
			for _, element := range exportClause.Elements() {
				if isExportSpecifierValue(state, element) {
					namePrereqs := NewPrereqs()
					exportName := transformPropertyName(state, namePrereqs, element.Name())
					propertyName := element.PropertyName()
					if propertyName == nil {
						propertyName = element.Name()
					}
					importName := transformPropertyName(state, namePrereqs, propertyName)
					statements.PushList(namePrereqs.Statements)
					statements.Push(luau.Create(&luau.Assignment{
						Left: luau.Create(&luau.ComputedIndexExpression{
							Expression: moduleId,
							Index:      exportName,
						}),
						Operator: "=",
						Right: luau.Create(&luau.ComputedIndexExpression{
							Expression: importExp,
							Index:      importName,
						}),
					}))
				}
			}
		} else {
			// export * as foo from "./module";
			namePrereqs := NewPrereqs()
			name := transformPropertyName(state, namePrereqs, exportClause.Name())
			statements.PushList(namePrereqs.Statements)
			statements.Push(luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: moduleId,
					Index:      name,
				}),
				Operator: "=",
				Right:    importExp,
			}))
		}
	} else {
		// export * from "./module";
		keyId := luau.TempId("k")
		valueId := luau.TempId("v")
		statements.Push(luau.Create(&luau.ForStatement{
			Ids: luau.MakeList(keyId, valueId),
			// importExp may be `nil` in .d.ts files, so default to `{}`
			// boolean `or` is safe, because importExp can only be a table if not `nil`
			Expression: luau.Binary(importExp, "or", luau.NewMap()),
			Statements: luau.MakeList(
				luau.Create(&luau.Assignment{
					Left: luau.Create(&luau.ComputedIndexExpression{
						Expression: moduleId,
						Index:      keyId,
					}),
					Operator: "=",
					Right:    valueId,
				}),
			),
		}))
	}

	state.HasExportFrom = true

	return statements
}

// transformExportDeclaration is TSTransformer/nodes/statements/transformExportDeclaration.ts transformExportDeclaration.
func transformExportDeclaration(state *TransformState, node *ast.Node) *luau.List {
	if node.IsTypeOnly() {
		return luau.MakeList()
	}

	if node.AsExportDeclaration().ModuleSpecifier != nil {
		return transformExportFrom(state, node)
	}

	return luau.MakeList()
}
