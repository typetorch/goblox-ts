package transform

// Port of TSTransformer/nodes/statements/transformForOfStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// loopBuilder is LoopBuilder.
type loopBuilder func(
	state *TransformState,
	prereqs *Prereqs,
	statements *luau.List,
	initializer *ast.Node,
	exp luau.Expression,
) *luau.List

func makeForLoopBuilder(
	callback func(
		state *TransformState,
		prereqs *Prereqs,
		initializer *ast.Node,
		exp luau.Expression,
		ids *luau.List,
		initializers *luau.List,
	) luau.Expression,
) loopBuilder {
	return func(state *TransformState, prereqs *Prereqs, statements *luau.List, name *ast.Node, exp luau.Expression) *luau.List {
		ids := luau.MakeList()
		initializers := luau.MakeList()
		expression := callback(state, prereqs, name, exp, ids, initializers)
		statements.UnshiftList(initializers)
		return luau.MakeList(luau.Create(&luau.ForStatement{Ids: ids, Expression: expression, Statements: statements}))
	}
}

func transformForInitializerExpressionDirect(
	state *TransformState,
	prereqs *Prereqs,
	initializer *ast.Node,
	initializers *luau.List,
	value luau.Expression,
) luau.Expression {
	if ast.IsArrayLiteralExpression(initializer) {
		bindingPrereqs := NewPrereqs()
		parentId := bindingPrereqs.PushToVar(value, "binding")
		transformArrayAssignmentPattern(state, bindingPrereqs, initializer, parentId)
		initializers.PushList(bindingPrereqs.Statements)
		return parentId
	} else if ast.IsObjectLiteralExpression(initializer) {
		bindingPrereqs := NewPrereqs()
		parentId := bindingPrereqs.PushToVar(value, "binding")
		transformObjectAssignmentPattern(state, bindingPrereqs, initializer, parentId)
		initializers.PushList(bindingPrereqs.Statements)
		return parentId
	} else {
		expression := transformWritableExpression(state, prereqs, initializer, false)
		initializers.Push(luau.Create(&luau.Assignment{
			Left:     expression,
			Operator: "=",
			Right:    value,
		}))
	}
	return nil
}

func transformForInitializer(
	state *TransformState,
	prereqs *Prereqs,
	initializer *ast.Node,
	initializers *luau.List,
) luau.AnyIdentifier {
	if ast.IsVariableDeclarationList(initializer) {
		return transformBindingName(state, initializer.AsVariableDeclarationList().Declarations.Nodes[0].Name(), initializers)
	} else if ast.IsArrayLiteralExpression(initializer) {
		parentId := luau.TempId("binding")
		bindingPrereqs := NewPrereqs()
		transformArrayAssignmentPattern(state, bindingPrereqs, initializer, parentId)
		initializers.PushList(bindingPrereqs.Statements)
		return parentId
	} else if ast.IsObjectLiteralExpression(initializer) {
		parentId := luau.TempId("binding")
		bindingPrereqs := NewPrereqs()
		transformObjectAssignmentPattern(state, bindingPrereqs, initializer, parentId)
		initializers.PushList(bindingPrereqs.Statements)
		return parentId
	} else {
		valueId := luau.TempId("v")
		expression := transformWritableExpression(state, prereqs, initializer, false)
		initializers.Push(luau.Create(&luau.Assignment{
			Left:     expression,
			Operator: "=",
			Right:    valueId,
		}))
		return valueId
	}
}

var buildArrayLoop loopBuilder = makeForLoopBuilder(func(state *TransformState, prereqs *Prereqs, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
	ids.Push(luau.TempId())
	ids.Push(transformForInitializer(state, prereqs, initializer, initializers))
	return exp
})

var buildSetLoop loopBuilder = makeForLoopBuilder(func(state *TransformState, prereqs *Prereqs, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
	ids.Push(transformForInitializer(state, prereqs, initializer, initializers))
	return exp
})

