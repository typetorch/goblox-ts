package transform

// Port of TSTransformer/nodes/expressions/transformImportExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformImportExpression is TSTransformer/nodes/expressions/transformImportExpression.ts transformImportExpression.
func transformImportExpression(state *TransformState, node *ast.Node) luau.Expression {
	var moduleSpecifier *ast.Node
	if args := node.Arguments(); len(args) > 0 {
		moduleSpecifier = args[0]
	}

	if moduleSpecifier == nil || !ast.IsStringLiteral(moduleSpecifier) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoNonStringModuleSpecifier.Create(node))
		return luau.NewNone()
	}

	importExpression := createImportExpression(state, ast.GetSourceFileOfNode(node), moduleSpecifier)
	resolveId := luau.Id("resolve")

	return luau.Call(luau.Property(state.TS(node, "Promise"), "new"),
		luau.Create(&luau.FunctionExpression{
			HasDotDotDot: false,
			Parameters:   luau.MakeList(resolveId),
			Statements: luau.MakeList(
				luau.Create(&luau.CallStatement{
					Expression: luau.Call(resolveId, importExpression),
				}),
			),
		}),
	)
}
