package transform

// Port of TSTransformer/util/evaluation/builtins.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// isBuiltinLibrary is TSTransformer/util/evaluation/builtins.ts isBuiltinLibrary.
func isBuiltinLibrary(state *TransformState, node *ast.Node) bool {
	panic(todo("util/evaluation/builtins.ts", "isBuiltinLibrary"))
}

// tryMarkBuiltinLibrary is TSTransformer/util/evaluation/builtins.ts tryMarkBuiltinLibrary.
func tryMarkBuiltinLibrary(state *TransformState, node *ast.Node, expression luau.Expression) {
	panic(todo("util/evaluation/builtins.ts", "tryMarkBuiltinLibrary"))
}

// isStableBuiltinMember is TSTransformer/util/evaluation/builtins.ts isStableBuiltinMember.
func isStableBuiltinMember(state *TransformState, node *ast.Node) bool {
	panic(todo("util/evaluation/builtins.ts", "isStableBuiltinMember"))
}

// tryMarkBuiltinMember is TSTransformer/util/evaluation/builtins.ts tryMarkBuiltinMember.
func tryMarkBuiltinMember(state *TransformState, node *ast.Node, expression luau.Expression) {
	panic(todo("util/evaluation/builtins.ts", "tryMarkBuiltinMember"))
}
