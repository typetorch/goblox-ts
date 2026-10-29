package transform

// Port of TSTransformer/nodes/jsx/transformJsxAttributes.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func createJsxAttributeLoop(
	state *TransformState,
	prereqs *Prereqs,
	attributesPtrValue luau.Expression,
	expression luau.Expression,
	tsExpression *ast.Node,
) luau.Statement {
	definitelyObject := isDefinitelyType(state.GetType(tsExpression), isObjectType)
	if !definitelyObject {
		expression = prereqs.PushToVarIfComplex(expression, "attribute")
	}

	keyId := luau.TempId("k")
	valueId := luau.TempId("v")
	var statement luau.Statement = luau.Create(&luau.ForStatement{
		Ids:        luau.MakeList(keyId, valueId),
		Expression: expression,
		Statements: luau.MakeList(
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: attributesPtrValue,
					Index:      keyId,
				}),
				Operator: "=",
				Right:    valueId,
			}),
		),
	})

	if !definitelyObject {
		statement = luau.Create(&luau.IfStatement{
			Condition:  createTruthinessChecks(state, prereqs, expression, tsExpression),
			Statements: luau.MakeList(statement),
			ElseBody:   luau.MakeList(),
		})
	}

	return statement
}

func transformJsxAttribute(
	state *TransformState,
	prereqs *Prereqs,
	attribute *ast.Node,
	attributesPtr *MapPointer,
) {
	initializer := attribute.AsJsxAttribute().Initializer
	if initializer != nil && ast.IsJsxExpression(initializer) {
		initializer = initializer.AsJsxExpression().Expression
	}

	initPrereqs := NewPrereqs()
	var init luau.Expression
	if initializer != nil {
		init = transformExpression(state, initPrereqs, initializer)
	} else {
		init = luau.Bool(true)
	}

	if !initPrereqs.Statements.IsEmpty() {
		disableMapInline(prereqs, attributesPtr)
		prereqs.PushList(initPrereqs.Statements)
	}

	// attribute.name is an Identifier or a JsxNamespacedName, whose Text() is ts.getTextOfJsxNamespacedName
	text := attribute.Name().Text()
	name := luau.String(text)
	assignToMapPointer(prereqs, attributesPtr, name, init)
}

// transformJsxAttributes is TSTransformer/nodes/jsx/transformJsxAttributes.ts transformJsxAttributes.
func transformJsxAttributes(state *TransformState, prereqs *Prereqs, attributes *ast.Node, attributesPtr *MapPointer) {
	properties := attributes.AsJsxAttributes().Properties.Nodes
	for _, attribute := range properties {
		if ast.IsJsxAttribute(attribute) {
			transformJsxAttribute(state, prereqs, attribute, attributesPtr)
		} else {
			// spread attributes: `<frame { ...x }/>`
			attributeExpression := attribute.AsJsxSpreadAttribute().Expression

			expType := state.TypeChecker.GetNonOptionalType(state.GetType(attributeExpression))
			symbol := getFirstDefinedSymbol(state, expType)
			if symbol != nil && state.Services.MacroManager.IsMacroOnlyClass(symbol) {
				DiagnosticService.AddDiagnostic(diagnostics.Errors.NoMacroObjectSpread.Create(attribute))
			}

			expression := transformExpression(state, prereqs, attributeExpression)

			if attribute == properties[0] && isDefinitelyType(expType, isObjectType) {
				attributesPtr.Value = prereqs.PushToVar(
					luau.Call(luau.Globals.Table.Clone, expression),
					attributesPtr.Name,
				)
				prereqs.Push(
					luau.Create(&luau.CallStatement{
						// Explicitly remove metatable because things like classes can be spread
						Expression: luau.Call(luau.Globals.Setmetatable, attributesPtr.Value, luau.Nil()),
					}),
				)
				continue
			}

			disableMapInline(prereqs, attributesPtr)
			prereqs.Push(createJsxAttributeLoop(state, prereqs, attributesPtr.Value, expression, attributeExpression))
		}
	}
}
