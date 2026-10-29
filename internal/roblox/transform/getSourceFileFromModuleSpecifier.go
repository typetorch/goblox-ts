package transform

// Port of TSTransformer/util/getSourceFileFromModuleSpecifier.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
)

// getSourceFileFromModuleSpecifier is TSTransformer/util/getSourceFileFromModuleSpecifier.ts getSourceFileFromModuleSpecifier.
func getSourceFileFromModuleSpecifier(state *TransformState, moduleSpecifier *ast.Node) *ast.SourceFile {
	symbol := state.TypeChecker.GetSymbolAtLocation(moduleSpecifier)
	if symbol == nil {
		symbol = state.TypeChecker.ResolveExternalModuleName(moduleSpecifier)
	}
	if symbol != nil {
		declaration := symbol.ValueDeclaration

		if declaration != nil && ast.IsModuleDeclaration(declaration) && ast.IsStringLiteralLike(declaration.Name()) {
			sourceFile := ast.GetSourceFileOfNode(moduleSpecifier)
			mode := state.Program.GetModeForUsageLocation(sourceFile, declaration.Name())
			resolvedModuleInfo := state.Program.GetResolvedModule(sourceFile, declaration.Name().Text(), mode)
			if resolvedModuleInfo != nil && resolvedModuleInfo.IsResolved() {
				return state.Program.GetSourceFile(resolvedModuleInfo.ResolvedFileName)
			}
		}

		if declaration != nil && ast.IsSourceFile(declaration) {
			return declaration.AsSourceFile()
		}
	}

	// Fallback for $getModuleTree when module is not referenced by any regular import
	if ast.IsStringLiteralLike(moduleSpecifier) {
		sourceFile := ast.GetSourceFileOfNode(moduleSpecifier)
		result := state.Program.ResolveModuleName(moduleSpecifier.Text(), string(sourceFile.Path()), core.ResolutionModeNone)
		if result != nil && result.IsResolved() {
			return state.Program.GetSourceFile(result.ResolvedFileName)
		}
	}
	return nil
}
