package transform

// Port of TSTransformer/util/bitwise.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// isBitwiseOperator is TSTransformer/util/bitwise.ts isBitwiseOperator.
func isBitwiseOperator(operatorKind ast.Kind) bool {
	panic(todo("util/bitwise.ts", "isBitwiseOperator"))
}

// createBitwiseCall is TSTransformer/util/bitwise.ts createBitwiseCall.
func createBitwiseCall(operatorKind ast.Kind, expressions []luau.Expression) luau.Expression {
	panic(todo("util/bitwise.ts", "createBitwiseCall"))
}

// createBitwiseFromOperator is TSTransformer/util/bitwise.ts createBitwiseFromOperator.
func createBitwiseFromOperator(state *TransformState, prereqs *Prereqs, operatorKind ast.Kind, node *ast.Node) luau.Expression {
	panic(todo("util/bitwise.ts", "createBitwiseFromOperator"))
}
