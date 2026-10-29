package transform

// Port of TSTransformer/nodes/transformWritable.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformWritableExpression is TSTransformer/nodes/transformWritable.ts transformWritableExpression.
func transformWritableExpression(state *TransformState, prereqs *Prereqs, node *ast.Node, readAfterWrite bool) luau.WritableExpression {
	if ast.IsPrototypeAccess(node) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoPrototype.Create(node))
	}
	if ast.IsPropertyAccessExpression(node) {
		expression := transformExpression(state, prereqs, node.Expression())
		var base luau.IndexableExpression
		if readAfterWrite {
			base = prereqs.PushToVarIfNonId(expression, "exp")
		} else {
			base = convertToIndexableExpression(expression)
		}
		return luau.Property(base, node.Name().Text())
	} else if ast.IsElementAccessExpression(node) {
		transformed := ensureTransformOrder(state, prereqs, []*ast.Node{node.Expression(), node.AsElementAccessExpression().ArgumentExpression}, nil)
		expression, index := transformed[0], transformed[1]
		if isLateRead(expression) && !effectsCommute(getEffects(expression), getEffects(index)) {
			expression = prereqs.PushToVar(expression, "exp")
		}
		indexExp := addOneIfArrayType(state, state.GetType(node.Expression()), index)
		var base luau.Expression
		if readAfterWrite {
			base = prereqs.PushToVarIfNonId(expression, "exp")
		} else {
			base = convertToIndexableExpression(expression)
		}
		if readAfterWrite {
			indexExp = prereqs.PushToVarIfComplex(indexExp, "index")
		}
		return luau.Create(&luau.ComputedIndexExpression{
			Expression: base,
			Index:      indexExp,
		})
	} else {
		transformed := transformExpression(state, prereqs, skipDownwards(node))
		assert(luau.IsWritableExpression(transformed))
		return transformed
	}
}

// transformWritableAssignment is TSTransformer/nodes/transformWritable.ts transformWritableAssignment.
func transformWritableAssignment(state *TransformState, prereqs *Prereqs, writeNode *ast.Node, valueNode *ast.Node, readAfterWrite bool, readBeforeWrite bool) (writable luau.WritableExpression, readable luau.WritableExpression, value luau.Expression) {
	writable = transformWritableExpression(state, prereqs, writeNode, readAfterWrite)
	valuePrereqs := NewPrereqs()
	value = transformExpression(state, valuePrereqs, valueNode)
	prereqEffects := getEffectsOfListOrNode(valuePrereqs.Statements)
	valueEffects := getEffects(value)
	effects := joinEffects(prereqEffects, valueEffects)
	writable = captureWritableAssignmentTarget(prereqs, writable, prereqEffects, valueEffects)
	// the assignment target and its old value may need separate snapshots
	if readBeforeWrite && !effectsCommute(getEffects(writable), effects) {
		readable = prereqs.PushToVar(writable, "readable")
	} else {
		readable = writable
	}
	prereqs.PushList(valuePrereqs.Statements)

	return writable, readable, value
}

// captureWritableAssignmentTarget is TSTransformer/nodes/transformWritable.ts captureWritableAssignmentTarget.
func captureWritableAssignmentTarget(prereqs *Prereqs, writable luau.WritableExpression, prereqEffects *EvaluationEffects, valueEffects *EvaluationEffects) luau.WritableExpression {
	// Luau evaluates complex bases and keys before an inline RHS, but can read
	// locals at the store instruction; hoisted prerequisites precede both
	intervening := func(expression luau.Expression) *EvaluationEffects {
		if isLateRead(expression) {
			return joinEffects(prereqEffects, valueEffects)
		}
		return joinEffects(prereqEffects, NO_EFFECTS)
	}
	if !luau.IsAnyIdentifier(writable) {
		var base, index luau.Expression
		switch w := writable.(type) {
		case *luau.ComputedIndexExpression:
			base, index = w.Expression, w.Index
		case *luau.PropertyAccessExpression:
			base = w.Expression
		default:
			panic("captureWritableAssignmentTarget: unexpected writable " + writable.Kind().String())
		}
		captureIndex := index != nil && !effectsCommute(getEffects(index), intervening(index))
		var baseEffects *EvaluationEffects
		if captureIndex {
			baseEffects = joinEffects(intervening(base), getEffects(index))
		} else {
			baseEffects = intervening(base)
		}
		stableBase := base
		if !effectsCommute(getEffects(base), baseEffects) {
			stableBase = prereqs.PushToVar(base, "exp")
		}
		if w, ok := writable.(*luau.ComputedIndexExpression); ok {
			newIndex := w.Index
			if captureIndex {
				newIndex = prereqs.PushToVar(w.Index, "index")
			}
			writable = luau.Create(&luau.ComputedIndexExpression{
				Expression: stableBase,
				Index:      newIndex,
			})
		} else {
			writable = luau.Property(stableBase, writable.(*luau.PropertyAccessExpression).Name)
		}
	}
	return writable
}
