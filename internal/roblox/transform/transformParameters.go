package transform

// Port of TSTransformer/nodes/transformParameters.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// optimizeArraySpreadParameter optimizes parameters in the form `...[a, b, c]: [A, B, C]` to be just `(a, b, c)`
func optimizeArraySpreadParameter(state *TransformState, prereqs *Prereqs, parameters *luau.List, bindingPattern *ast.Node) {
	for _, element := range bindingPattern.Elements() {
		if isOmittedBindingElement(element) {
			parameters.Push(luau.TempId())
		} else {
			name := element.Name()
			if ast.IsIdentifier(name) {
				paramId := transformIdentifierDefined(state, name)
				validateIdentifier(name)
				parameters.Push(paramId)
				if element.Initializer() != nil {
					prereqs.Push(transformInitializer(state, paramId, element.Initializer()))
				}
			} else {
				paramId := luau.TempId("param")
				parameters.Push(paramId)
				if element.Initializer() != nil {
					prereqs.Push(transformInitializer(state, paramId, element.Initializer()))
				}
				if ast.IsArrayBindingPattern(name) {
					transformArrayBindingPattern(state, prereqs, name, paramId)
				} else {
					transformObjectBindingPattern(state, prereqs, name, paramId)
				}
			}
		}
	}
}

// transformParameters is TSTransformer/nodes/transformParameters.ts transformParameters.
func transformParameters(state *TransformState, node *ast.Node) (parameters *luau.List, statements *luau.List, hasDotDotDot bool) {
	parameters = luau.MakeList()
	statements = luau.MakeList()
	hasDotDotDot = false

	if isMethod(state, node) {
		parameters.Push(luau.Globals.Self)
	}

	for _, parameter := range node.Parameters() {
		if ast.IsThisIdentifier(parameter.Name()) {
			continue
		}

		dotDotDotToken := parameter.AsParameterDeclaration().DotDotDotToken
		if dotDotDotToken != nil &&
			ast.IsArrayBindingPattern(parameter.Name()) &&
			!arrayLikeExpressionContainsSpread(parameter.Name()) {
			parameterPrereqs := NewPrereqs()
			optimizeArraySpreadParameter(state, parameterPrereqs, parameters, parameter.Name())
			statements.PushList(parameterPrereqs.Statements)
			continue
		}

		var paramId luau.AnyIdentifier
		if ast.IsIdentifier(parameter.Name()) {
			paramId = transformIdentifierDefined(state, parameter.Name())
			validateIdentifier(parameter.Name())
		} else {
			paramId = luau.TempId("param")
		}

		if dotDotDotToken != nil {
			hasDotDotDot = true
			statements.Push(luau.Create(&luau.VariableDeclaration{
				Left: paramId,
				Right: luau.Create(&luau.Array{
					Members: luau.MakeList(luau.Create(&luau.VarArgsLiteral{})),
				}),
			}))
		} else {
			parameters.Push(paramId)
		}

		if parameter.Initializer() != nil {
			statements.Push(transformInitializer(state, paramId, parameter.Initializer()))
		}

		// destructuring
		if !ast.IsIdentifier(parameter.Name()) {
			bindingPattern := parameter.Name()
			bindingPrereqs := NewPrereqs()
			if ast.IsArrayBindingPattern(bindingPattern) {
				transformArrayBindingPattern(state, bindingPrereqs, bindingPattern, paramId)
			} else {
				transformObjectBindingPattern(state, bindingPrereqs, bindingPattern, paramId)
			}
			statements.PushList(bindingPrereqs.Statements)
		}
	}

	return parameters, statements, hasDotDotDot
}
