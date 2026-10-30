package transform

// Port of TSTransformer/util/expressionMightMutate.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func isEnumMemberAccess(state *TransformState, node *ast.Node) bool {
	if !ast.IsPropertyAccessExpression(node) && !ast.IsElementAccessExpression(node) {
		return false
	}

	var memberNode *ast.Node
	if ast.IsPropertyAccessExpression(node) {
		memberNode = node.Name()
	} else {
		memberNode = node.AsElementAccessExpression().ArgumentExpression
	}
	memberSymbol := state.TypeChecker.GetSymbolAtLocation(memberNode)
	enumSymbol := state.TypeChecker.GetSymbolAtLocation(skipDownwards(node.Expression()))
	return memberSymbol != nil &&
		(memberSymbol.Flags&ast.SymbolFlagsEnumMember) != 0 &&
		enumSymbol != nil &&
		(skipAliases(state.TypeChecker, enumSymbol).Flags&ast.SymbolFlagsEnum) != 0
}

// expressionMightMutate is TSTransformer/util/expressionMightMutate.ts expressionMightMutate.
func expressionMightMutate(state *TransformState, expression luau.Expression, node *ast.Node) bool {
	mightMutate := func(value luau.Node) bool { return expressionMightMutate(state, value, nil) }
	switch e := expression.(type) {
	case *luau.TemporaryIdentifier:
		// Assume tempIds are never re-assigned after being returned
		// TODO: Is this actually safe to assume?
		return false
	case *luau.ParenthesizedExpression:
		return mightMutate(e.Expression)
	}
	if luau.IsSimplePrimitive(expression) {
		return false
	} else if luau.IsFunctionExpression(expression) {
		return false
	} else if luau.IsVarArgsLiteral(expression) {
		return false
	}
	switch e := expression.(type) {
	case *luau.IfExpression:
		return mightMutate(e.Condition) || mightMutate(e.Expression) || mightMutate(e.Alternative)
	case *luau.BinaryExpression:
		return mightMutate(e.Left) || mightMutate(e.Right)
	case *luau.UnaryExpression:
		return mightMutate(e.Expression)
	case *luau.Array:
		return e.Members.Some(mightMutate)
	case *luau.Set:
		return e.Members.Some(mightMutate)
	case *luau.Map:
		return e.Fields.Some(func(value luau.Node) bool {
			field := value.(*luau.MapField)
			return mightMutate(field.Index) || mightMutate(field.Value)
		})
	}
	if node != nil {
		node = skipDownwards(node)
		if isEnumMemberAccess(state, node) {
			return false
		} else if ast.IsIdentifier(node) {
			symbol := state.TypeChecker.GetSymbolAtLocation(node)
			if symbol != nil && !isSymbolMutable(state, symbol) {
				return false
			}
		}
	}
	// Identifier
	// ComputedIndexExpression
	// PropertyAccessExpression
	// CallExpression
	// MethodCallExpression
	return true
}
