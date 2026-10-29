package transform

// Port of TSTransformer/nodes/class/transformPropertyDeclaration.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformPropertyDeclaration is TSTransformer/nodes/class/transformPropertyDeclaration.ts transformPropertyDeclaration.
func transformPropertyDeclaration(state *TransformState, prereqs *Prereqs, node *ast.Node, name luau.AnyIdentifier) *luau.List {
	if !ast.HasStaticModifier(node) {
		return luau.MakeList()
	}

	if ast.IsPrivateIdentifier(node.Name()) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoPrivateIdentifier.Create(node))
		return luau.MakeList()
	}

	initializer := node.Initializer()
	if initializer == nil {
		return luau.MakeList()
	}

	left := luau.Create(&luau.ComputedIndexExpression{
		Expression: name,
		Index:      transformPropertyName(state, prereqs, node.Name()),
	})
	return luau.MakeList(
		luau.Create(&luau.Assignment{
			Left:     left,
			Operator: "=",
			Right:    transformExpression(state, prereqs, initializer),
		}),
	)
}
