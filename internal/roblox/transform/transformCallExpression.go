package transform

// Port of TSTransformer/nodes/expressions/transformCallExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

/**
 * Some C functions like `tonumber()` will error if the given argument is a function that returns nothing.
 * i.e.
 * ```lua
 * local function foo()
 * end
 * local x = tonumber(foo()) -- error!
 * ```
 *
 * To protect against this, we can wrap possibly-undefined arguments with `()` to coerce the values to `nil`
 */
func fixVoidArgumentsForRobloxFunctions(state *TransformState, type_ *checker.Type, args []luau.Expression, nodeArguments []*ast.Node) {
	if getOrSetDefault(state.MultiTransformState.IsPossiblyRobloxTypeCache, type_, func() bool {
		return isPossiblyType(type_, isRobloxType(state))
	}) {
		for i := range args {
			arg := args[i]
			if i >= len(nodeArguments) {
				continue
			}
			nodeArg := nodeArguments[i]
			if ast.IsCallExpression(nodeArg) && isPossiblyType(state.GetType(nodeArg), isUndefinedType) {
				args[i] = luau.Create(&luau.ParenthesizedExpression{
					Expression: arg,
				})
			}
		}
	}
}

// transformCallExpressionInner is TSTransformer/nodes/expressions/transformCallExpression.ts transformCallExpressionInner.
func transformCallExpressionInner(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, nodeArguments []*ast.Node) luau.Expression {
	if ast.IsImportCall(node) {
		return transformImportExpression(state, node)
	}

	// a in a()
	validateNotAnyType(state, node.Expression())

	if ast.IsSuperCall(node) {
		return luau.Call(luau.Property(convertToIndexableExpression(expression), "constructor"), append(
			[]luau.Expression{luau.Globals.Self},
			ensureTransformOrder(state, prereqs, node.Arguments(), nil)...,
		)...)
	}

	expType := state.TypeChecker.GetNonOptionalType(state.GetType(node.Expression()))
	symbol := getFirstDefinedSymbol(state, expType)
	if symbol != nil {
		macro := state.Services.MacroManager.GetCallMacro(symbol)
		if macro != nil {
			return transformMacroCall(macro, state, prereqs, node, expression, nodeArguments)
		}
	}

	argsPrereqs := NewPrereqs()
	args := ensureTransformOrder(state, argsPrereqs, nodeArguments, nil)
	fixVoidArgumentsForRobloxFunctions(state, expType, args, nodeArguments)

	if !effectsCommute(getEffects(expression), getStatementsEffects(argsPrereqs.Statements)) {
		expression = prereqs.PushToVar(expression, "fn")
	}
	prereqs.PushList(argsPrereqs.Statements)

	if isMethodFromType(state, node.Expression(), expType) {
		// direct calls have no receiver, but the implementation still expects its slot
		args = append([]luau.Expression{luau.Nil()}, args...)
	}

	exp := luau.Call(convertToIndexableExpression(expression), args...)

	return wrapReturnIfLuaTuple(state, node, exp)
}

// createOrderedPropertyCall is transformCallExpression.ts createOrderedPropertyCall. `key: string |
// luau.Expression` is keyName when keyExpression is nil.
func createOrderedPropertyCall(
	state *TransformState,
	prereqs *Prereqs,
	source *ast.Node,
	base luau.Expression,
	keyName string,
	keyExpression luau.Expression,
	args []luau.Expression,
	prereqStatements *luau.List,
	method bool,
) luau.Expression {
	property := func() luau.IndexableExpression {
		var expression luau.IndexableExpression
		if keyExpression == nil {
			expression = luau.Property(convertToIndexableExpression(base), keyName)
		} else {
			expression = luau.Create(&luau.ComputedIndexExpression{
				Expression: convertToIndexableExpression(base),
				Index:      keyExpression,
			})
		}
		tryMarkBuiltinMember(state, source, expression)
		return expression
	}
	prereqEffects := getStatementsEffects(prereqStatements)
	name, hasName := keyName, keyExpression == nil
	if !hasName {
		if stringLiteral, ok := keyExpression.(*luau.StringLiteral); ok {
			name, hasName = stringLiteral.Value, true
		}
	}
	stableMethod := method && isStableBuiltinMember(state, source)
	var callee luau.IndexableExpression = property()
	lookupEffects := getEffects(callee)
	captureCallee := !effectsCommute(lookupEffects, prereqEffects)
	if method &&
		hasName &&
		luau.IsValidIdentifier(name) &&
		(stableMethod || (!captureCallee && keyExpression == nil)) {
		// a stable method lookup can stay inline even when its receiver needs a snapshot
		if stableMethod && !effectsCommute(getEffects(base), prereqEffects) {
			base = prereqs.PushToVar(base, "self")
		}
		prereqs.PushList(prereqStatements)
		methodArgs := make([]luau.Node, len(args))
		for i, arg := range args {
			methodArgs[i] = arg
		}
		return luau.Create(&luau.MethodCallExpression{
			Expression: convertToIndexableExpression(base),
			Name:       name,
			Args:       luau.MakeList(methodArgs...),
		})
	}
	// an explicit self argument must denote the receiver from before the lookup
	if method &&
		(!luau.IsSimple(base) || !effectsCommute(getEffects(base), joinEffects(lookupEffects, prereqEffects))) {
		base = prereqs.PushToVar(base, "self")
		callee = property()
	}
	if captureCallee {
		callee = prereqs.PushToVar(callee, "fn")
	}
	prereqs.PushList(prereqStatements)
	if method {
		return luau.Call(callee, append([]luau.Expression{base}, args...)...)
	}
	return luau.Call(callee, args...)
}

