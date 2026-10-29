package transform

// Port of TSTransformer/nodes/expressions/transformIdentifier.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformIdentifierDefined is TSTransformer/nodes/expressions/transformIdentifier.ts transformIdentifierDefined.
func transformIdentifierDefined(state *TransformState, node *ast.Node) luau.AnyIdentifier {
	panic(todo("nodes/expressions/transformIdentifier.ts", "transformIdentifierDefined"))
}

// transformIdentifier is TSTransformer/nodes/expressions/transformIdentifier.ts transformIdentifier.
func transformIdentifier(state *TransformState, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformIdentifier.ts", "transformIdentifier"))
}
