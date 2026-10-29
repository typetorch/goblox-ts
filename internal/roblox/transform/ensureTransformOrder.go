package transform

// Port of TSTransformer/util/ensureTransformOrder.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// ensureTransformOrder is TSTransformer/util/ensureTransformOrder.ts ensureTransformOrder.
func ensureTransformOrder(state *TransformState, prereqs *Prereqs, nodes []*ast.Node, transformer func(*TransformState, *Prereqs, *ast.Node) luau.Expression) []luau.Expression {
	panic(todo("util/ensureTransformOrder.ts", "ensureTransformOrder"))
}
