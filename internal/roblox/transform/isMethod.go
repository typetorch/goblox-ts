package transform

// Port of TSTransformer/util/isMethod.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
)

// isMethodFromType is TSTransformer/util/isMethod.ts isMethodFromType.
func isMethodFromType(state *TransformState, node *ast.Node, type_ *checker.Type) bool {
	panic(todo("util/isMethod.ts", "isMethodFromType"))
}

// isMethod is TSTransformer/util/isMethod.ts isMethod.
func isMethod(state *TransformState, node *ast.Node) bool {
	panic(todo("util/isMethod.ts", "isMethod"))
}
