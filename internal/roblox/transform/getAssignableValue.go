package transform

// Port of TSTransformer/util/getAssignableValue.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// getAssignableValue is TSTransformer/util/getAssignableValue.ts getAssignableValue.
func getAssignableValue(operator luau.AssignmentOperator, value luau.Expression, valueType *checker.Type) luau.Expression {
	panic(todo("util/getAssignableValue.ts", "getAssignableValue"))
}
