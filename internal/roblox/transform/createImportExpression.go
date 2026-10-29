package transform

// Port of TSTransformer/util/createImportExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// getImportParts is TSTransformer/util/createImportExpression.ts getImportParts.
func getImportParts(state *TransformState, sourceFile *ast.SourceFile, moduleSpecifier *ast.Node) []luau.Expression {
	panic(todo("util/createImportExpression.ts", "getImportParts"))
}

// createImportExpression is TSTransformer/util/createImportExpression.ts createImportExpression.
func createImportExpression(state *TransformState, sourceFile *ast.SourceFile, moduleSpecifier *ast.Node) luau.IndexableExpression {
	panic(todo("util/createImportExpression.ts", "createImportExpression"))
}
