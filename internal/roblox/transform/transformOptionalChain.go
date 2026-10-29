package transform

// Port of TSTransformer/nodes/transformOptionalChain.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// OptionalChainItemKind is the OptionalChainItemKind enum.
const (
	OptionalChainItemKindPropertyAccess = iota
	OptionalChainItemKindElementAccess
	OptionalChainItemKindCall
	OptionalChainItemKindPropertyCall
	OptionalChainItemKindElementCall
)

// ChainItem is the union of the TypeScript chain item interfaces (PropertyAccessItem, ElementAccessItem,
// CallItem, PropertyCallItem and ElementCallItem), told apart by Kind. Expression is the
// argumentExpression of an ElementAccessItem and the callee (a property or element access) of a
// PropertyCallItem or ElementCallItem.
type ChainItem struct {
	Kind               int
	Node               *ast.Node
	Optional           bool
	Name               string
	Expression         *ast.Node
	ArgumentExpression *ast.Node
	CallOptional       bool
	Args               []*ast.Node
}

func createPropertyAccessItem(node *ast.Node) ChainItem {
	return ChainItem{
		Node:     node,
		Kind:     OptionalChainItemKindPropertyAccess,
		Optional: node.AsPropertyAccessExpression().QuestionDotToken != nil,
		Name:     node.Name().Text(),
	}
}

func createElementAccessItem(node *ast.Node) ChainItem {
	return ChainItem{
		Node:       node,
		Kind:       OptionalChainItemKindElementAccess,
		Optional:   node.AsElementAccessExpression().QuestionDotToken != nil,
		Expression: node.AsElementAccessExpression().ArgumentExpression,
	}
}

func createCallItem(node *ast.Node) ChainItem {
	return ChainItem{
		Node:     node,
		Kind:     OptionalChainItemKindCall,
		Optional: node.AsCallExpression().QuestionDotToken != nil,
		Args:     node.AsCallExpression().Arguments.Nodes,
	}
}

func createPropertyCallItem(node *ast.Node, expression *ast.Node) ChainItem {
	return ChainItem{
		Node:         node,
		Expression:   expression,
		Kind:         OptionalChainItemKindPropertyCall,
		Optional:     expression.AsPropertyAccessExpression().QuestionDotToken != nil,
		Name:         expression.Name().Text(),
		CallOptional: node.AsCallExpression().QuestionDotToken != nil,
		Args:         node.AsCallExpression().Arguments.Nodes,
	}
}

func createElementCallItem(node *ast.Node, expression *ast.Node) ChainItem {
	return ChainItem{
		Node:               node,
		Expression:         expression,
		Kind:               OptionalChainItemKindElementCall,
		Optional:           expression.AsElementAccessExpression().QuestionDotToken != nil,
		ArgumentExpression: expression.AsElementAccessExpression().ArgumentExpression,
		CallOptional:       node.AsCallExpression().QuestionDotToken != nil,
		Args:               node.AsCallExpression().Arguments.Nodes,
	}
}

// flattenOptionalChain is TSTransformer/nodes/transformOptionalChain.ts flattenOptionalChain.
func flattenOptionalChain(expression *ast.Node) ([]ChainItem, *ast.Node) {
	var chain []ChainItem
	unshift := func(item ChainItem) { chain = append([]ChainItem{item}, chain...) }
	for {
		if ast.IsPropertyAccessExpression(expression) {
			unshift(createPropertyAccessItem(expression))
			expression = expression.Expression()
		} else if ast.IsElementAccessExpression(expression) {
			unshift(createElementAccessItem(expression))
			expression = expression.Expression()
		} else if ast.IsCallExpression(expression) {
			// this is a bit of a mess..
			subExp := skipDownwards(expression.Expression())
			if ast.IsPropertyAccessExpression(subExp) {
				unshift(createPropertyCallItem(expression, subExp))
				expression = subExp.Expression()
			} else if ast.IsElementAccessExpression(subExp) {
				unshift(createElementCallItem(expression, subExp))
				expression = subExp.Expression()
			} else {
				unshift(createCallItem(expression))
				expression = subExp
			}
		} else {
			break
		}
	}
	return chain, expression
}

func transformChainItem(state *TransformState, prereqs *Prereqs, baseExpression luau.Expression, item ChainItem) luau.Expression {
	if item.Kind == OptionalChainItemKindPropertyAccess {
		return transformPropertyAccessExpressionInner(state, prereqs, item.Node, baseExpression, item.Name)
	} else if item.Kind == OptionalChainItemKindElementAccess {
		return transformElementAccessExpressionInner(state, prereqs, item.Node, baseExpression, item.Expression)
	} else if item.Kind == OptionalChainItemKindCall {
		return transformCallExpressionInner(state, prereqs, item.Node, baseExpression, item.Args)
	} else if item.Kind == OptionalChainItemKindPropertyCall {
		return transformPropertyCallExpressionInner(
			state,
			prereqs,
			item.Node,
			item.Expression,
			baseExpression,
			item.Name,
			item.Args,
		)
	} else {
		return transformElementCallExpressionInner(
			state,
			prereqs,
			item.Node,
			item.Expression,
			baseExpression,
			item.ArgumentExpression,
			item.Args,
		)
	}
}

