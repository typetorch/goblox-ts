package transform

// Port of TSTransformer/nodes/statements/transformForStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
	"github.com/microsoft/typescript-go/internal/scanner"
)

func addFinalizersToIfStatement(node *luau.IfStatement, finalizers *luau.List) {
	if node.Statements.IsNonEmpty() {
		addFinalizers(node.Statements, node.Statements.Head, finalizers)
	}
	if elseBody, ok := node.ElseBody.(*luau.List); ok {
		if elseBody.IsNonEmpty() {
			addFinalizers(elseBody, elseBody.Head, finalizers)
		}
	} else {
		addFinalizersToIfStatement(node.ElseBody.(*luau.IfStatement), finalizers)
	}
}

func addFinalizers(list *luau.List, node *luau.ListNode, finalizers *luau.List) {
	assert(!list.IsEmpty())

	statement := node.Value
	if luau.IsContinueStatement(statement) {
		finalizersClone := finalizers.Clone()

		// fix node parents
		finalizersClone.ForEach(func(node luau.Node) { luau.SetParent(node, statement.Parent()) })

		if node.Prev != nil {
			node.Prev.Next = finalizersClone.Head
		} else {
			assert(node == list.Head)
			list.Head = finalizersClone.Head
		}

		node.Prev = finalizersClone.Tail

		finalizersClone.Tail.Next = node
	}

	if doStatement, ok := statement.(*luau.DoStatement); ok {
		if doStatement.Statements.IsNonEmpty() {
			addFinalizers(doStatement.Statements, doStatement.Statements.Head, finalizers)
		}
	} else if ifStatement, ok := statement.(*luau.IfStatement); ok {
		addFinalizersToIfStatement(ifStatement, finalizers)
	}

	if node.Next != nil {
		addFinalizers(list, node.Next, finalizers)
	}
}

func canSkipClone(state *TransformState, initializer *ast.Node, id *ast.Node) bool {
	// is symbol used in initializer (besides its definition)
	return !state.TypeChecker.IsSymbolReferencedInFileWithin(id, ast.GetSourceFileOfNode(id), initializer)
}

func isIdWriteOrAsyncRead(state *TransformState, forStatement *ast.Node, id *ast.Node) bool {
	incrementor := forStatement.AsForStatement().Incrementor
	return state.TypeChecker.EachSymbolReferenceInFile(
		id,
		ast.GetSourceFileOfNode(id),
		func(token *ast.Node) bool {
			// write
			if ast.IsWriteAccess(token) && (incrementor == nil || !isAncestorOf(incrementor, token)) {
				return true
			}

			// async read
			ancestor := getAncestor(token, func(v *ast.Node) bool { return v == forStatement || ast.IsFunctionLike(v) })
			if ancestor != nil && ancestor != forStatement {
				return true
			}
			return false
		},
		forStatement,
	)
}

