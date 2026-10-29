package transform

// Port of TSTransformer/util/getAddIterableToArrayBuilder.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

type AddIterableToArrayBuilder func(prereqs *Prereqs, expression luau.Expression, arrayId luau.AnyIdentifier, lengthId luau.AnyIdentifier, amtElementsSinceUpdate int, shouldUpdateLengthId bool) *luau.List

// getAddIterableToArrayBuilder is TSTransformer/util/getAddIterableToArrayBuilder.ts getAddIterableToArrayBuilder.
func getAddIterableToArrayBuilder(state *TransformState, node *ast.Node, type_ *checker.Type) AddIterableToArrayBuilder {
	panic(todo("util/getAddIterableToArrayBuilder.ts", "getAddIterableToArrayBuilder"))
}