func createOrSetTempId(
	prereqs *Prereqs,
	tempId *luau.TemporaryIdentifier,
	expression luau.Expression,
	node *ast.Node,
) *luau.TemporaryIdentifier {
	if tempId == nil {
		name := "result"
		if node.Parent != nil && ast.IsVariableDeclaration(node.Parent) && ast.IsIdentifier(node.Parent.Name()) {
			name = node.Parent.Name().Text()
		}
		tempId = prereqs.PushToVar(expression, name)
	} else {
		if luau.Node(tempId) != expression {
			prereqs.Push(luau.Create(&luau.Assignment{
				Left:     tempId,
				Operator: "=",
				Right:    expression,
			}))
		}
	}
	return tempId
}

func createNilCheck(tempId *luau.TemporaryIdentifier, statements *luau.List) *luau.IfStatement {
	return luau.Create(&luau.IfStatement{
		Condition:  luau.Binary(tempId, "~=", luau.Nil()),
		Statements: statements,
		ElseBody:   luau.MakeList(),
	})
}

func isCompoundCall(item ChainItem) bool {
	return item.Kind == OptionalChainItemKindPropertyCall || item.Kind == OptionalChainItemKindElementCall
}

func transformOptionalChainInner(
	state *TransformState,
	prereqs *Prereqs,
	chain []ChainItem,
	baseExpression luau.Expression,
	tempId *luau.TemporaryIdentifier,
	index int,
) luau.Expression {
	if index >= len(chain) {
		return baseExpression
	}
	item := chain[index]
	if item.Optional || (isCompoundCall(item) && item.CallOptional) {
		isMethodCall := false
		isSuperCall := false
		var selfParam *luau.TemporaryIdentifier

		if isCompoundCall(item) {
			isMethodCall = isMethod(state, item.Expression)
			isSuperCall = ast.IsSuperProperty(item.Expression)

			if item.CallOptional && isMethodCall && !isSuperCall {
				selfParam = prereqs.PushToVar(baseExpression, "self")
				baseExpression = selfParam
			}

			if item.Optional {
				tempId = createOrSetTempId(prereqs, tempId, baseExpression, chain[len(chain)-1].Node)
				baseExpression = tempId
			}

			if item.CallOptional {
				if item.Kind == OptionalChainItemKindPropertyCall {
					baseExpression = luau.Property(convertToIndexableExpression(baseExpression), item.Name)
				} else {
					expType := state.GetType(item.Expression.Expression())

					baseExpression = luau.Create(&luau.ComputedIndexExpression{
						Expression: convertToIndexableExpression(baseExpression),
						Index: addOneIfArrayType(
							state,
							expType,
							transformExpression(state, prereqs, item.ArgumentExpression),
						),
					})
				}
			}
		}

		// receiver initialization may need its own nil check before the optional call
		receiverPrereqs := NewPrereqs()
		tempId = createOrSetTempId(receiverPrereqs, tempId, baseExpression, chain[len(chain)-1].Node)

		branchPrereqs := NewPrereqs()
		var newExpression luau.Expression
		if isCompoundCall(item) && item.CallOptional {
			expType := state.TypeChecker.GetNonOptionalType(state.GetType(item.Node.Expression()))
			symbol := getFirstDefinedSymbol(state, expType)
			var macro PropertyCallMacro
			if symbol != nil {
				macro = state.Services.MacroManager.GetPropertyCallMacro(symbol)
			}
			if macro != nil {
				DiagnosticService.AddDiagnostic(diagnostics.Errors.NoOptionalMacroCall.Create(item.Node))
			} else {
				args := ensureTransformOrder(state, branchPrereqs, item.Args, transformExpression)
				if isMethodCall {
					if isSuperCall {
						args = append([]luau.Expression{luau.Globals.Self}, args...)
					} else {
						args = append([]luau.Expression{selfParam}, args...)
					}
				}
				newExpression = wrapReturnIfLuaTuple(state, item.Node, luau.Call(tempId, args...))
			}
		} else {
			newExpression = transformChainItem(state, branchPrereqs, tempId, item)
		}
		var newValue luau.Expression
		if newExpression != nil {
			newValue = transformOptionalChainInner(state, branchPrereqs, chain, newExpression, tempId, index+1)
		} else {
			newValue = luau.NewNone()
		}
		isUsed := !luau.IsNone(newValue) && !isUsedAsStatement(item.Node)

		if luau.Node(tempId) != newValue && isUsed {
			branchPrereqs.Push(luau.Create(&luau.Assignment{
				Left:     tempId,
				Operator: "=",
				Right:    newValue,
			}))
		} else if luau.IsCall(newValue) {
			branchPrereqs.Push(luau.Create(&luau.CallStatement{
				Expression: newValue,
			}))
		}

		receiverPrereqs.Push(createNilCheck(tempId, branchPrereqs.Statements))

		if isCompoundCall(item) && item.Optional && item.CallOptional {
			prereqs.Push(createNilCheck(tempId, receiverPrereqs.Statements))
		} else {
			prereqs.PushList(receiverPrereqs.Statements)
		}

		if isUsed {
			return tempId
		}
		return luau.NewNone()
	} else {
		return transformOptionalChainInner(
			state,
			prereqs,
			chain,
			transformChainItem(state, prereqs, baseExpression, item),
			tempId,
			index+1,
		)
	}
}

// transformOptionalChain is TSTransformer/nodes/transformOptionalChain.ts transformOptionalChain.
func transformOptionalChain(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	chain, expression := flattenOptionalChain(node)
	return transformOptionalChainInner(state, prereqs, chain, transformExpression(state, prereqs, expression), nil, 0)
}
