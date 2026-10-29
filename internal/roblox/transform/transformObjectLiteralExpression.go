package transform

// Port of TSTransformer/nodes/expressions/transformObjectLiteralExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func transformPropertyAssignment(state *TransformState, prereqs *Prereqs, ptr *MapPointer, name *ast.Node, initializer *ast.Node) {
	leftPrereqs := NewPrereqs()
	left := transformPropertyName(state, leftPrereqs, name)
	rightPrereqs := NewPrereqs()
	right := transformExpression(state, rightPrereqs, initializer)

	if !leftPrereqs.Statements.IsEmpty() || !rightPrereqs.Statements.IsEmpty() {
		disableMapInline(prereqs, ptr)
		prereqs.PushList(leftPrereqs.Statements)
		left = prereqs.PushToVar(left, "left")
	}

	prereqs.PushList(rightPrereqs.Statements)

	assignToMapPointer(prereqs, ptr, left, right)
}

func transformSpreadAssignment(state *TransformState, prereqs *Prereqs, ptr *MapPointer, property *ast.Node) {
	expType := state.TypeChecker.GetNonOptionalType(state.GetType(property.Expression()))
	symbol := getFirstDefinedSymbol(state, expType)
	if symbol != nil && state.Services.MacroManager.IsMacroOnlyClass(symbol) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoMacroObjectSpread.Create(property))
	}

	type_ := state.GetType(property.Expression())
	definitelyObject := isDefinitelyType(type_, isObjectType)

	if m, ok := ptr.Value.(*luau.Map); definitelyObject && ok && m.Fields.IsEmpty() {
		ptr.Value = prereqs.PushToVar(
			luau.Call(luau.Globals.Table.Clone, transformExpression(state, prereqs, property.Expression())),
			ptr.Name,
		)
		prereqs.Push(luau.Create(&luau.CallStatement{
			// Explicitly remove metatable because things like classes can be spread
			Expression: luau.Call(luau.Globals.Setmetatable, ptr.Value, luau.Nil()),
		}))
		return
	}

	disableMapInline(prereqs, ptr)
	spreadExp := transformExpression(state, prereqs, property.Expression())
	if !definitelyObject {
		spreadExp = prereqs.PushToVarIfComplex(spreadExp, "spread")
	}

	keyId := luau.TempId("k")
	valueId := luau.TempId("v")
	var statement luau.Statement = luau.Create(&luau.ForStatement{
		Ids:        luau.MakeList(keyId, valueId),
		Expression: spreadExp,
		Statements: luau.MakeList(
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: ptr.Value,
					Index:      keyId,
				}),
				Operator: "=",
				Right:    valueId,
			}),
		),
	})

	if !definitelyObject {
		statement = luau.Create(&luau.IfStatement{
			Condition:  createTruthinessChecks(state, prereqs, spreadExp, property.Expression()),
			Statements: luau.MakeList(statement),
			ElseBody:   luau.MakeList(),
		})
	}

	prereqs.Push(statement)
}

// transformObjectLiteralExpression is TSTransformer/nodes/expressions/transformObjectLiteralExpression.ts transformObjectLiteralExpression.
func transformObjectLiteralExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	// starts as luau.Map, becomes luau.TemporaryIdentifier when `disableInline` is called
	ptr := createMapPointer("object")
	for _, property := range node.Properties() {
		validateMethodAssignment(state, property)
		if ast.IsPropertyAssignment(property) {
			if ast.IsPrivateIdentifier(property.Name()) {
				DiagnosticService.AddDiagnostic(diagnostics.Errors.NoPrivateIdentifier.Create(property.Name()))
				continue
			}
			transformPropertyAssignment(state, prereqs, ptr, property.Name(), property.Initializer())
		} else if ast.IsShorthandPropertyAssignment(property) {
			transformPropertyAssignment(state, prereqs, ptr, property.Name(), property.Name())
		} else if ast.IsSpreadAssignment(property) {
			transformSpreadAssignment(state, prereqs, ptr, property)
		} else if ast.IsMethodDeclaration(property) {
			prereqs.PushList(transformMethodDeclaration(state, prereqs, property, ptr))
		} else {
			// must be ts.AccessorDeclaration, which is banned
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoGetterSetter.Create(property))
		}
	}
	return ptr.Value
}
