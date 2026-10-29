package transform

// Port of TSTransformer/nodes/statements/transformEnumDeclaration.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/jsnum"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func needsInverseEntry(state *TransformState, member *ast.Node) bool {
	_, isString := state.TypeChecker.GetConstantValue(member).(string)
	return !isString
}

// transformEnumDeclaration is TSTransformer/nodes/statements/transformEnumDeclaration.ts transformEnumDeclaration.
func transformEnumDeclaration(state *TransformState, node *ast.Node) *luau.List {
	if ast.HasSyntacticModifier(node, ast.ModifierFlagsConst) && !state.CompilerOptions.PreserveConstEnums.IsTrue() {
		return luau.MakeList()
	}

	symbol := state.TypeChecker.GetSymbolAtLocation(node.Name())
	if symbol != nil &&
		hasMultipleDefinitions(
			symbol,
			func(declaration *ast.Node) bool {
				return ast.IsEnumDeclaration(declaration) && !ast.HasSyntacticModifier(declaration, ast.ModifierFlagsConst)
			},
		) {
		AddDiagnosticWithCache(
			symbol,
			diagnostics.Errors.NoEnumMerging.Create(node),
			state.MultiTransformState.IsReportedByMultipleDefinitionsCache,
		)
		return luau.MakeList()
	}

	validateIdentifier(node.Name())

	left := transformIdentifierDefined(state, node.Name())
	isHoisted := symbol != nil && state.IsHoisted[symbol]
	statements := luau.MakeList()
	members := node.AsEnumDeclaration().Members.Nodes

	everyMemberSkipsInverse := true
	for _, member := range members {
		if needsInverseEntry(state, member) {
			everyMemberSkipsInverse = false
			break
		}
	}
	if everyMemberSkipsInverse {
		prereqs := NewPrereqs()
		fields := make([]luau.MapEntry, len(members))
		for i, member := range members {
			fields[i] = luau.MapEntry{
				Index: prereqs.PushToVarIfComplex(transformPropertyName(state, prereqs, member.Name()), ""),
				Value: luau.String(state.TypeChecker.GetConstantValue(member).(string)),
			}
		}
		right := luau.NewMap(fields...)
		statements.PushList(prereqs.Statements)
		if isHoisted {
			statements.Push(luau.Create(&luau.Assignment{Left: left, Operator: "=", Right: right}))
		} else {
			statements.Push(luau.Create(&luau.VariableDeclaration{Left: left, Right: right}))
		}
		return statements
	}

	enumStatements := luau.MakeList()
	inverseId := luau.TempId("inverse")
	enumStatements.Push(luau.Create(&luau.VariableDeclaration{Left: inverseId, Right: luau.NewMap()}))
	enumStatements.Push(luau.Create(&luau.Assignment{
		Left:     left,
		Operator: "=",
		Right:    luau.Call(luau.Globals.Setmetatable, luau.NewMap(), luau.NewMap(luau.MapEntry{Index: luau.Strings.Index, Value: inverseId})),
	}))

	for _, member := range members {
		memberPrereqs := NewPrereqs()
		memberName := member.Name()
		name := transformPropertyName(state, memberPrereqs, memberName)
		mutationNode := memberName
		if ast.IsComputedPropertyName(memberName) {
			mutationNode = memberName.Expression()
		}
		var index luau.Expression = name
		if expressionMightMutate(state, name, mutationNode) {
			// note: we don't use pushToVarIfComplex here
			// because identifier also needs to be pushed
			// since the value calculation might reassign the variable
			index = memberPrereqs.PushToVar(name, "")
		}

		var valueExp luau.Expression
		switch value := state.TypeChecker.GetConstantValue(member).(type) {
		case string:
			valueExp = luau.String(value)
		case jsnum.Number:
			valueExp = luau.Number(float64(value))
		default:
			// constantValue is always number without initializer, so assert is safe
			initializer := member.Initializer()
			assert(initializer != nil)
			valueExp = memberPrereqs.PushToVarIfComplex(
				transformExpression(state, memberPrereqs, initializer),
				"value",
			)
		}

		enumStatements.PushList(memberPrereqs.Statements)
		enumStatements.Push(luau.Create(&luau.Assignment{
			Left: luau.Create(&luau.ComputedIndexExpression{
				Expression: left,
				Index:      index,
			}),
			Operator: "=",
			Right:    valueExp,
		}))

		if needsInverseEntry(state, member) {
			enumStatements.Push(luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: inverseId,
					Index:      valueExp,
				}),
				Operator: "=",
				Right:    index,
			}))
		}
	}

	statements.Push(luau.Create(&luau.DoStatement{Statements: enumStatements}))
	if !isHoisted {
		statements.Unshift(luau.Create(&luau.VariableDeclaration{Left: left, Right: nil}))
	}
	return statements
}
