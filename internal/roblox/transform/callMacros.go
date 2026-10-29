package transform

// Port of TSTransformer/macros/callMacros.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

var PRIMITIVE_LUAU_TYPES = map[string]bool{
	"nil":      true,
	"boolean":  true,
	"string":   true,
	"number":   true,
	"table":    true,
	"userdata": true,
	"function": true,
	"thread":   true,
	"vector":   true,
	"buffer":   true,
}

var CALL_MACROS = MacroList[CallMacro]{
	"assert": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		args[0] = createTruthinessChecks(state, prereqs, args[0], node.Arguments()[0])
		return luau.Call(luau.Globals.Assert, args...)
	},

	"typeOf": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		return luau.Call(luau.Globals.Typeof, args...)
	},

	"typeIs": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		value, typeStr := args[0], args[1]
		var typeFunc luau.IndexableExpression = luau.Globals.Typeof
		if stringLiteral, ok := typeStr.(*luau.StringLiteral); ok && PRIMITIVE_LUAU_TYPES[stringLiteral.Value] {
			typeFunc = luau.Globals.Type
		}
		return luau.Binary(luau.Call(typeFunc, value), "==", typeStr)
	},

	"classIs": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		value, typeStr := args[0], args[1]
		return luau.Binary(luau.Property(convertToIndexableExpression(value), "ClassName"), "==", typeStr)
	},

	"identity": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		return args[0]
	},

	"$range": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoRangeMacroOutsideForOf.Create(node.Expression()))
		return luau.NewNone()
	},

	"$tuple": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoTupleMacroOutsideReturn.Create(node))
		return luau.NewNone()
	},

	"$getModuleTree": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		parts := getImportParts(state, ast.GetSourceFileOfNode(node), node.Arguments()[0])
		// converts the flat array into { root, { "rest", "of", "path" } }
		return luau.NewArray(parts[0], luau.NewArray(parts[1:]...))
	},
}