func transformInLineArrayBindingPattern(
	state *TransformState,
	pattern *ast.Node,
	ids *luau.List,
	initializers *luau.List,
) {
	for _, element := range pattern.AsBindingPattern().Elements.Nodes {
		if isOmittedBindingElement(element) {
			ids.Push(luau.TempId())
		} else {
			id := transformBindingName(state, element.Name(), initializers)
			if initializer := element.Initializer(); initializer != nil {
				initializers.Push(transformInitializer(state, id, initializer))
			}
			ids.Push(id)
		}
	}
}

func transformInLineArrayAssignmentPattern(
	state *TransformState,
	assignmentPattern *ast.Node,
	ids *luau.List,
	initializers *luau.List,
) {
	bindingPrereqs := NewPrereqs()
	for _, element := range assignmentPattern.AsArrayLiteralExpression().Elements.Nodes {
		if ast.IsOmittedExpression(element) {
			ids.Push(luau.TempId())
		} else {
			var initializer *ast.Node
			if ast.IsBinaryExpression(element) {
				initializer = skipDownwards(element.AsBinaryExpression().Right)
				element = skipDownwards(element.AsBinaryExpression().Left)
			}

			valueId := luau.TempId("binding")
			if ast.IsIdentifier(element) ||
				ast.IsElementAccessExpression(element) ||
				ast.IsPropertyAccessExpression(element) {
				id := transformWritableExpression(state, bindingPrereqs, element, initializer != nil)
				bindingPrereqs.Push(luau.Create(&luau.Assignment{
					Left:     id,
					Operator: "=",
					Right:    valueId,
				}))
				if initializer != nil {
					bindingPrereqs.Push(transformInitializer(state, id, initializer))
				}
			} else if ast.IsArrayLiteralExpression(element) {
				if initializer != nil {
					bindingPrereqs.Push(transformInitializer(state, valueId, initializer))
				}
				transformArrayAssignmentPattern(state, bindingPrereqs, element, valueId)
			} else {
				assert(ast.IsObjectLiteralExpression(element), "Expected object assignment pattern")
				if initializer != nil {
					bindingPrereqs.Push(transformInitializer(state, valueId, initializer))
				}
				transformObjectAssignmentPattern(state, bindingPrereqs, element, valueId)
			}

			ids.Push(valueId)
		}
	}
	initializers.PushList(bindingPrereqs.Statements)
}

var buildMapLoop loopBuilder = makeForLoopBuilder(func(state *TransformState, prereqs *Prereqs, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
	// TEST
	if ast.IsVariableDeclarationList(initializer) {
		name := initializer.AsVariableDeclarationList().Declarations.Nodes[0].Name()
		if ast.IsArrayBindingPattern(name) && !arrayLikeExpressionContainsSpread(name) {
			transformInLineArrayBindingPattern(state, name, ids, initializers)
			return exp
		}
	} else if ast.IsArrayLiteralExpression(initializer) && !arrayLikeExpressionContainsSpread(initializer) {
		transformInLineArrayAssignmentPattern(state, initializer, ids, initializers)
		return exp
	}

	keyId := luau.TempId("k")
	valueId := luau.TempId("v")
	ids.Push(keyId)
	ids.Push(valueId)

	if ast.IsVariableDeclarationList(initializer) {
		bindingList := luau.MakeList()
		initializers.Push(luau.Create(&luau.VariableDeclaration{
			Left:  transformForInitializer(state, prereqs, initializer, bindingList),
			Right: luau.NewArray(keyId, valueId),
		}))
		initializers.PushList(bindingList)
	} else {
		transformForInitializerExpressionDirect(
			state,
			prereqs,
			initializer,
			initializers,
			luau.NewArray(keyId, valueId),
		)
	}

	return exp
})

var buildStringLoop loopBuilder = makeForLoopBuilder(func(state *TransformState, prereqs *Prereqs, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
	ids.Push(transformForInitializer(state, prereqs, initializer, initializers))
	return luau.Call(luau.Globals.String.Gmatch, exp, luau.Globals.Utf8.Charpattern)
})

