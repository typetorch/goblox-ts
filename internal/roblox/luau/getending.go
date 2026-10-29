package luau

// Port of LuauRenderer/util/getEnding.ts.

func endsWithIndexableExpressionInner(node Expression) bool {
	if IsIndexableExpression(node) {
		// `a` or `(a)` or `a.b` or `a[b]` or `a()`
		return true
	}
	switch n := node.(type) {
	case *BinaryExpression:
		// `a + b`
		return endsWithIndexableExpressionInner(n.Right)
	case *UnaryExpression:
		// `-a`
		return endsWithIndexableExpressionInner(n.Expression)
	case *IfExpression:
		// `if a then b else c`
		// `if a then b elseif c then d else e`
		return endsWithIndexableExpressionInner(n.Alternative)
	}
	return false
}

func furthestRightOf(left ListOrNode, right ListOrNode) Expression {
	if right != nil {
		if l, ok := right.(*List); ok {
			Assert(l.IsNonEmpty())
			return l.Tail.Value
		}
		return right.(Node)
	} else if l, ok := left.(*List); ok {
		Assert(l.IsNonEmpty())
		return l.Tail.Value
	}
	return left.(Node)
}

func endsWithIndexableExpression(node Statement) bool {
	switch n := node.(type) {
	case *CallStatement:
		// `a()`
		return true
	case *VariableDeclaration:
		// `local a = b` or `local a` or `local a, b`
		return endsWithIndexableExpressionInner(furthestRightOf(n.Left, n.Right))
	case *Assignment:
		// `a = b`
		return endsWithIndexableExpressionInner(furthestRightOf(n.Left, n.Right))
	}
	return false
}

func startsWithParenthesisInner(node Expression) bool {
	switch n := node.(type) {
	case *ParenthesizedExpression:
		// `(a)`
		return true
	case *CallExpression:
		// `(a)()`
		return startsWithParenthesisInner(n.Expression)
	case *MethodCallExpression:
		// `(a):b()`
		return startsWithParenthesisInner(n.Expression)
	case *PropertyAccessExpression:
		// `(a).b`
		return startsWithParenthesisInner(n.Expression)
	case *ComputedIndexExpression:
		// `(a)[b]`
		return startsWithParenthesisInner(n.Expression)
	}
	return false
}

func callInnerExpression(node Node) Expression {
	switch n := node.(type) {
	case *CallExpression:
		return n.Expression
	case *MethodCallExpression:
		return n.Expression
	}
	panic("CallStatement.Expression must be a CallExpression or a MethodCallExpression")
}

func startsWithParenthesis(node Statement) bool {
	switch n := node.(type) {
	case *CallStatement:
		// `(a)()`
		return startsWithParenthesisInner(callInnerExpression(n.Expression))
	case *Assignment:
		if l, ok := n.Left.(*List); ok {
			// `(a).b, c = d`
			Assert(l.IsNonEmpty())
			return startsWithParenthesisInner(l.Head.Value)
		}
		// `(a).b = c`
		return startsWithParenthesisInner(n.Left.(Node))
	}
	return false
}

func getNextNonComment(state *RenderState) Node {
	var listNode *ListNode
	if top := state.PeekListNode(); top != nil {
		listNode = top.Next
	}
	for listNode != nil && IsComment(listNode.Value) {
		listNode = listNode.Next
	}
	if listNode == nil {
		return nil
	}
	return listNode.Value
}

// getEnding resolves if the given statement needs to end with a `;` or not, to avoid "ambiguous
// syntax" errors in Luau. This is only necessary in statements which can end in an
// IndexableExpression: CallStatement, VariableDeclaration and Assignment.
func getEnding(state *RenderState, node Statement) string {
	nextStatement := getNextNonComment(state)
	if nextStatement != nil && endsWithIndexableExpression(node) && startsWithParenthesis(nextStatement) {
		return ";"
	}
	return ""
}
