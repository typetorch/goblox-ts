package transform

// Port of TSTransformer/util/getAssignableValue.ts.

import (
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// getAssignableValue is TSTransformer/util/getAssignableValue.ts getAssignableValue.
func getAssignableValue(operator luau.AssignmentOperator, value luau.Expression, valueType *checker.Type) luau.Expression {
	if operator == "..=" && !isDefinitelyType(valueType, isStringType) {
		return luau.Call(luau.Globals.Tostring, value)
	}
	return value
}
