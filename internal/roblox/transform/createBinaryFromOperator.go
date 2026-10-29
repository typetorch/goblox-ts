package transform

// Port of TSTransformer/util/createBinaryFromOperator.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// createBinaryFromOperator is TSTransformer/util/createBinaryFromOperator.ts createBinaryFromOperator.
func createBinaryFromOperator(prereqs *Prereqs, left luau.Expression, leftType *checker.Type, operatorKind ast.Kind, right luau.Expression, rightType *checker.Type) luau.Expression {
	panic(todo("util/createBinaryFromOperator.ts", "createBinaryFromOperator"))
}
