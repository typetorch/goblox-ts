package transform

// Port of TSTransformer/util/binding/createObjectRest.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// createObjectRest is TSTransformer/util/binding/createObjectRest.ts createObjectRest.
func createObjectRest(state *TransformState, prereqs *Prereqs, node *ast.Node, type_ *checker.Type, source luau.AnyIdentifier, keys []luau.Expression) luau.Expression {
	panic(todo("util/binding/createObjectRest.ts", "createObjectRest"))
}
