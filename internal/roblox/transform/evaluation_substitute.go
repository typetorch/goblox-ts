package transform

// Port of TSTransformer/util/evaluation/substitute.ts.

import (
	"github.com/microsoft/typescript-go/internal/jsnum"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// substitute is TSTransformer/util/evaluation/substitute.ts substitute.
func substitute(node luau.Node, replacements map[int]luau.Expression) luau.Node {
	if temporary, ok := node.(*luau.TemporaryIdentifier); ok {
		if replacement := replacements[temporary.Id]; replacement != nil {
			return luau.ShallowClone(replacement)
		}
	}
	// `{ ...node }` without its parent, with every child node and list substituted
	updated := luau.ShallowClone(node)
	substituteChildren(updated, replacements)
	updated = luau.Create(updated)
	// substitution can turn an indexable reference into a call/literal/conditional
	switch n := updated.(type) {
	case *luau.PropertyAccessExpression:
		n.Expression = convertToIndexableExpression(n.Expression)
	case *luau.ComputedIndexExpression:
		n.Expression = convertToIndexableExpression(n.Expression)
	case *luau.CallExpression:
		n.Expression = convertToIndexableExpression(n.Expression)
	case *luau.MethodCallExpression:
		n.Expression = convertToIndexableExpression(n.Expression)
	}
	// recover arithmetic folding which was intentionally hidden behind references
	if binary, ok := updated.(*luau.BinaryExpression); ok && (binary.Operator == "+" || binary.Operator == "-") {
		if literal, ok := binary.Right.(*luau.NumberLiteral); ok {
			sign := 1.0
			if binary.Operator == "-" {
				sign = -1
			}
			return offset(binary.Left, float64(jsnum.FromString(literal.Value))*sign)
		}
	}
	return updated
}

// substituteChildren replaces every child field of a fresh clone: the `Object.entries(fields)` loop of
// substitute.
func substituteChildren(node luau.Node, replacements map[int]luau.Expression) {
	sub := func(child luau.Node) luau.Node {
		if child == nil {
			return nil
		}
		return substitute(child, replacements)
	}
	subList := func(list *luau.List) *luau.List {
		if list == nil {
			return nil
		}
		return luau.MakeList(luau.ListMapToArray(list, func(child luau.Node) luau.Node {
			return substitute(child, replacements)
		})...)
	}
	subListOrNode := func(value luau.ListOrNode) luau.ListOrNode {
		switch v := value.(type) {
		case nil:
			return nil
		case *luau.List:
			return subList(v)
		case luau.Node:
			return sub(v)
		}
		return value
	}
	switch n := node.(type) {
	case *luau.ComputedIndexExpression:
		n.Expression = sub(n.Expression)
		n.Index = sub(n.Index)
	case *luau.PropertyAccessExpression:
		n.Expression = sub(n.Expression)
	case *luau.CallExpression:
		n.Expression = sub(n.Expression)
		n.Args = subList(n.Args)
	case *luau.MethodCallExpression:
		n.Expression = sub(n.Expression)
		n.Args = subList(n.Args)
	case *luau.ParenthesizedExpression:
		n.Expression = sub(n.Expression)
	case *luau.FunctionExpression:
		n.Parameters = subList(n.Parameters)
		n.Statements = subList(n.Statements)
	case *luau.BinaryExpression:
		n.Left = sub(n.Left)
		n.Right = sub(n.Right)
	case *luau.UnaryExpression:
		n.Expression = sub(n.Expression)
	case *luau.IfExpression:
		n.Condition = sub(n.Condition)
		n.Expression = sub(n.Expression)
		n.Alternative = sub(n.Alternative)
	case *luau.InterpolatedString:
		n.Parts = subList(n.Parts)
	case *luau.Array:
		n.Members = subList(n.Members)
	case *luau.Map:
		n.Fields = subList(n.Fields)
	case *luau.Set:
		n.Members = subList(n.Members)
	case *luau.MixedTable:
		n.Fields = subList(n.Fields)
	case *luau.Assignment:
		n.Left = subListOrNode(n.Left)
		n.Right = subListOrNode(n.Right)
	case *luau.CallStatement:
		n.Expression = sub(n.Expression)
	case *luau.DoStatement:
		n.Statements = subList(n.Statements)
	case *luau.WhileStatement:
		n.Condition = sub(n.Condition)
		n.Statements = subList(n.Statements)
	case *luau.RepeatStatement:
		n.Statements = subList(n.Statements)
		n.Condition = sub(n.Condition)
	case *luau.IfStatement:
		n.Condition = sub(n.Condition)
		n.Statements = subList(n.Statements)
		n.ElseBody = subListOrNode(n.ElseBody)
	case *luau.NumericForStatement:
		n.Id = sub(n.Id)
		n.Start = sub(n.Start)
		n.End = sub(n.End)
		n.Step = sub(n.Step)
		n.Statements = subList(n.Statements)
	case *luau.ForStatement:
		n.Ids = subList(n.Ids)
		n.Expression = sub(n.Expression)
		n.Statements = subList(n.Statements)
	case *luau.FunctionDeclaration:
		n.Name = sub(n.Name)
		n.Parameters = subList(n.Parameters)
		n.Statements = subList(n.Statements)
	case *luau.MethodDeclaration:
		n.Expression = sub(n.Expression)
		n.Parameters = subList(n.Parameters)
		n.Statements = subList(n.Statements)
	case *luau.VariableDeclaration:
		n.Left = subListOrNode(n.Left)
		n.Right = subListOrNode(n.Right)
	case *luau.ReturnStatement:
		n.Expression = subListOrNode(n.Expression)
	case *luau.MapField:
		n.Index = sub(n.Index)
		n.Value = sub(n.Value)
	}
}