func transformForStatementFallback(state *TransformState, node *ast.Node) *luau.List {
	forStatement := node.AsForStatement()
	initializer, condition, incrementor, statement := forStatement.Initializer, forStatement.Condition, forStatement.Incrementor, forStatement.Statement

	result := luau.MakeList()
	whileStatements := luau.MakeList()
	finalizerStatements := luau.MakeList()

	var variables []*ast.Node
	if initializer != nil && ast.IsVariableDeclarationList(initializer) {
		variables = getDeclaredVariables(initializer)
	}
	hasWriteOrAsyncRead := map[*ast.Symbol]bool{}
	skipClone := map[*ast.Symbol]bool{}

	if initializer != nil && ast.IsVariableDeclarationList(initializer) {
		for _, id := range variables {
			symbol := state.TypeChecker.GetSymbolAtLocation(id)
			assert(symbol != nil)
			if isIdWriteOrAsyncRead(state, node, id) {
				hasWriteOrAsyncRead[symbol] = true
			}
			if canSkipClone(state, initializer, id) {
				skipClone[symbol] = true
			}
		}
	}

	if initializer != nil {
		if ast.IsVariableDeclarationList(initializer) {
			if isVarDeclaration(initializer) {
				DiagnosticService.AddDiagnostic(diagnostics.Errors.NoVar.Create(node))
			}

			for _, id := range variables {
				symbol := state.TypeChecker.GetSymbolAtLocation(id)
				assert(symbol != nil)
				if hasWriteOrAsyncRead[symbol] {
					if skipClone[symbol] {
						state.SymbolToIdMap[symbol] = luau.TempId(scanner.GetTextOfNode(id))
					} else {
						copyId := luau.TempId(scanner.GetTextOfNode(id) + "Copy")
						state.SymbolToIdMap[symbol] = copyId
					}
				}
			}

			for _, declaration := range initializer.AsVariableDeclarationList().Declarations.Nodes {
				result.PushList(transformVariableDeclaration(state, declaration))
			}

			for _, id := range variables {
				symbol := state.TypeChecker.GetSymbolAtLocation(id)
				assert(symbol != nil)
				if hasWriteOrAsyncRead[symbol] {
					var tempId *luau.TemporaryIdentifier
					if skipClone[symbol] {
						tempId = state.SymbolToIdMap[symbol]
						assert(tempId != nil)
					} else {
						tempId = luau.TempId(scanner.GetTextOfNode(id))
						copyId := state.SymbolToIdMap[symbol]
						assert(copyId != nil)

						// local _i = _iCopy
						result.Push(luau.Create(&luau.VariableDeclaration{
							Left:  tempId,
							Right: copyId,
						}))
					}
					delete(state.SymbolToIdMap, symbol)
					realId := transformIdentifierDefined(state, id)

					// local i = _i
					whileStatements.Push(luau.Create(&luau.VariableDeclaration{
						Left:  realId,
						Right: tempId,
					}))

					// _i = i
					finalizerStatements.Push(luau.Create(&luau.Assignment{
						Left:     tempId,
						Operator: "=",
						Right:    realId,
					}))
				}
			}
		} else {
			result.PushList(transformExpressionStatementInner(state, initializer))
		}
	}

	if incrementor != nil {
		shouldIncrement := luau.TempId("shouldIncrement")

		// local _shouldIncrement = false
		result.Push(luau.Create(&luau.VariableDeclaration{
			Left:  shouldIncrement,
			Right: luau.Bool(false),
		}))

		incrementorStatements := transformExpressionStatementInner(state, incrementor)

		// if _shouldIncrement then
		// 	[incrementorStatements]
		// else
		// 	_shouldIncrement = true
		// end
		whileStatements.Push(luau.Create(&luau.IfStatement{
			Condition:  shouldIncrement,
			Statements: incrementorStatements,
			ElseBody: luau.MakeList(
				luau.Create(&luau.Assignment{
					Left:     shouldIncrement,
					Operator: "=",
					Right:    luau.Bool(true),
				}),
			),
		}))
	}

	conditionPrereqs := NewPrereqs()
	var conditionExp luau.Expression
	if condition != nil {
		conditionExp = createTruthinessChecks(
			state,
			conditionPrereqs,
			transformExpression(state, conditionPrereqs, condition),
			condition,
		)
	} else {
		conditionExp = luau.Bool(true)
	}

	whileStatements.PushList(conditionPrereqs.Statements)

	if !whileStatements.IsEmpty() {
		if condition != nil {
			// if not [conditionExp] then
			//	break
			// end
			whileStatements.Push(luau.Create(&luau.IfStatement{
				Condition:  luau.Unary("not", conditionExp),
				Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
				ElseBody:   luau.MakeList(),
			}))
		}
		conditionExp = luau.Bool(true)
	}

	whileStatements.PushList(transformStatementList(state, statement, getStatements(statement), nil))

	if whileStatements.IsNonEmpty() && finalizerStatements.IsNonEmpty() {
		addFinalizers(whileStatements, whileStatements.Head, finalizerStatements)
	}

	if whileStatements.Tail == nil || !luau.IsFinalStatement(whileStatements.Tail.Value) {
		whileStatements.PushList(finalizerStatements)
	}

	result.Push(luau.Create(&luau.WhileStatement{
		Condition:  conditionExp,
		Statements: whileStatements,
	}))

	if result.Head == result.Tail {
		return result
	}
	return luau.MakeList(luau.Create(&luau.DoStatement{Statements: result}))
}

