package transform

// Port of TSTransformer/util/binding/objectAccessor.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// objectAccessor is TSTransformer/util/binding/objectAccessor.ts objectAccessor.
func objectAccessor(state *TransformState, prereqs *Prereqs, parentId luau.IndexableExpression, receiverType *checker.Type, name *ast.Node) (key luau.Expression, value luau.Expression) {
	memberType := state.GetType(name)
	addIndexDiagnostics(state, name, memberType, receiverType)

	// NoSubstitutionTemplateLiteral is part of ts.PropertyName but TS rejects it as a binding key
	// (TS1180/TS1136), so it can never reach here
	assert(!ast.IsNoSubstitutionTemplateLiteral(name))

	if isMixedStringType(receiverType) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoMixedStringIndex.Create(name))
		return luau.NewNone(), luau.NewNone()
	}

	if ast.IsPrivateIdentifier(name) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoPrivateIdentifier.Create(name))
		return luau.NewNone(), luau.NewNone()
	}

	if ast.IsIdentifier(name) {
		return luau.String(name.Text()), luau.Property(parentId, name.Text())
	}

	keyNode := name
	if ast.IsComputedPropertyName(name) {
		keyNode = name.Expression()
	}
	key = transformExpression(state, prereqs, keyNode)
	if ast.IsComputedPropertyName(name) && !luau.IsSimplePrimitive(key) {
		// computed keys are evaluated before assignment targets and reused by object rest
		key = prereqs.PushToVar(key, "key")
	}

	if isDefinitelyType(receiverType, isStringType) {
		var indexType *checker.Type
		if ast.IsStringLiteral(keyNode) {
			indexType = state.TypeChecker.GetStringLiteralType(keyNode.Text())
		} else {
			indexType = state.GetType(keyNode)
		}
		return key, createStringIndexExpression(
			prereqs,
			parentId,
			&EvaluationOperand{Expression: key, Prereqs: luau.MakeList()},
			indexType,
		)
	}

	key = addOneIfArrayType(state, receiverType, key)
	return key, luau.Create(&luau.ComputedIndexExpression{Expression: parentId, Index: key})
}
