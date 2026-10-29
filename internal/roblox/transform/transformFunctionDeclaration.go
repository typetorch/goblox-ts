package transform

// Port of TSTransformer/nodes/statements/transformFunctionDeclaration.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformFunctionDeclaration is TSTransformer/nodes/statements/transformFunctionDeclaration.ts transformFunctionDeclaration.
func transformFunctionDeclaration(state *TransformState, node *ast.Node) *luau.List {
	if node.Body() == nil {
		return luau.MakeList()
	}

	isExportDefault := ast.HasSyntacticModifier(node, ast.ModifierFlagsExportDefault)

	assert(node.Name() != nil || isExportDefault)

	if node.Name() != nil {
		validateIdentifier(node.Name())
	}

	var name luau.AnyIdentifier
	if node.Name() != nil {
		name = transformIdentifierDefined(state, node.Name())
	} else {
		name = luau.Id("default")
	}

	parameters, statements, hasDotDotDot := transformParameters(state, node)
	statements.PushList(transformStatementList(state, node.Body(), node.Body().Statements(), nil))

	localize := isExportDefault
	if node.Name() != nil {
		symbol := state.TypeChecker.GetSymbolAtLocation(node.Name())
		assert(symbol != nil)
		localize = !state.IsHoisted[symbol]
	}

	isAsync := ast.HasSyntacticModifier(node, ast.ModifierFlagsAsync)
	asteriskToken := node.BodyData().AsteriskToken
	if node.Name() != nil && !isAsync && asteriskToken == nil {
		symbol := state.TypeChecker.GetSymbolAtLocation(node.Name())
		assert(symbol != nil)
		state.MultiTransformState.FunctionEffects[symbol] = getFunctionEffects(parameters, statements)
	}

	if asteriskToken != nil {
		if isAsync {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoAsyncGeneratorFunctions.Create(node))
		}
		statements = wrapStatementsAsGenerator(state, node, statements)
	}

	if isAsync {
		right := luau.Call(state.TS(node, "async"),
			luau.Create(&luau.FunctionExpression{
				HasDotDotDot: hasDotDotDot,
				Parameters:   parameters,
				Statements:   statements,
			}),
		)
		if localize {
			return luau.MakeList(
				luau.Create(&luau.VariableDeclaration{
					Left:  name,
					Right: right,
				}),
			)
		} else {
			return luau.MakeList(
				luau.Create(&luau.Assignment{
					Left:     name,
					Operator: "=",
					Right:    right,
				}),
			)
		}
	} else {
		return luau.MakeList(
			luau.Create(&luau.FunctionDeclaration{Localize: localize, Name: name, Statements: statements, Parameters: parameters, HasDotDotDot: hasDotDotDot}),
		)
	}
}