func isLoopVariable(state *TransformState, expression *ast.Node, symbol *ast.Symbol) bool {
	node := skipDownwards(expression)
	return ast.IsIdentifier(node) && state.TypeChecker.GetSymbolAtLocation(node) == symbol
}

// optimizedStep is OptimizedStep.
type optimizedStep struct {
	value      float64
	expression *ast.Node
	negate     bool
}

func getOptimizedIncrementorStep(state *TransformState, incrementor *ast.Node, idSymbol *ast.Symbol) *optimizedStep {
	incrementor = skipDownwards(incrementor)
	if ast.IsBinaryExpression(incrementor) && isLoopVariable(state, incrementor.AsBinaryExpression().Left, idSymbol) {
		update := incrementor
		incrementorOperator := incrementor.AsBinaryExpression().OperatorToken.Kind
		if incrementorOperator == ast.KindEqualsToken {
			right := skipDownwards(incrementor.AsBinaryExpression().Right)
			if !ast.IsBinaryExpression(right) {
				return nil
			}
			update = right
		} else if incrementorOperator != ast.KindPlusEqualsToken && incrementorOperator != ast.KindMinusEqualsToken {
			return nil
		}

		updateBinary := update.AsBinaryExpression()
		operator := updateBinary.OperatorToken.Kind
		var negate bool
		if operator == ast.KindPlusToken || operator == ast.KindPlusEqualsToken {
			negate = false
		} else if operator == ast.KindMinusToken || operator == ast.KindMinusEqualsToken {
			negate = true
		} else {
			return nil
		}

		var expression *ast.Node
		if isLoopVariable(state, updateBinary.Left, idSymbol) {
			expression = updateBinary.Right
		} else if operator == ast.KindPlusToken && isLoopVariable(state, updateBinary.Right, idSymbol) {
			expression = updateBinary.Left
		} else {
			return nil
		}

		if value, ok := getConstantInteger(state, expression, true); ok {
			if negate {
				value = -value
			}
			return &optimizedStep{value: value, expression: expression, negate: negate}
		}
	} else if ast.IsPostfixUnaryExpression(incrementor) || ast.IsPrefixUnaryExpression(incrementor) {
		var operand *ast.Node
		var operator ast.Kind
		if ast.IsPostfixUnaryExpression(incrementor) {
			operand = incrementor.AsPostfixUnaryExpression().Operand
			operator = incrementor.AsPostfixUnaryExpression().Operator
		} else {
			operand = incrementor.AsPrefixUnaryExpression().Operand
			operator = incrementor.AsPrefixUnaryExpression().Operator
		}
		if isLoopVariable(state, operand, idSymbol) {
			if operator == ast.KindPlusPlusToken {
				return &optimizedStep{value: 1, negate: false}
			} else if operator == ast.KindMinusMinusToken {
				return &optimizedStep{value: -1, negate: false}
			}
		}
	}
	return nil
}

func getOptimizedCondition(state *TransformState, condition *ast.Node, idSymbol *ast.Symbol) (bound *ast.Node, operator ast.Kind, ok bool) {
	condition = skipDownwards(condition)
	if !ast.IsBinaryExpression(condition) {
		return nil, 0, false
	}
	binary := condition.AsBinaryExpression()

	bound = binary.Right
	operator = binary.OperatorToken.Kind
	if !isLoopVariable(state, binary.Left, idSymbol) {
		if !isLoopVariable(state, binary.Right, idSymbol) {
			return nil, 0, false
		}

		bound = binary.Left
		switch operator {
		case ast.KindLessThanToken:
			operator = ast.KindGreaterThanToken
		case ast.KindLessThanEqualsToken:
			operator = ast.KindGreaterThanEqualsToken
		case ast.KindGreaterThanToken:
			operator = ast.KindLessThanToken
		case ast.KindGreaterThanEqualsToken:
			operator = ast.KindLessThanEqualsToken
		}
	}

	return bound, operator, true
}

