package transform

// Port of TSTransformer/nodes/expressions/transformBinaryExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// rhs is a luau.Expression or a *luau.List of them.
func transformOptimizedArrayAssignmentPattern(
	state *TransformState,
	prereqs *Prereqs,
	assignmentPattern *ast.Node,
	rhs luau.ListOrNode,
) {
	variables := luau.MakeList()
	writes := luau.MakeList()
	writesPrereqs := luau.MakeList()
	bindingPrereqs := NewPrereqs()
	for _, element := range assignmentPattern.Elements() {
		if ast.IsOmittedExpression(element) {
			writes.Push(luau.TempId())
		} else {
			// callers only select this optimization for patterns without spread elements
			assert(!ast.IsSpreadElement(element), "Cannot optimize-assign spread element")
			var initializer *ast.Node
			if ast.IsBinaryExpression(element) {
				initializer = skipDownwards(element.AsBinaryExpression().Right)
				element = skipDownwards(element.AsBinaryExpression().Left)
			}

			if ast.IsIdentifier(element) ||
				ast.IsElementAccessExpression(element) ||
				ast.IsPropertyAccessExpression(element) {
				idPrereqs := NewPrereqs()
				id := transformWritableExpression(state, idPrereqs, element, true)
				writesPrereqs.PushList(idPrereqs.Statements)
				writes.Push(id)
				if initializer != nil {
					bindingPrereqs.Push(transformInitializer(state, id, initializer))
				}
			} else if ast.IsArrayLiteralExpression(element) {
				id := luau.TempId("binding")
				variables.Push(id)
				writes.Push(id)
				if initializer != nil {
					bindingPrereqs.Push(transformInitializer(state, id, initializer))
				}
				transformArrayAssignmentPattern(state, bindingPrereqs, element, id)
			} else {
				assert(ast.IsObjectLiteralExpression(element), "Expected object assignment pattern")
				id := luau.TempId("binding")
				variables.Push(id)
				writes.Push(id)
				if initializer != nil {
					bindingPrereqs.Push(transformInitializer(state, id, initializer))
				}
				transformObjectAssignmentPattern(state, bindingPrereqs, element, id)
			}
		}
	}
	if !variables.IsEmpty() {
		prereqs.Push(
			luau.Create(&luau.VariableDeclaration{
				Left:  variables,
				Right: nil,
			}),
		)
	}

	targetEffects := getStatementsEffects(writesPrereqs)
	writes.ForEach(func(target luau.Node) {
		switch target := target.(type) {
		case *luau.PropertyAccessExpression:
			// joining NO_EFFECTS (the index of a property access) is the identity
			targetEffects = joinEffects(targetEffects, getEffects(target.Expression))
		case *luau.ComputedIndexExpression:
			targetEffects = joinEffects(targetEffects, getEffects(target.Expression), getEffects(target.Index))
		}
	})

	// the complete RHS precedes destination evaluation, including newly hoisted captures
	if rhsList, ok := rhs.(*luau.List); ok {
		expressions := rhsList.ToArray()
		captures := make([]bool, len(expressions))
		for i := len(expressions) - 1; i >= 0; i-- {
			effects := getEffects(expressions[i])
			captures[i] = !effectsCommute(effects, targetEffects)
			if captures[i] {
				targetEffects = joinEffects(effects, targetEffects)
			}
		}
		var flattened []luau.Node
		for index, expression := range expressions {
			if !captures[index] {
				flattened = append(flattened, expression)
				continue
			}

			// a final array member can supply multiple return values to the remaining destinations
			if index == len(expressions)-1 && luau.IsCall(expression) {
				count := writes.Size() - index
				if count > 1 {
					ids := make([]luau.Node, count)
					for j := range ids {
						ids[j] = luau.TempId("binding")
					}
					prereqs.Push(
						luau.Create(&luau.VariableDeclaration{
							Left:  luau.MakeList(ids...),
							Right: expression,
						}),
					)
					flattened = append(flattened, ids...)
					continue
				}
			}

			flattened = append(flattened, prereqs.PushToVar(expression, "binding"))
		}
		rhs = luau.MakeList(flattened...)
	} else if !effectsCommute(getEffects(rhs.(luau.Node)), targetEffects) {
		captures := luau.MakeList(luau.ListMapToArray(writes, func(luau.Node) luau.Node { return luau.TempId("binding") })...)
		prereqs.Push(luau.Create(&luau.VariableDeclaration{Left: captures, Right: rhs}))
		rhs = captures
	}

	prereqs.PushList(writesPrereqs)
	assert(!writes.IsEmpty())
	prereqs.Push(
		luau.Create(&luau.Assignment{
			Left:     writes,
			Operator: "=",
			Right:    rhs,
		}),
	)
	prereqs.PushList(bindingPrereqs.Statements)
}

