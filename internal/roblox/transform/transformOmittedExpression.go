package transform

// Port of TSTransformer/nodes/expressions/transformOmittedExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformOmittedExpression is TSTransformer/nodes/expressions/transformOmittedExpression.ts transformOmittedExpression.
func transformOmittedExpression() *luau.NilLiteral {
	return luau.Nil()
}