// transformPropertyCallExpressionInner is TSTransformer/nodes/expressions/transformCallExpression.ts transformPropertyCallExpressionInner.
func transformPropertyCallExpressionInner(state *TransformState, prereqs *Prereqs, node *ast.Node, expression *ast.Node, baseExpression luau.Expression, name string, nodeArguments []*ast.Node) luau.Expression {
	// a in a.b()
	validateNotAnyType(state, expression.Expression())
	// a.b in a.b()
	validateNotAnyType(state, node.Expression())

	if ast.IsSuperProperty(expression) {
		callee := luau.Property(convertToIndexableExpression(baseExpression), expression.Name().Text())
		args := ensureTransformOrder(state, prereqs, node.Arguments(), nil)
		if isMethod(state, expression) {
			args = append([]luau.Expression{luau.Globals.Self}, args...)
		}
		return luau.Call(callee, args...)
	}

	expType := state.TypeChecker.GetNonOptionalType(state.GetType(node.Expression()))
	symbol := getFirstDefinedSymbol(state, expType)
	if symbol != nil {
		macro := state.Services.MacroManager.GetPropertyCallMacro(symbol)
		if macro != nil {
			return transformMacroCall(macro, state, prereqs, node, baseExpression, nodeArguments)
		}
	}

	argsPrereqs := NewPrereqs()
	args := ensureTransformOrder(state, argsPrereqs, nodeArguments, nil)
	fixVoidArgumentsForRobloxFunctions(state, expType, args, nodeArguments)

	exp := createOrderedPropertyCall(
		state,
		prereqs,
		expression,
		baseExpression,
		name,
		nil,
		args,
		argsPrereqs.Statements,
		isMethod(state, expression),
	)
	return wrapReturnIfLuaTuple(state, node, exp)
}

// transformElementCallExpressionInner is TSTransformer/nodes/expressions/transformCallExpression.ts transformElementCallExpressionInner.
func transformElementCallExpressionInner(state *TransformState, prereqs *Prereqs, node *ast.Node, expression *ast.Node, baseExpression luau.Expression, argumentExpression *ast.Node, nodeArguments []*ast.Node) luau.Expression {
	elementAccess := expression.AsElementAccessExpression()
	// a in a[b]()
	validateNotAnyType(state, elementAccess.Expression)
	// b in a[b]()
	validateNotAnyType(state, elementAccess.ArgumentExpression)
	// a[b] in a[b]()
	validateNotAnyType(state, node.Expression())

	if ast.IsSuperProperty(expression) {
		callee := luau.Create(&luau.ComputedIndexExpression{
			Expression: convertToIndexableExpression(baseExpression),
			Index:      transformExpression(state, prereqs, elementAccess.ArgumentExpression),
		})
		args := ensureTransformOrder(state, prereqs, node.Arguments(), nil)
		if isMethod(state, expression) {
			args = append([]luau.Expression{luau.Globals.Self}, args...)
		}
		return luau.Call(callee, args...)
	}

	expType := state.TypeChecker.GetNonOptionalType(state.GetType(node.Expression()))
	symbol := getFirstDefinedSymbol(state, expType)
	if symbol != nil {
		macro := state.Services.MacroManager.GetPropertyCallMacro(symbol)
		if macro != nil {
			return transformMacroCall(macro, state, prereqs, node, baseExpression, nodeArguments)
		}
	}

	keyPrereqs := NewPrereqs()
	argumentExp := transformExpression(state, keyPrereqs, argumentExpression)
	lateReadEffects := NO_EFFECTS
	if isLateRead(baseExpression) {
		lateReadEffects = getEffects(argumentExp)
	}
	if !effectsCommute(
		getEffects(baseExpression),
		joinEffects(
			getStatementsEffects(keyPrereqs.Statements),
			lateReadEffects,
		),
	) {
		baseExpression = prereqs.PushToVar(baseExpression, "exp")
	}
	prereqs.PushList(keyPrereqs.Statements)
	argsPrereqs := NewPrereqs()
	args := ensureTransformOrder(state, argsPrereqs, nodeArguments, nil)
	fixVoidArgumentsForRobloxFunctions(state, expType, args, nodeArguments)
	key := addOneIfArrayType(
		state,
		state.TypeChecker.GetNonOptionalType(state.GetType(elementAccess.Expression)),
		argumentExp,
	)
	exp := createOrderedPropertyCall(
		state,
		prereqs,
		expression,
		baseExpression,
		"",
		key,
		args,
		argsPrereqs.Statements,
		isMethod(state, expression),
	)
	return wrapReturnIfLuaTuple(state, node, exp)
}

// transformCallExpression is TSTransformer/nodes/expressions/transformCallExpression.ts transformCallExpression.
func transformCallExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	return transformOptionalChain(state, prereqs, node)
}