func isMutatedInBody(state *TransformState, identifier *ast.Node, body *ast.Node) bool {
	return state.TypeChecker.EachSymbolReferenceInFile(
		identifier,
		ast.GetSourceFileOfNode(identifier),
		ast.IsWriteAccess,
		body,
	)
}

func transformForStatementOptimized(state *TransformState, node *ast.Node) *luau.List {
	forStatement := node.AsForStatement()
	initializer, condition, incrementor, statement := forStatement.Initializer, forStatement.Condition, forStatement.Incrementor, forStatement.Statement

	if initializer == nil ||
		!ast.IsVariableDeclarationList(initializer) ||
		initializer.Flags&ast.NodeFlagsLet == 0 ||
		len(initializer.AsVariableDeclarationList().Declarations.Nodes) != 1 {
		return nil
	}

	declaration := initializer.AsVariableDeclarationList().Declarations.Nodes[0].AsVariableDeclaration()
	decName, decInit := declaration.Name(), declaration.Initializer
	if !ast.IsIdentifier(decName) || decInit == nil {
		return nil
	}

	idSymbol := state.TypeChecker.GetSymbolAtLocation(decName)
	assert(idSymbol != nil)

	if _, ok := getConstantInteger(state, decInit, false); !ok {
		return nil
	}

	// require a nonzero constant step that updates the declared loop variable

	if incrementor == nil {
		return nil
	}

	increment := getOptimizedIncrementorStep(state, incrementor, idSymbol)
	if increment == nil || increment.value == 0 {
		return nil
	}

	if condition == nil {
		return nil
	}
	bound, operator, ok := getOptimizedCondition(state, condition, idSymbol)
	if !ok {
		return nil
	}

	if operator == ast.KindLessThanToken || operator == ast.KindLessThanEqualsToken {
		if increment.value < 0 {
			return nil
		}
	} else if operator == ast.KindGreaterThanToken || operator == ast.KindGreaterThanEqualsToken {
		if increment.value > 0 {
			return nil
		}
	} else {
		return nil
	}

	if _, ok := getConstantInteger(state, bound, false); !ok {
		return nil
	}

	if isMutatedInBody(state, decName, statement) {
		return nil
	}

	// commit to the optimization and start transforming..

	result := luau.MakeList()

	id := transformIdentifierDefined(state, decName)

	startPrereqs := NewPrereqs()
	start := transformExpression(state, startPrereqs, decInit)
	assert(startPrereqs.Statements.IsEmpty())
	endPrereqs := NewPrereqs()
	end := transformExpression(state, endPrereqs, bound)
	assert(endPrereqs.Statements.IsEmpty())

	var step luau.Expression = luau.Number(increment.value)
	if increment.expression != nil {
		stepPrereqs := NewPrereqs()
		expression := transformExpression(state, stepPrereqs, increment.expression)
		assert(stepPrereqs.Statements.IsEmpty())

		// retain references and arithmetic, but keep the existing compact spelling of literal steps
		if _, isLiteral := getLiteralNumberValue(expression); !isLiteral {
			if increment.negate {
				step = luau.Unary("-", expression)
			} else {
				step = expression
			}
		}
	}

	statements := transformStatementList(state, statement, getStatements(statement), nil)

	if operator == ast.KindLessThanToken {
		end = offset(end, -1)
	} else if operator == ast.KindGreaterThanToken {
		end = offset(end, 1)
	}

	result.Push(luau.Create(&luau.NumericForStatement{Id: id, Start: start, End: end, Step: step, Statements: statements}))

	return result
}

// transformForStatement is TSTransformer/nodes/statements/transformForStatement.ts transformForStatement.
func transformForStatement(state *TransformState, node *ast.Node) *luau.List {
	if state.Data.ProjectOptions.OptimizedLoops {
		if optimized := transformForStatementOptimized(state, node); optimized != nil {
			return optimized
		}
	}
	return transformForStatementFallback(state, node)
}
