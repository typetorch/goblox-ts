package transform

// Port of TSTransformer/nodes/transformMethodDeclaration.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformMethodDeclaration is TSTransformer/nodes/transformMethodDeclaration.ts transformMethodDeclaration.
func transformMethodDeclaration(state *TransformState, prereqs *Prereqs, node *ast.Node, ptr *Pointer) *luau.List {
	result := luau.MakeList()

	if node.Body() == nil {
		return luau.MakeList()
	}

	assert(node.Name() != nil)
	if ast.IsPrivateIdentifier(node.Name()) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoPrivateIdentifier.Create(node.Name()))
		return luau.MakeList()
	}

	parameters, statements, hasDotDotDot := transformParameters(state, node)
	statements.PushList(transformStatementList(state, node.Body(), node.Body().Statements(), nil))

	name := transformPropertyName(state, prereqs, node.Name())

	parameterHasDecorators := false
	for _, parameter := range node.Parameters() {
		if ast.HasDecorators(parameter) {
			parameterHasDecorators = true
			break
		}
	}
	if ast.HasDecorators(node) || parameterHasDecorators {
		if !luau.IsSimplePrimitive(name) {
			tempId := luau.TempId("key")
			result.Push(luau.Create(&luau.VariableDeclaration{
				Left:  tempId,
				Right: name,
			}))
			name = tempId
		}
		state.SetClassElementObjectKey(node, name)
	}

	isAsync := ast.HasSyntacticModifier(node, ast.ModifierFlagsAsync)

	if node.BodyData().AsteriskToken != nil {
		if isAsync {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoAsyncGeneratorFunctions.Create(node))
		}
		statements = wrapStatementsAsGenerator(state, node, statements)
	}

	// can we use `function class:name() end`?
	if stringLiteral, ok := name.(*luau.StringLiteral); !isAsync && ok && !luau.IsMap(ptr.Value) && luau.IsValidIdentifier(stringLiteral.Value) {
		if isMethod(state, node) {
			parameters.Shift() // remove `self`
			result.Push(luau.Create(&luau.MethodDeclaration{
				Expression:   ptr.Value,
				Name:         stringLiteral.Value,
				Statements:   statements,
				Parameters:   parameters,
				HasDotDotDot: hasDotDotDot,
			}))
		} else {
			result.Push(luau.Create(&luau.FunctionDeclaration{
				Name:         luau.Property(ptr.Value, stringLiteral.Value),
				Localize:     false,
				Statements:   statements,
				Parameters:   parameters,
				HasDotDotDot: hasDotDotDot,
			}))
		}
		return result
	}

	var expression luau.Expression = luau.Create(&luau.FunctionExpression{
		Statements:   statements,
		Parameters:   parameters,
		HasDotDotDot: hasDotDotDot,
	})

	if isAsync {
		expression = luau.Call(state.TS(node, "async"), expression)
	}

	// we have to use `class[name] = function()`
	assignmentPrereqs := NewPrereqs()
	assignToMapPointer(assignmentPrereqs, ptr, name, expression)
	result.PushList(assignmentPrereqs.Statements)

	return result
}
