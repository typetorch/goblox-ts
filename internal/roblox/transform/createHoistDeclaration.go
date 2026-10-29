package transform

// Port of TSTransformer/util/createHoistDeclaration.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// createHoistDeclaration is TSTransformer/util/createHoistDeclaration.ts createHoistDeclaration. It
// returns nil (TypeScript: undefined) when statement has no hoisted identifiers.
func createHoistDeclaration(state *TransformState, statement *ast.Node) *luau.VariableDeclaration {
	hoists := state.HoistsByStatement[statement]
	if len(hoists) > 0 {
		for _, hoist := range hoists {
			validateIdentifier(hoist)
		}
		left := luau.MakeList()
		for _, hoistId := range hoists {
			left.Push(transformIdentifierDefined(state, hoistId))
		}
		return luau.Create(&luau.VariableDeclaration{
			Left:  left,
			Right: nil,
		})
	}
	return nil
}
