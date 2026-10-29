package transform

// Port of TSTransformer/util/addOneIfArrayType.ts.

import (
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// addOneIfArrayType is TSTransformer/util/addOneIfArrayType.ts addOneIfArrayType.
func addOneIfArrayType(state *TransformState, type_ *checker.Type, expression luau.Expression) luau.Expression {
	if isDefinitelyType(type_, isArrayType(state), isUndefinedType) {
		return offset(expression, 1)
	} else {
		return expression
	}
}
