package transform

// Port of TSTransformer/util/binding/createArrayBindingAccessor.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

type BindingAccessor interface {
	Read(prereqs *Prereqs, index int, omitted bool) luau.Expression
	Rest(prereqs *Prereqs, index int) luau.Expression
}

// createArrayBindingAccessor is TSTransformer/util/binding/createArrayBindingAccessor.ts createArrayBindingAccessor.
func createArrayBindingAccessor(state *TransformState, prereqs *Prereqs, node *ast.Node, type_ *checker.Type, parentId luau.AnyIdentifier) BindingAccessor {
	panic(todo("util/binding/createArrayBindingAccessor.ts", "createArrayBindingAccessor"))
}
