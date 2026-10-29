package transform

// Port of TSTransformer/util/binding/objectAccessor.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// objectAccessor is TSTransformer/util/binding/objectAccessor.ts objectAccessor.
func objectAccessor(state *TransformState, prereqs *Prereqs, parentId luau.IndexableExpression, receiverType *checker.Type, name *ast.Node) (key luau.Expression, value luau.Expression) {
	panic(todo("util/binding/objectAccessor.ts", "objectAccessor"))
}
