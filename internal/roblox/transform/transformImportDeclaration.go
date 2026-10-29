package transform

// Port of TSTransformer/nodes/statements/transformImportDeclaration.ts.

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// isReferencedAliasDeclaration is `state.resolver.isReferencedAliasDeclaration(node)`.
func (s *TransformState) isReferencedAliasDeclaration(node *ast.Node) bool {
	return s.TypeChecker.IsReferencedAliasDeclaration(node)
}

func countImportExpUses(state *TransformState, importClause *ast.Node) int {
	uses := 0

	if name := importClause.Name(); name != nil {
		symbol := getOriginalSymbolOfNode(state.TypeChecker, name)
		if state.isReferencedAliasDeclaration(importClause) && (symbol == nil || isSymbolOfValue(symbol)) {
			uses++
		}
	}

	if namedBindings := importClause.AsImportClause().NamedBindings; namedBindings != nil {
		if ast.IsNamespaceImport(namedBindings) {
			uses++
		} else {
			for _, element := range namedBindings.Elements() {
				symbol := getOriginalSymbolOfNode(state.TypeChecker, element.Name())
				if state.isReferencedAliasDeclaration(element) && (symbol == nil || isSymbolOfValue(symbol)) {
					uses++
				}
			}
		}
	}

	return uses
}

// transformImportDeclaration is TSTransformer/nodes/statements/transformImportDeclaration.ts transformImportDeclaration.
func transformImportDeclaration(state *TransformState, node *ast.Node) *luau.List {
	importDeclaration := node.AsImportDeclaration()
	// no emit for type only
	importClause := importDeclaration.ImportClause
	if importClause != nil && importClause.AsImportClause().PhaseModifier == ast.KindTypeKeyword {
		return luau.MakeList()
	}

	statements := luau.MakeList()

	moduleSpecifier := importDeclaration.ModuleSpecifier
	assert(ast.IsStringLiteral(moduleSpecifier))
	// Lazy<luau.IndexableExpression>
	var importExpValue luau.IndexableExpression
	getImportExp := func() luau.IndexableExpression {
		if importExpValue == nil {
			importExpValue = createImportExpression(state, ast.GetSourceFileOfNode(node), moduleSpecifier)
		}
		return importExpValue
	}

	if importClause != nil {
		// detect if we need to push to a new var or not
		uses := countImportExpUses(state, importClause)
		if uses > 1 {
			moduleName := strings.Split(moduleSpecifier.Text(), "/")
			id := luau.TempId(cleanModuleName(moduleName[len(moduleName)-1]))
			statements.Push(luau.Create(&luau.VariableDeclaration{
				Left:  id,
				Right: getImportExp(),
			}))
			importExpValue = id
		}

		bindingPrereqs := NewPrereqs()

		// default import logic
		if importClauseName := importClause.Name(); importClauseName != nil {
			symbol := getOriginalSymbolOfNode(state.TypeChecker, importClauseName)
			if state.isReferencedAliasDeclaration(importClause) && (symbol == nil || isSymbolOfValue(symbol)) {
				moduleFile := getSourceFileFromModuleSpecifier(state, moduleSpecifier)
				var moduleSymbol *ast.Symbol
				if moduleFile != nil {
					moduleSymbol = state.TypeChecker.GetSymbolAtLocation(moduleFile.AsNode())
				}
				hasDefault := false
				if moduleSymbol != nil {
					for _, v := range state.GetModuleExports(moduleSymbol) {
						if v.Name == "default" {
							hasDefault = true
							break
						}
					}
				}
				if hasDefault {
					transformVariable(state, bindingPrereqs, importClauseName, luau.Property(getImportExp(), "default"))
				} else {
					transformVariable(state, bindingPrereqs, importClauseName, getImportExp())
				}
			}
		}

		if importClauseNamedBindings := importClause.AsImportClause().NamedBindings; importClauseNamedBindings != nil {
			// namespace import logic
			if ast.IsNamespaceImport(importClauseNamedBindings) {
				transformVariable(state, bindingPrereqs, importClauseNamedBindings.Name(), getImportExp())
			} else {
				// named elements import logic
				for _, element := range importClauseNamedBindings.Elements() {
					symbol := getOriginalSymbolOfNode(state.TypeChecker, element.Name())
					// check that import is referenced and has a value at runtime
					if state.isReferencedAliasDeclaration(element) && (symbol == nil || isSymbolOfValue(symbol)) {
						propertyName := element.PropertyName()
						if propertyName == nil {
							propertyName = element.Name()
						}
						transformVariable(state, bindingPrereqs, element.Name(), luau.Create(&luau.ComputedIndexExpression{
							Expression: getImportExp(),
							Index:      transformPropertyName(state, bindingPrereqs, propertyName),
						}))
					}
				}
			}
		}

		statements.PushList(bindingPrereqs.Statements)
	}

	// ensure we emit something
	if importClause == nil || (state.CompilerOptions.VerbatimModuleSyntax.IsTrue() && statements.IsEmpty()) {
		expression := getImportExp()
		assert(luau.IsCallExpression(expression))
		statements.Push(luau.Create(&luau.CallStatement{Expression: expression}))
	}

	return statements
}
