package transform

// Port of TSTransformer/util/evaluation/bindings.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type BindingUsage struct {
	Captured         map[*ast.Symbol]bool
	WrittenByClosure map[*ast.Symbol]bool
}

// getBindingKey is TSTransformer/util/evaluation/bindings.ts getBindingKey.
func getBindingKey(state *TransformState, symbol *ast.Symbol, reference *ast.Node) *BindingKey {
	panic(todo("util/evaluation/bindings.ts", "getBindingKey"))
}