var buildIterableFunctionLoop loopBuilder = makeForLoopBuilder(
	func(state *TransformState, prereqs *Prereqs, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
		ids.Push(transformForInitializer(state, prereqs, initializer, initializers))
		return exp
	},
)

func makeIterableFunctionLuaTupleShorthand(
	state *TransformState,
	array *ast.Node,
	statements *luau.List,
	expression luau.Expression,
) *luau.List {
	ids := luau.MakeList()
	initializers := luau.MakeList()
	if ast.IsArrayBindingPattern(array) {
		transformInLineArrayBindingPattern(state, array, ids, initializers)
	} else {
		transformInLineArrayAssignmentPattern(state, array, ids, initializers)
	}
	statements.UnshiftList(initializers)
	return luau.MakeList(luau.Create(&luau.ForStatement{Ids: ids, Expression: expression, Statements: statements}))
}

func buildIterableFunctionLuaTupleLoop(type_ *checker.Type) loopBuilder {
	return func(state *TransformState, prereqs *Prereqs, statements *luau.List, initializer *ast.Node, exp luau.Expression) *luau.List {
		if ast.IsVariableDeclarationList(initializer) {
			// for (const [a, b] of iter())
			name := initializer.AsVariableDeclarationList().Declarations.Nodes[0].Name()
			if ast.IsArrayBindingPattern(name) && !arrayLikeExpressionContainsSpread(name) {
				return makeIterableFunctionLuaTupleShorthand(state, name, statements, exp)
			}
		} else if ast.IsArrayLiteralExpression(initializer) && !arrayLikeExpressionContainsSpread(initializer) {
			// for ([a, b] of iter())
			return makeIterableFunctionLuaTupleShorthand(state, initializer, statements, exp)
		}

		var iteratorReturnIds []*luau.TemporaryIdentifier

		// get call signature of IterableFunction<T> which is `(): T`
		// and get return type of call signature which is `T`
		luaTupleType := state.TypeChecker.GetReturnTypeOfSignature(typeGetCallSignatures(type_)[0])
		assert(
			luaTupleType != nil && luaTupleType.Alias() != nil && len(luaTupleType.Alias().TypeArguments()) == 1,
			"Incorrect LuaTuple<T> type arguments",
		)
		tupleArgType := luaTupleType.Alias().TypeArguments()[0]
		// if initializer is a variable declaration `for (const a of iter())`
		// and LuaTuple has defined element amount, and no rest elements
		// then use lua for-in loop, specifying all elements and putting them in table
		if ast.IsVariableDeclarationList(initializer) &&
			checker.IsTupleType(tupleArgType) &&
			tupleCombinedFlags(tupleArgType.TargetTupleType())&checker.ElementFlagsVariable == 0 {
			tupleType := tupleArgType.TargetTupleType()
			elementInfos := tupleType.ElementInfos()
			for i := range elementInfos {
				name := "element"
				label := elementInfos[i].LabeledDeclaration()
				if label != nil && ast.IsIdentifier(label.Name()) && luau.IsValidIdentifier(label.Name().Text()) {
					name = label.Name().Text()
				}
				iteratorReturnIds = append(iteratorReturnIds, luau.TempId(name))
			}
		} else {
			iterFuncName := valueToIdStr(exp)
			if iterFuncName == "" {
				iterFuncName = "iterFunc"
			}
			iterFuncId := prereqs.PushToVar(exp, iterFuncName)
			loopStatements := luau.MakeList()

			initializerStatements := luau.MakeList()
			valueId := transformForInitializer(state, prereqs, initializer, initializerStatements)

			loopStatements.Push(luau.Create(&luau.VariableDeclaration{
				Left:  valueId,
				Right: luau.NewArray(luau.Call(iterFuncId)),
			}))

			loopStatements.Push(luau.Create(&luau.IfStatement{
				// Luau stops an iterator when its first return is nil, regardless of later returns
				Condition: luau.Binary(
					luau.Create(&luau.ComputedIndexExpression{
						Expression: valueId,
						Index:      luau.Number(1),
					}),
					"==",
					luau.Nil(),
				),
				Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
				ElseBody:   luau.MakeList(),
			}))

			loopStatements.PushList(initializerStatements)

			loopStatements.PushList(statements)

			return luau.MakeList(
				luau.Create(&luau.WhileStatement{
					Condition:  luau.Bool(true),
					Statements: loopStatements,
				}),
			)
		}

		bindingInitializers := luau.MakeList()
		tupleId := transformForInitializer(state, prereqs, initializer, bindingInitializers)
		statements.UnshiftList(bindingInitializers)

		builder := makeForLoopBuilder(func(state *TransformState, prereqs *Prereqs, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
			for _, id := range iteratorReturnIds {
				ids.Push(id)
			}

			initializers.Push(luau.Create(&luau.VariableDeclaration{
				Left:  tupleId,
				Right: luau.Create(&luau.Array{Members: luau.MakeListOf(iteratorReturnIds)}),
			}))
			return exp
		})

		return builder(state, prereqs, statements, initializer, exp)
	}
}

