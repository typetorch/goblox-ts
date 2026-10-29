package luau

import (
	"strconv"
	"strings"
)

// Port of LuauRenderer/render.ts and LuauRenderer/util/*.ts.

func itoa(i int) string { return strconv.Itoa(i) }

// Render returns a string that represents the given syntax node as Luau code. It is called
// recursively until the node is completely rendered.
func Render(state *RenderState, node Node) string {
	switch n := node.(type) {
	// indexable expressions
	case *Identifier:
		return renderIdentifier(state, n)
	case *TemporaryIdentifier:
		return renderTemporaryIdentifier(state, n)
	case *ComputedIndexExpression:
		return renderComputedIndexExpression(state, n)
	case *PropertyAccessExpression:
		return renderPropertyAccessExpression(state, n)
	case *CallExpression:
		return renderCallExpression(state, n)
	case *MethodCallExpression:
		return renderMethodCallExpression(state, n)
	case *ParenthesizedExpression:
		return renderParenthesizedExpression(state, n)

	// expressions
	case *None:
		Assert(false, "Cannot render None")
	case *NilLiteral:
		return "nil"
	case *FalseLiteral:
		return "false"
	case *TrueLiteral:
		return "true"
	case *NumberLiteral:
		return renderNumberLiteral(state, n)
	case *StringLiteral:
		return renderStringLiteral(state, n)
	case *VarArgsLiteral:
		return "..."
	case *FunctionExpression:
		return renderFunctionExpression(state, n)
	case *BinaryExpression:
		return renderBinaryExpression(state, n)
	case *UnaryExpression:
		return renderUnaryExpression(state, n)
	case *IfExpression:
		return renderIfExpression(state, n)
	case *InterpolatedString:
		return renderInterpolatedString(state, n)
	case *Array:
		return renderArray(state, n)
	case *Map:
		return renderMap(state, n)
	case *Set:
		return renderSet(state, n)
	case *MixedTable:
		return renderMixedTable(state, n)

	// statements
	case *Assignment:
		return renderAssignment(state, n)
	case *BreakStatement:
		return renderBreakStatement(state)
	case *CallStatement:
		return renderCallStatement(state, n)
	case *ContinueStatement:
		return renderContinueStatement(state)
	case *DoStatement:
		return renderDoStatement(state, n)
	case *WhileStatement:
		return renderWhileStatement(state, n)
	case *RepeatStatement:
		return renderRepeatStatement(state, n)
	case *IfStatement:
		return renderIfStatement(state, n)
	case *NumericForStatement:
		return renderNumericForStatement(state, n)
	case *ForStatement:
		return renderForStatement(state, n)
	case *FunctionDeclaration:
		return renderFunctionDeclaration(state, n)
	case *MethodDeclaration:
		return renderMethodDeclaration(state, n)
	case *VariableDeclaration:
		return renderVariableDeclaration(state, n)
	case *ReturnStatement:
		return renderReturnStatement(state, n)
	case *Comment:
		return renderComment(state, n)

	// fields
	case *MapField:
		return renderMapField(state, n)
	case *InterpolatedStringPart:
		return renderInterpolatedStringPart(state, n)
	}
	panic("luau.Render: unknown node")
}

// RenderAST returns a string that represents the given syntax tree as Luau code.
func RenderAST(ast *List) string {
	state := NewRenderState()
	SolveTempIds(state, ast)
	return RenderStatements(state, ast)
}

// RenderStatements renders the given list of statements. Each list node is pushed onto the state's
// list node stack while it renders, so statements can look at their next sibling.
func RenderStatements(state *RenderState, statements *List) string {
	var result strings.Builder
	hasFinalStatement := false
	for listNode := statements.Head; listNode != nil; listNode = listNode.Next {
		Assert(!hasFinalStatement || IsComment(listNode.Value), "Cannot render statement after break, continue, or return!")
		hasFinalStatement = hasFinalStatement || IsFinalStatement(listNode.Value)

		state.PushListNode(listNode)
		result.WriteString(Render(state, listNode.Value))
		state.PopListNode()
	}
	return result.String()
}

func renderList(state *RenderState, list *List) []string {
	return ListMapToArray(list, func(v Node) string { return Render(state, v) })
}

// renderArguments renders the given list of expressions into a string separated by commas.
func renderArguments(state *RenderState, expressions *List) string {
	return strings.Join(renderList(state, expressions), ", ")
}

// renderParameters renders the parameters of a function-like node separated by commas, adding `...`
// when hasDotDotDot is true.
func renderParameters(state *RenderState, parameters *List, hasDotDotDot bool) string {
	paramStrs := renderList(state, parameters)
	if hasDotDotDot {
		paramStrs = append(paramStrs, "...")
	}
	return strings.Join(paramStrs, ", ")
}

func getSafeBracketEquals(str string) string {
	amtEquals := 0
	for strings.Contains(str, "]"+strings.Repeat("=", amtEquals)+"]") || strings.HasSuffix(str, "]"+strings.Repeat("=", amtEquals)) {
		amtEquals++
	}
	return strings.Repeat("=", amtEquals)
}