func tryTransformOptimizedObjectAssignmentPattern(
	state *TransformState,
	prereqs *Prereqs,
	assignmentPattern *ast.Node,
	rhs luau.Expression,
) bool {
	properties := assignmentPattern.Properties()
	if len(properties) != 1 {
		return false
	}

	property := properties[0]
	if !(ast.IsShorthandPropertyAssignment(property) || ast.IsPropertyAssignment(property)) ||
		!ast.IsIdentifier(property.Name()) {
		return false
	}

	var target, initializer *ast.Node
	if ast.IsShorthandPropertyAssignment(property) {
		target = property.Name()
		initializer = property.AsShorthandPropertyAssignment().ObjectAssignmentInitializer
	} else {
		target = property.Initializer()
	}
	if ast.IsBinaryExpression(target) {
		initializer = skipDownwards(target.AsBinaryExpression().Right)
		target = skipDownwards(target.AsBinaryExpression().Left)
	}

	// property and indexed targets can evaluate before an inline receiver
	if !ast.IsIdentifier(target) {
		return false
	}

	_, value := objectAccessor(
		state,
		prereqs,
		convertToIndexableExpression(rhs),
		state.TypeChecker.GetTypeOfAssignmentPattern(assignmentPattern),
		property.Name(),
	)
	writable := transformWritableExpression(state, prereqs, target, initializer != nil)
	prereqs.Push(luau.Create(&luau.Assignment{Left: writable, Operator: "=", Right: value}))
	if initializer != nil {
		prereqs.Push(transformInitializer(state, writable, initializer))
	}

	return true
}