// tupleCombinedFlags is TupleType.combinedFlags (unexported in typescript-go): the union of the flags of
// all elements.
func tupleCombinedFlags(tupleType *checker.TupleType) checker.ElementFlags {
	var combinedFlags checker.ElementFlags
	for _, flags := range tupleType.ElementFlags() {
		combinedFlags |= flags
	}
	return combinedFlags
}

var buildGeneratorLoop loopBuilder = makeForLoopBuilder(func(state *TransformState, prereqs *Prereqs, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
	loopId := luau.TempId("result")
	ids.Push(loopId)

	initializers.Push(luau.Create(&luau.IfStatement{
		Condition:  luau.Property(loopId, "done"),
		Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
		ElseBody:   luau.MakeList(),
	}))

	if ast.IsVariableDeclarationList(initializer) {
		bindingList := luau.MakeList()
		initializers.Push(luau.Create(&luau.VariableDeclaration{
			Left:  transformForInitializer(state, prereqs, initializer, bindingList),
			Right: luau.Property(loopId, "value"),
		}))
		initializers.PushList(bindingList)
	} else {
		transformForInitializerExpressionDirect(
			state,
			prereqs,
			initializer,
			initializers,
			luau.Property(loopId, "value"),
		)
	}

	return luau.Property(convertToIndexableExpression(exp), "next")
})

func emptyLoopBuilder(*TransformState, *Prereqs, *luau.List, *ast.Node, luau.Expression) *luau.List {
	return luau.MakeList()
}

func getLoopBuilder(state *TransformState, node *ast.Node, type_ *checker.Type) loopBuilder {
	if isDefinitelyType(type_, isArrayType(state)) {
		return buildArrayLoop
	} else if isDefinitelyType(type_, isSetType(state)) {
		return buildSetLoop
	} else if isDefinitelyType(type_, isMapType(state)) || isDefinitelyType(type_, isSharedTableType(state)) {
		return buildMapLoop
	} else if isDefinitelyType(type_, isStringType) {
		return buildStringLoop
	} else if isDefinitelyType(type_, isIterableFunctionLuaTupleType(state)) {
		return buildIterableFunctionLuaTupleLoop(type_)
	} else if isDefinitelyType(type_, isIterableFunctionType(state)) {
		return buildIterableFunctionLoop
	} else if isDefinitelyType(type_, isGeneratorType(state)) {
		return buildGeneratorLoop
	} else if isDefinitelyType(type_, isIterableType(state)) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoIterableIteration.Create(node))
		return emptyLoopBuilder
	} else if type_.IsUnion() {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoMacroUnion.Create(node))
		return emptyLoopBuilder
	} else {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoUnsupportedIteration.Create(node))
		return emptyLoopBuilder
	}
}

