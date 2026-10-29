package transform

// Port of TSTransformer/nodes/expressions/transformFunctionExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformFunctionExpression is TSTransformer/nodes/expressions/transformFunctionExpression.ts transformFunctionExpression.
func transformFunctionExpression(state *TransformState, node *ast.Node) luau.Expression {
	if node.Name() != nil {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoFunctionExpressionName.Create(node.Name()))
	}

	parameters, statements, hasDotDotDot := transformParameters(state, node)

	body := node.Body()
	if ast.IsBlock(body) {
		statements.PushList(transformStatementList(state, body, body.Statements(), nil))
	} else {
		statements.PushList(transformReturnStatementInner(state, body))
	}

	isAsync := ast.HasSyntacticModifier(node, ast.ModifierFlagsAsync)
	asteriskToken := node.BodyData().AsteriskToken

	if asteriskToken != nil {
		if isAsync {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoAsyncGeneratorFunctions.Create(node))
		}
		statements = wrapStatementsAsGenerator(state, node, statements)
	}

	var expression luau.Expression = luau.Create(&luau.FunctionExpression{
		HasDotDotDot: hasDotDotDot,
		Parameters:   parameters,
		Statements:   statements,
	})

	if isAsync {
		expression = luau.Call(state.TS(node, "async"), expression)
	} else if asteriskToken == nil {
		setCallEffects(expression, getFunctionEffects(parameters, statements))
	}

	return expression
}