// transformBinaryExpression is TSTransformer/nodes/expressions/transformBinaryExpression.ts transformBinaryExpression.
func transformBinaryExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	binary := node.AsBinaryExpression()
	operatorKind := binary.OperatorToken.Kind

	validateNotAnyType(state, binary.Left)
	validateNotAnyType(state, binary.Right)

	// banned
	if operatorKind == ast.KindEqualsEqualsToken {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoEqualsEquals.Create(node))
		return luau.NewNone()
	} else if operatorKind == ast.KindExclamationEqualsToken {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoExclamationEquals.Create(node))
		return luau.NewNone()
	}

	// logical
	if operatorKind == ast.KindAmpersandAmpersandToken ||
		operatorKind == ast.KindBarBarToken ||
		operatorKind == ast.KindQuestionQuestionToken {
		return transformLogical(state, prereqs, node)
	}

	if ast.IsLogicalOrCoalescingAssignmentExpression(node) {
		return transformLogicalOrCoalescingAssignmentExpression(state, prereqs, node)
	}

	if ast.IsAssignmentOperator(operatorKind) {
		// in destructuring, rhs must be executed first
		if ast.IsArrayLiteralExpression(binary.Left) {
			rightExp := transformExpression(state, prereqs, binary.Right)

			// optimize empty array destructure
			if len(binary.Left.Elements()) == 0 {
				if isUsedAsStatement(node) && luau.IsArray(rightExp) && rightExp.(*luau.Array).Members.IsEmpty() {
					return luau.NewNone()
				}
				return rightExp
			}

			if luau.IsCall(rightExp) &&
				isLuaTupleType(state)(state.GetType(binary.Right)) &&
				!arrayLikeExpressionContainsSpread(binary.Left) {
				transformOptimizedArrayAssignmentPattern(state, prereqs, binary.Left, rightExp)
				if !isUsedAsStatement(node) {
					DiagnosticService.AddDiagnostic(diagnostics.Errors.NoLuaTupleDestructureAssignmentExpression.Create(node))
				}
				return luau.NewNone()
			}

			if luau.IsArray(rightExp) &&
				!rightExp.(*luau.Array).Members.IsEmpty() &&
				isUsedAsStatement(node) &&
				!arrayLikeExpressionContainsSpread(binary.Left) {
				transformOptimizedArrayAssignmentPattern(state, prereqs, binary.Left, rightExp.(*luau.Array).Members)
				return luau.NewNone()
			}

			parentId := prereqs.PushToVar(rightExp, "binding")
			transformArrayAssignmentPattern(state, prereqs, binary.Left, parentId)
			return parentId
		} else if ast.IsObjectLiteralExpression(binary.Left) {
			rightExp := transformExpression(state, prereqs, binary.Right)

			// optimize empty object destructure
			if len(binary.Left.Properties()) == 0 {
				if isUsedAsStatement(node) && luau.IsMap(rightExp) && rightExp.(*luau.Map).Fields.IsEmpty() {
					return luau.NewNone()
				}
				return rightExp
			}

			// a used assignment expression must still return the original receiver
			if isUsedAsStatement(node) &&
				tryTransformOptimizedObjectAssignmentPattern(state, prereqs, binary.Left, rightExp) {
				return luau.NewNone()
			}

			parentId := prereqs.PushToVar(rightExp, "binding")
			transformObjectAssignmentPattern(state, prereqs, binary.Left, parentId)
			return parentId
		}

		writableType := state.GetType(binary.Left)
		valueType := state.GetType(binary.Right)
		// "" is getSimpleAssignmentOperator's undefined
		operator := getSimpleAssignmentOperator(writableType, operatorKind, valueType)
		writable, readable, value := transformWritableAssignment(
			state,
			prereqs,
			binary.Left,
			binary.Right,
			true,
			operator != "=",
		)
		if operator != "" {
			return createAssignmentExpression(
				prereqs,
				writable,
				operator,
				getAssignableValue(operator, value, valueType),
				readable,
			)
		} else {
			return createCompoundAssignmentExpression(
				prereqs,
				writable,
				writableType,
				readable,
				operatorKind,
				value,
				valueType,
			)
		}
	}

	if isBitwiseOperator(operatorKind) {
		return createBitwiseFromOperator(state, prereqs, operatorKind, node)
	}

	ordered := ensureTransformOrder(state, prereqs, []*ast.Node{binary.Left, binary.Right}, transformExpression)
	left, right := ordered[0], ordered[1]

	if operatorKind == ast.KindInKeyword {
		return luau.Binary(
			luau.Create(&luau.ComputedIndexExpression{
				Expression: convertToIndexableExpression(right),
				Index:      left,
			}),
			"~=",
			luau.Nil(),
		)
	} else if operatorKind == ast.KindInstanceOfKeyword {
		if isPossiblyType(state.GetType(binary.Right), isRobloxType(state)) {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoRobloxSymbolInstanceof.Create(binary.Right))
		}
		return luau.Call(state.TS(node, "instanceof"), left, right)
	}

	leftType := state.GetType(binary.Left)
	rightType := state.GetType(binary.Right)

	if operatorKind == ast.KindLessThanToken ||
		operatorKind == ast.KindLessThanEqualsToken ||
		operatorKind == ast.KindGreaterThanToken ||
		operatorKind == ast.KindGreaterThanEqualsToken {
		if (!isDefinitelyType(leftType, isStringType) && !isDefinitelyType(leftType, isNumberType)) ||
			(!isDefinitelyType(rightType, isStringType) && !isDefinitelyType(leftType, isNumberType)) {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoNonNumberStringRelationOperator.Create(node))
		}
	}

	return createBinaryFromOperator(prereqs, left, leftType, operatorKind, right, rightType)
}
