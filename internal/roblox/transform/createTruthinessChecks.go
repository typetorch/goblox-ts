package transform

// Port of TSTransformer/util/createTruthinessChecks.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// willCreateTruthinessChecks is TSTransformer/util/createTruthinessChecks.ts willCreateTruthinessChecks.
func willCreateTruthinessChecks(type_ *checker.Type) bool {
	panic(todo("util/createTruthinessChecks.ts", "willCreateTruthinessChecks"))
}

// createTruthinessChecks is TSTransformer/util/createTruthinessChecks.ts createTruthinessChecks.
func createTruthinessChecks(state *TransformState, prereqs *Prereqs, exp luau.Expression, node *ast.Node) luau.Expression {
	panic(todo("util/createTruthinessChecks.ts", "createTruthinessChecks"))
}
