package transform

// Port of TSTransformer/util/getConstantValueLiteral.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/jsnum"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// getConstantValueLiteral is TSTransformer/util/getConstantValueLiteral.ts getConstantValueLiteral.
func getConstantValueLiteral(state *TransformState, node *ast.Node) luau.Expression {
	switch constantValue := state.TypeChecker.GetConstantValue(node).(type) {
	case string:
		return luau.String(constantValue)
	case jsnum.Number:
		return luau.Number(float64(constantValue))
	case float64:
		return luau.Number(constantValue)
	}
	return nil
}
