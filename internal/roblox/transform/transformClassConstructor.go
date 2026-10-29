package transform

// Port of TSTransformer/nodes/class/transformClassConstructor.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

const CONSTRUCTOR = "constructor"

func transformPropertyInitializers(state *TransformState, node *ast.Node) *luau.List {
	statements := luau.MakeList()
	for _, member := range node.Members() {
		if !ast.IsPropertyDeclaration(member) {
			continue
		}
		if ast.HasStaticModifier(member) {
			continue
		}

		name := member.Name()
		if ast.IsPrivateIdentifier(name) {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoPrivateIdentifier.Create(node))
			continue
		}

		initializer := member.Initializer()
		if initializer == nil {
			continue
		}

		indexPrereqs := NewPrereqs()
		index := transformPropertyName(state, indexPrereqs, name)
		statements.PushList(indexPrereqs.Statements)

		rightPrereqs := NewPrereqs()
		right := transformExpression(state, rightPrereqs, initializer)
		statements.PushList(rightPrereqs.Statements)

		statements.Push(
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: luau.Globals.Self,
					Index:      index,
				}),
				Operator: "=",
				Right:    right,
			}),
		)
	}
	return statements
}

// transformImplicitClassConstructor is TSTransformer/nodes/class/transformClassConstructor.ts transformImplicitClassConstructor.
func transformImplicitClassConstructor(state *TransformState, node *ast.Node, name luau.AnyIdentifier) *luau.List {
	statements := luau.MakeList()

	hasDotDotDot := false

	// if extends + no constructor:
	// - add ... to params
	// - add super.constructor(self, ...)
	if getExtendsNode(node) != nil {
		hasDotDotDot = true
		statements.Push(
			luau.Create(&luau.CallStatement{
				Expression: luau.Call(luau.Property(luau.Globals.Super, CONSTRUCTOR),
					luau.Globals.Self,
					luau.Create(&luau.VarArgsLiteral{}),
				),
			}),
		)
	}

	statements.PushList(transformPropertyInitializers(state, node))

	return luau.MakeList(
		luau.Create(&luau.MethodDeclaration{
			Expression:   name,
			Name:         CONSTRUCTOR,
			Statements:   statements,
			Parameters:   luau.MakeList(),
			HasDotDotDot: hasDotDotDot,
		}),
	)
}

// transformClassConstructor is TSTransformer/nodes/class/transformClassConstructor.ts transformClassConstructor.
func transformClassConstructor(state *TransformState, node *ast.Node, name luau.AnyIdentifier) *luau.List {
	parameters, statements, hasDotDotDot := transformParameters(state, node)
	body := node.Body()
	bodyStatements := getStatements(body)

	// property parameters must come after the first super() call
	superIndex := -1
	for i, v := range bodyStatements {
		if ast.IsExpressionStatement(v) && ast.IsSuperCall(v.Expression()) {
			superIndex = i
			break
		}
	}

	statements.PushList(transformStatementList(state, body, bodyStatements[:superIndex+1], nil))

	for _, parameter := range node.Parameters() {
		if ast.IsParameterPropertyDeclaration(parameter, parameter.Parent) {
			paramId := transformIdentifierDefined(state, parameter.Name())
			statements.Push(
				luau.Create(&luau.Assignment{
					Left:     luau.Property(luau.Globals.Self, paramId.(*luau.Identifier).Name),
					Operator: "=",
					Right:    paramId,
				}),
			)
		}
	}

	statements.PushList(transformPropertyInitializers(state, node.Parent))

	statements.PushList(transformStatementList(state, body, bodyStatements[superIndex+1:], nil))

	return luau.MakeList(
		luau.Create(&luau.MethodDeclaration{
			Expression:   name,
			Name:         CONSTRUCTOR,
			Statements:   statements,
			Parameters:   parameters,
			HasDotDotDot: hasDotDotDot,
		}),
	)
}