func findRangeMacro(state *TransformState, node *ast.Node) *ast.Node {
	expression := skipDownwards(node.AsForInOrOfStatement().Expression)
	if ast.IsCallExpression(expression) {
		symbol := getFirstDefinedSymbol(state, state.GetType(expression.Expression()))
		if symbol != nil && symbol == state.Services.MacroManager.GetSymbolOrThrow(SYMBOL_NAMES.Range) {
			return expression
		}
	}
	return nil
}

// transformForOfRangeMacro is TSTransformer/nodes/statements/transformForOfStatement.ts transformForOfRangeMacro.
func transformForOfRangeMacro(state *TransformState, node *ast.Node, macroCall *ast.Node) *luau.List {
	forOf := node.AsForInOrOfStatement()
	arguments := macroCall.AsCallExpression().Arguments.Nodes
	result := luau.MakeList()

	statements := luau.MakeList()
	initializerPrereqs := NewPrereqs()
	id := transformForInitializer(state, initializerPrereqs, forOf.Initializer, statements)
	result.PushList(initializerPrereqs.Statements)

	rangePrereqs := NewPrereqs()
	rangeArgs := ensureTransformOrder(state, rangePrereqs, arguments, transformExpression)
	var start, end, step luau.Expression
	if len(rangeArgs) > 0 {
		start = rangeArgs[0]
	}
	if len(rangeArgs) > 1 {
		end = rangeArgs[1]
	}
	if len(rangeArgs) > 2 {
		step = rangeArgs[2]
	}
	result.PushList(rangePrereqs.Statements)

	statements.PushList(transformStatementList(state, forOf.Statement, getStatements(forOf.Statement), nil))

	var stepNode *ast.Node
	if len(arguments) > 2 {
		stepNode = skipDownwards(arguments[2])
	}
	stepIsNegation := stepNode != nil && ast.IsPrefixUnaryExpression(stepNode) &&
		stepNode.AsPrefixUnaryExpression().Operator == ast.KindMinusToken

	// a numeric for loop throws on a nil step, so dynamic steps fall back to 1
	// literals and numeric negations cannot produce nil
	if step != nil {
		if _, isLiteral := getLiteralNumberValue(step); !isLiteral && !stepIsNegation {
			step = luau.Binary(step, "or", luau.Number(1))
		}
	}

	result.Push(luau.Create(&luau.NumericForStatement{
		Id:         id,
		Start:      start,
		End:        end,
		Step:       step,
		Statements: statements,
	}))

	return result
}

// transformForOfStatement is TSTransformer/nodes/statements/transformForOfStatement.ts transformForOfStatement.
func transformForOfStatement(state *TransformState, node *ast.Node) *luau.List {
	forOf := node.AsForInOrOfStatement()
	if forOf.AwaitModifier != nil {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoAwaitForOf.Create(node))
	}

	if ast.IsVariableDeclarationList(forOf.Initializer) {
		name := forOf.Initializer.AsVariableDeclarationList().Declarations.Nodes[0].Name()
		if ast.IsIdentifier(name) {
			validateIdentifier(name)
		}
	}

	if rangeMacroCall := findRangeMacro(state, node); rangeMacroCall != nil {
		return transformForOfRangeMacro(state, node, rangeMacroCall)
	}

	expPrereqs := NewPrereqs()
	exp := transformExpression(state, expPrereqs, forOf.Expression)

	expType := state.GetType(forOf.Expression)
	statements := transformStatementList(state, forOf.Statement, getStatements(forOf.Statement), nil)

	loopBuilder := getLoopBuilder(state, forOf.Expression, expType)
	loopPrereqs := NewPrereqs()
	loopStatements := loopBuilder(state, loopPrereqs, statements, forOf.Initializer, exp)

	result := luau.MakeList()
	result.PushList(loopPrereqs.Statements)
	result.PushList(expPrereqs.Statements)
	result.PushList(loopStatements)

	return result
}
