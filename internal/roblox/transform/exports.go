package transform

// Entry points for the driver in internal/roblox/project (TSTransformer/index.ts re-exports these).

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// CreateTransformServices is createTransformServices.
func CreateTransformServices(typeChecker *checker.Checker) *TransformServices {
	return createTransformServices(typeChecker)
}

// TransformSourceFile is transformSourceFile.
func TransformSourceFile(state *TransformState, sourceFile *ast.SourceFile) *luau.List {
	if state.CompilerOptions.NoCheck.IsTrue() {
		// import elision reads what the type check marks, as TypeScript's ImportElisionTransformer does
		state.TypeChecker.MarkLinkedReferencesRecursively(sourceFile)
	}
	return transformSourceFile(state, sourceFile)
}

// IsPathDescendantOf is Shared/util/isPathDescendantOf.ts.
func IsPathDescendantOf(filePath string, dirPath string) bool {
	return isPathDescendantOf(filePath, dirPath)
}
