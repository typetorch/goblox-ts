package luau

// Port of LuauRenderer/util/visit.ts.

// Visitor is called before and after visiting each node. Either function may be nil.
type Visitor struct {
	Before func(node Node)
	After  func(node Node)
}

func visitChildren(node Node, visitor *Visitor) {
	switch n := node.(type) {
	case *ComputedIndexExpression:
		visitNode(n.Expression, visitor)
		visitNode(n.Index, visitor)
	case *PropertyAccessExpression:
		visitNode(n.Expression, visitor)
	case *CallExpression:
		visitNode(n.Expression, visitor)
		visitList(n.Args, visitor)
	case *MethodCallExpression:
		visitNode(n.Expression, visitor)
		visitList(n.Args, visitor)
	case *ParenthesizedExpression:
		visitNode(n.Expression, visitor)
	case *FunctionExpression:
		visitList(n.Parameters, visitor)
		visitList(n.Statements, visitor)
	case *BinaryExpression:
		visitNode(n.Left, visitor)
		visitNode(n.Right, visitor)
	case *UnaryExpression:
		visitNode(n.Expression, visitor)
	case *IfExpression:
		visitNode(n.Condition, visitor)
		visitNode(n.Expression, visitor)
		visitNode(n.Alternative, visitor)
	case *InterpolatedString:
		visitList(n.Parts, visitor)
	case *Array:
		visitList(n.Members, visitor)
	case *Map:
		visitList(n.Fields, visitor)
	case *Set:
		visitList(n.Members, visitor)
	case *MixedTable:
		visitList(n.Fields, visitor)
	case *Assignment:
		visitListOrNode(n.Left, visitor)
		visitListOrNode(n.Right, visitor)
	case *CallStatement:
		visitNode(n.Expression, visitor)
	case *DoStatement:
		visitList(n.Statements, visitor)
	case *WhileStatement:
		visitNode(n.Condition, visitor)
		visitList(n.Statements, visitor)
	case *RepeatStatement:
		visitList(n.Statements, visitor)
		visitNode(n.Condition, visitor)
	case *IfStatement:
		visitNode(n.Condition, visitor)
		visitList(n.Statements, visitor)
		visitListOrNode(n.ElseBody, visitor)
	case *NumericForStatement:
		visitNode(n.Id, visitor)
		visitNode(n.Start, visitor)
		visitNode(n.End, visitor)
		if n.Step != nil {
			visitNode(n.Step, visitor)
		}
		visitList(n.Statements, visitor)
	case *ForStatement:
		visitList(n.Ids, visitor)
		visitList(n.Statements, visitor)
	case *FunctionDeclaration:
		visitNode(n.Name, visitor)
		visitList(n.Parameters, visitor)
		visitList(n.Statements, visitor)
	case *MethodDeclaration:
		visitNode(n.Expression, visitor)
		visitList(n.Parameters, visitor)
		visitList(n.Statements, visitor)
	case *VariableDeclaration:
		visitListOrNode(n.Left, visitor)
		if n.Right != nil {
			visitListOrNode(n.Right, visitor)
		}
	case *ReturnStatement:
		visitListOrNode(n.Expression, visitor)
	case *MapField:
		visitNode(n.Index, visitor)
		visitNode(n.Value, visitor)
	}
}

func visitNode(node Node, visitor *Visitor) {
	if visitor.Before != nil {
		visitor.Before(node)
	}
	visitChildren(node, visitor)
	if visitor.After != nil {
		visitor.After(node)
	}
}

func visitList(list *List, visitor *Visitor) {
	for listNode := list.Head; listNode != nil; listNode = listNode.Next {
		visitNode(listNode.Value, visitor)
	}
}

func visitListOrNode(value ListOrNode, visitor *Visitor) {
	if l, ok := value.(*List); ok {
		visitList(l, visitor)
	} else {
		visitNode(value.(Node), visitor)
	}
}

// Visit walks ast (a *List or a Node) depth first.
func Visit(ast ListOrNode, visitor *Visitor) {
	visitListOrNode(ast, visitor)
}
