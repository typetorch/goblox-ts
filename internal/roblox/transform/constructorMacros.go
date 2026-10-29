package transform

// Port of TSTransformer/macros/constructorMacros.ts.

import (
	"slices"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func wrapWeak(state *TransformState, prereqs *Prereqs, node *ast.Node, macro ConstructorMacro) luau.Expression {
	return luau.Call(luau.Globals.Setmetatable,
		macro(state, prereqs, node),
		luau.NewMap(luau.MapEntry{Index: luau.Strings.Mode, Value: luau.Strings.K}),
	)
}

func ArrayConstructor(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	if len(node.Arguments()) > 0 {
		args := ensureTransformOrder(state, prereqs, node.Arguments(), nil)
		return luau.Call(luau.Globals.Table.Create, args...)
	}
	return luau.NewArray()
}

func SetConstructor(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	if len(node.Arguments()) == 0 {
		return luau.NewSet()
	}
	arg := node.Arguments()[0]
	// spreads cause prereq array, which cannot be optimised like this
	if ast.IsArrayLiteralExpression(arg) && !slices.ContainsFunc(arg.Elements(), ast.IsSpreadElement) {
		return luau.NewSet(ensureTransformOrder(state, prereqs, arg.Elements(), nil)...)
	} else {
		id := prereqs.PushToVar(luau.NewSet(), "set")
		valueId := luau.TempId("v")
		prereqs.Push(
			luau.Create(&luau.ForStatement{
				Ids:        luau.MakeList(luau.TempId(), valueId),
				Expression: transformExpression(state, prereqs, arg),
				Statements: luau.MakeList(
					luau.Create(&luau.Assignment{
						Left: luau.Create(&luau.ComputedIndexExpression{
							Expression: id,
							Index:      valueId,
						}),
						Operator: "=",
						Right:    luau.Bool(true),
					}),
				),
			}),
		)
		return id
	}
}

func MapConstructor(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	if len(node.Arguments()) == 0 {
		return luau.NewMap()
	}
	arg := node.Arguments()[0]
	transformed := transformExpression(state, prereqs, arg)
	if array, ok := transformed.(*luau.Array); ok && array.Members.Every(func(member luau.Node) bool { return luau.IsArray(member) }) {
		elements := luau.ListMapToArray(array.Members, func(e luau.Node) luau.MapEntry {
			// non-null and type assertion because array will always have 2 members,
			// due to map constructor typing
			eArray, ok := e.(*luau.Array)
			assert(ok && eArray.Members.IsNonEmpty())
			return luau.MapEntry{Index: eArray.Members.Head.Value, Value: eArray.Members.Head.Next.Value}
		})
		return luau.NewMap(elements...)
	} else {
		id := prereqs.PushToVar(luau.NewMap(), "map")
		valueId := luau.TempId("v")
		prereqs.Push(
			luau.Create(&luau.ForStatement{
				Ids:        luau.MakeList(luau.TempId(), valueId),
				Expression: transformed,
				Statements: luau.MakeList(
					luau.Create(&luau.Assignment{
						Left: luau.Create(&luau.ComputedIndexExpression{
							Expression: id,
							Index: luau.Create(&luau.ComputedIndexExpression{
								Expression: valueId,
								Index:      luau.Number(1),
							}),
						}),
						Operator: "=",
						Right: luau.Create(&luau.ComputedIndexExpression{
							Expression: valueId,
							Index:      luau.Number(2),
						}),
					}),
				),
			}),
		)
		return id
	}
}

var CONSTRUCTOR_MACROS = MacroList[ConstructorMacro]{
	"ArrayConstructor": ArrayConstructor,
	"SetConstructor":   SetConstructor,
	"MapConstructor":   MapConstructor,
	"WeakSetConstructor": func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
		return wrapWeak(state, prereqs, node, SetConstructor)
	},
	"WeakMapConstructor": func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
		return wrapWeak(state, prereqs, node, MapConstructor)
	},
	"ReadonlyMapConstructor": MapConstructor,
	"ReadonlySetConstructor": SetConstructor,
}
