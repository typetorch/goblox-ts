package transform

// Port of TSTransformer/nodes/transformMacroCall.ts.

import (
	"fmt"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformMacroCall is TSTransformer/nodes/transformMacroCall.ts transformMacroCall.
func transformMacroCall(macro CallMacro, state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, nodeArguments []*ast.Node) luau.Expression {
	operands := []*EvaluationOperand{{Expression: expression, Prereqs: luau.MakeList()}}
	for _, argument := range nodeArguments {
		valuePrereqs := NewPrereqs()
		value := transformExpression(state, valuePrereqs, argument)
		if !ast.IsSpreadElement(argument) {
			// scalar arguments must supply one nil even when an inlined call returns no values
			if luau.IsCall(value) && isPossiblyType(state.GetType(argument), isUndefinedType) {
				value = luau.Create(&luau.ParenthesizedExpression{Expression: value})
			}
			operands = append(operands, &EvaluationOperand{Expression: value, Prereqs: valuePrereqs.Statements})
			continue
		}
		signature := state.TypeChecker.GetResolvedSignature(node)
		assert(signature != nil)
		var parameter *ast.Node
		if parameters := signature.Parameters(); len(parameters) > 0 {
			parameter = parameters[len(parameters)-1].ValueDeclaration
		}
		if parameter != nil && ast.IsParameterDeclaration(parameter) && parameter.AsParameterDeclaration().DotDotDotToken != nil {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoVarArgsMacroSpread.Create(argument))
			return luau.NewNone()
		}
		type_ := state.GetType(argument.Expression())
		assert(checker.IsTupleType(type_))
		count := len(type_.TargetTupleType().ElementFlags())
		ids := make([]luau.Node, count)
		for i := range ids {
			ids[i] = luau.TempId(fmt.Sprintf("spread%d", i))
		}
		if len(ids) == 0 {
			// an empty spread may still have effects
			valuePrereqs.Push(luau.Create(&luau.VariableDeclaration{Left: luau.TempId(), Right: value}))
			operands = append(operands, &EvaluationOperand{Expression: luau.NewNone(), Prereqs: valuePrereqs.Statements})
		} else {
			valuePrereqs.Push(luau.Create(&luau.VariableDeclaration{Left: luau.MakeList(ids...), Right: value}))
			for i, id := range ids {
				operandPrereqs := luau.MakeList()
				if i == 0 {
					operandPrereqs = valuePrereqs.Statements
				}
				operands = append(operands, &EvaluationOperand{Expression: id.(luau.Expression), Prereqs: operandPrereqs})
			}
		}
	}
	result := planEvaluation(prereqs, operands, func(expansionPrereqs *Prereqs, operands []luau.Expression) luau.Expression {
		receiver, args := operands[0], []luau.Expression{}
		for _, arg := range operands[1:] {
			if !luau.IsNone(arg) {
				args = append(args, arg)
			}
		}
		return macro(state, expansionPrereqs, node, receiver, args)
	})
	return wrapReturnIfLuaTuple(state, node, result)
}
