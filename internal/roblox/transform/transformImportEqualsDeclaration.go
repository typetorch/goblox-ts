package transform

// Port of TSTransformer/nodes/statements/transformImportEqualsDeclaration.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformImportEqualsDeclaration is TSTransformer/nodes/statements/transformImportEqualsDeclaration.ts transformImportEqualsDeclaration.
func transformImportEqualsDeclaration(state *TransformState, node *ast.Node) *luau.List {
	if node.IsTypeOnly() {
		return luau.MakeList()
	}

	moduleReference := node.AsImportEqualsDeclaration().ModuleReference
	if ast.IsExternalModuleReference(moduleReference) {
		statements := luau.MakeList()
		aliasSymbol := state.TypeChecker.GetSymbolAtLocation(node.Name())
		assert(aliasSymbol != nil)
		isValue := isSymbolOfValue(skipAliases(state.TypeChecker, aliasSymbol))
		if !isValue && !state.CompilerOptions.VerbatimModuleSyntax.IsTrue() {
			return statements
		}

		expression := moduleReference.AsExternalModuleReference().Expression
		assert(ast.IsStringLiteral(expression))
		importExp := createImportExpression(state, ast.GetSourceFileOfNode(node), expression)
		if isValue {
			importPrereqs := NewPrereqs()
			transformVariable(state, importPrereqs, node.Name(), importExp)
			statements.PushList(importPrereqs.Statements)
		}

		// ensure we emit something
		if state.CompilerOptions.VerbatimModuleSyntax.IsTrue() &&
			statements.IsEmpty() &&
			luau.IsCallExpression(importExp) {
			statements.Push(luau.Create(&luau.CallStatement{Expression: importExp}))
		}

		return statements
	} else {
		// Identifier | QualifiedName
		// see: https://github.com/roblox-ts/roblox-ts/issues/1895
		importPrereqs := NewPrereqs()
		transformVariable(state, importPrereqs, node.Name(), transformEntityName(state, moduleReference))
		return importPrereqs.Statements
	}
}
