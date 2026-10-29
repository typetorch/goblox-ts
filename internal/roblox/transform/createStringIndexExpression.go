package transform

// Port of TSTransformer/util/createStringIndexExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// createStringIndexExpression is TSTransformer/util/createStringIndexExpression.ts createStringIndexExpression.
func createStringIndexExpression(prereqs *Prereqs, expression luau.Expression, indexOperand *EvaluationOperand, indexType *checker.Type) luau.Expression {
	panic(todo("util/createStringIndexExpression.ts", "createStringIndexExpression"))
}
