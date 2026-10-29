package luau

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/microsoft/typescript-go/internal/jsnum"
)

// Ports of LuauRenderer/nodes/**/*.ts, one function per file.

// expressions/indexable

func renderCallExpression(state *RenderState, node *CallExpression) string {
	return Render(state, node.Expression) + "(" + renderArguments(state, node.Args) + ")"
}

func renderComputedIndexExpression(state *RenderState, node *ComputedIndexExpression) string {
	expStr := Render(state, node.Expression)
	if index, ok := node.Index.(*StringLiteral); ok && IsValidIdentifier(index.Value) {
		return expStr + "." + index.Value
	}
	indexStr := Render(state, node.Index)
	return expStr + "[" + indexStr + "]"
}

func renderIdentifier(state *RenderState, node *Identifier) string {
	Assert(IsValidIdentifier(node.Name), `Invalid Luau Identifier: "`+node.Name+`"`)
	return node.Name
}

func renderMethodCallExpression(state *RenderState, node *MethodCallExpression) string {
	Assert(IsValidIdentifier(node.Name))
	return Render(state, node.Expression) + ":" + node.Name + "(" + renderArguments(state, node.Args) + ")"
}

func renderParenthesizedExpression(state *RenderState, node *ParenthesizedExpression) string {
	// skip nested parentheses
	expression := node.Expression
	for {
		p, ok := expression.(*ParenthesizedExpression)
		if !ok {
			break
		}
		expression = p.Expression
	}
	if IsSimple(expression) {
		return Render(state, node.Expression)
	}
	return "(" + Render(state, node.Expression) + ")"
}

func renderPropertyAccessExpression(state *RenderState, node *PropertyAccessExpression) string {
	expStr := Render(state, node.Expression)
	nameStr := node.Name
	if IsValidIdentifier(nameStr) {
		return expStr + "." + nameStr
	}
	return expStr + `["` + nameStr + `"]`
}

func renderTemporaryIdentifier(state *RenderState, node *TemporaryIdentifier) string {
	name := state.GetTempName(node)
	Assert(IsValidIdentifier(name), `Invalid Temporary Identifier: "`+name+`"`)
	return name
}

// expressions

func renderArray(state *RenderState, node *Array) string {
	if node.Members.IsEmpty() {
		return "{}"
	}
	membersStr := strings.Join(renderList(state, node.Members), ", ")
	return "{ " + membersStr + " }"
}

func renderBinaryExpression(state *RenderState, node *BinaryExpression) string {
	result := Render(state, node.Left) + " " + node.Operator + " " + Render(state, node.Right)
	if needsParentheses(node) {
		result = "(" + result + ")"
	}
	return result
}

func renderFunctionExpression(state *RenderState, node *FunctionExpression) string {
	if node.Statements.IsEmpty() {
		return "function(" + renderParameters(state, node.Parameters, node.HasDotDotDot) + ") end"
	}
	result := ""
	result += state.Newline("function(" + renderParameters(state, node.Parameters, node.HasDotDotDot) + ")")
	result += state.Block(func() string { return RenderStatements(state, node.Statements) })
	result += state.Indented("end")
	return result
}

func renderIfExpression(state *RenderState, node *IfExpression) string {
	result := "if " + Render(state, node.Condition) + " then " + Render(state, node.Expression) + " "

	currentAlternative := node.Alternative
	for {
		alternative, ok := currentAlternative.(*IfExpression)
		if !ok {
			break
		}
		condition := Render(state, alternative.Condition)
		expression := Render(state, alternative.Expression)
		result += "elseif " + condition + " then " + expression + " "
		currentAlternative = alternative.Alternative
	}

	result += "else " + Render(state, currentAlternative)

	if needsParentheses(node) {
		result = "(" + result + ")"
	}
	return result
}

func renderInterpolatedString(state *RenderState, node *InterpolatedString) string {
	var result strings.Builder
	result.WriteString("`")
	node.Parts.ForEach(func(part Node) {
		expressionStr := Render(state, part)
		if IsInterpolatedStringPart(part) {
			result.WriteString(expressionStr)
		} else {
			result.WriteString("{")
			// `{{}}` is invalid, so we wrap it in parenthesis
			if IsTable(part) {
				expressionStr = "(" + expressionStr + ")"
			}
			result.WriteString(expressionStr)
			result.WriteString("}")
		}
	})
	result.WriteString("`")
	return result.String()
}

func renderTableFields(state *RenderState, fields *List, renderField func(field Node) string) string {
	if fields.IsEmpty() {
		return "{}"
	}
	var result strings.Builder
	result.WriteString("{\n")
	state.Block(func() string {
		fields.ForEach(func(field Node) { result.WriteString(state.Line(renderField(field))) })
		return ""
	})
	result.WriteString(state.Indented("}"))
	return result.String()
}

func renderMap(state *RenderState, node *Map) string {
	return renderTableFields(state, node.Fields, func(field Node) string { return Render(state, field) + "," })
}

func renderMixedTable(state *RenderState, node *MixedTable) string {
	return renderTableFields(state, node.Fields, func(field Node) string { return Render(state, field) + "," })
}

func renderNumberLiteral(state *RenderState, node *NumberLiteral) string {
	if IsValidNumberLiteral(node.Value) {
		return node.Value
	}
	return jsnum.FromString(strings.ReplaceAll(node.Value, "_", "")).String()
}

func renderSet(state *RenderState, node *Set) string {
	return renderTableFields(state, node.Members, func(member Node) string {
		if s, ok := member.(*StringLiteral); ok && IsValidIdentifier(s.Value) {
			return s.Value + " = true,"
		}
		return "[" + Render(state, member) + "] = true,"
	})
}

// separates long-bracket string delimiters from surrounding indexing brackets to avoid ambiguous syntax
func needsBracketSpacing(node *StringLiteral) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}

	switch p := parent.(type) {
	case *MapField:
		// [ [[a]] ] = b
		return Node(node) == p.Index
	case *ComputedIndexExpression:
		// a[ [[b]] ]
		return Node(node) == p.Index
	case *Set:
		// { [ [[a]] ] = true }
		return true
	}
	return false
}

func renderStringLiteral(state *RenderState, node *StringLiteral) string {
	if node.Quote != "" {
		return node.Quote + node.Value + node.Quote
	}

	isMultiline := strings.Contains(node.Value, "\n")
	if !isMultiline && !strings.Contains(node.Value, `"`) {
		return `"` + node.Value + `"`
	} else if !isMultiline && !strings.Contains(node.Value, "'") {
		return "'" + node.Value + "'"
	}
	eqStr := getSafeBracketEquals(node.Value)
	spacing := ""
	if needsBracketSpacing(node) {
		spacing = " "
	}
	return spacing + "[" + eqStr + "[" + node.Value + "]" + eqStr + "]" + spacing
}

func unaryNeedsSpace(node *UnaryExpression) bool {
	// not always needs a space
	if node.Operator == "not" {
		return true
	}

	// "--" will create a comment!
	if inner, ok := node.Expression.(*UnaryExpression); ok && inner.Operator == "-" {
		// previous expression was also "-"
		return true
	}

	return false
}

func renderUnaryExpression(state *RenderState, node *UnaryExpression) string {
	opStr := node.Operator
	if unaryNeedsSpace(node) {
		opStr += " "
	}

	result := opStr + Render(state, node.Expression)
	if needsParentheses(node) {
		result = "(" + result + ")"
	}
	return result
}

// fields

var interpolatedStringPartRegex = regexp.MustCompile(`(\\(?:u\{[a-fA-F0-9]+\}|\r\n|[\s\S]))|([{}]|\r\n?|\n)`)

func renderInterpolatedStringPart(state *RenderState, node *InterpolatedStringPart) string {
	// consume complete escapes so an escaped backslash cannot start a unicode escape
	text := node.Text
	var result strings.Builder
	last := 0
	for _, m := range interpolatedStringPartRegex.FindAllStringSubmatchIndex(text, -1) {
		result.WriteString(text[last:m[0]])
		if m[2] >= 0 {
			result.WriteString(text[m[2]:m[3]])
		} else {
			result.WriteString(`\`)
			result.WriteString(text[m[4]:m[5]])
		}
		last = m[1]
	}
	result.WriteString(text[last:])
	return result.String()
}

func renderMapField(state *RenderState, node *MapField) string {
	valueStr := Render(state, node.Value)
	if index, ok := node.Index.(*StringLiteral); ok && IsValidIdentifier(index.Value) {
		return index.Value + " = " + valueStr
	}
	indexStr := Render(state, node.Index)
	return "[" + indexStr + "] = " + valueStr
}

// statements

func renderListOrNode(state *RenderState, value ListOrNode) string {
	if l, ok := value.(*List); ok {
		Assert(!l.IsEmpty())
		return strings.Join(renderList(state, l), ", ")
	}
	return Render(state, value.(Node))
}

func renderAssignment(state *RenderState, node *Assignment) string {
	leftStr := renderListOrNode(state, node.Left)
	rightStr := renderListOrNode(state, node.Right)
	return state.Line(leftStr+" "+node.Operator+" "+rightStr, node)
}

func renderBreakStatement(state *RenderState) string {
	return state.Line("break")
}

func renderCallStatement(state *RenderState, node *CallStatement) string {
	return state.Line(Render(state, node.Expression), node)
}

// jsTrimIsEmpty is `text.trim() === ""` with JavaScript's definition of whitespace.
func jsTrimIsEmpty(text string) bool {
	for _, r := range text {
		if !unicode.IsSpace(r) && r != 0xFEFF {
			return false
		}
	}
	return true
}

func renderComment(state *RenderState, node *Comment) string {
	lines := strings.Split(node.Text, "\n")
	if len(lines) > 1 {
		eqStr := getSafeBracketEquals(node.Text)
		result := state.Line("--[" + eqStr + "[")
		// indenting a blank line would only add trailing whitespace
		result += state.Block(func() string {
			var b strings.Builder
			for _, line := range lines {
				if jsTrimIsEmpty(line) {
					b.WriteString(state.Newline(""))
				} else {
					b.WriteString(state.Line(line))
				}
			}
			return b.String()
		})
		result += state.Line("]" + eqStr + "]")
		return result
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(state.Line("--" + line))
	}
	return b.String()
}

func renderContinueStatement(state *RenderState) string {
	return state.Line("continue")
}

func renderBlock(state *RenderState, statements *List) string {
	return state.Block(func() string { return RenderStatements(state, statements) })
}

func renderDoStatement(state *RenderState, node *DoStatement) string {
	result := ""
	result += state.Line("do")
	result += renderBlock(state, node.Statements)
	result += state.Line("end")
	return result
}

func renderForStatement(state *RenderState, node *ForStatement) string {
	idsStr := strings.Join(renderList(state, node.Ids), ", ")
	if idsStr == "" {
		idsStr = "_"
	}
	expStr := Render(state, node.Expression)

	result := ""
	result += state.Line("for " + idsStr + " in " + expStr + " do")
	result += renderBlock(state, node.Statements)
	result += state.Line("end")
	return result
}

func renderFunctionDeclaration(state *RenderState, node *FunctionDeclaration) string {
	if node.Localize {
		Assert(IsAnyIdentifier(node.Name), "local function cannot be a property")
	}
	nameStr := Render(state, node.Name)
	paramStr := renderParameters(state, node.Parameters, node.HasDotDotDot)

	local := ""
	if node.Localize {
		local = "local "
	}
	result := ""
	result += state.Line(local + "function " + nameStr + "(" + paramStr + ")")
	result += renderBlock(state, node.Statements)
	result += state.Line("end")
	return result
}

func renderIfStatement(state *RenderState, node *IfStatement) string {
	result := ""

	result += state.Line("if " + Render(state, node.Condition) + " then")
	result += renderBlock(state, node.Statements)

	currentElseBody := node.ElseBody
	for {
		elseIf, ok := currentElseBody.(*IfStatement)
		if !ok {
			break
		}
		result += state.Line("elseif " + Render(state, elseIf.Condition) + " then")
		result += renderBlock(state, elseIf.Statements)
		currentElseBody = elseIf.ElseBody
	}

	if statements, ok := currentElseBody.(*List); ok && statements != nil && statements.IsNonEmpty() {
		result += state.Line("else")
		result += renderBlock(state, statements)
	}

	result += state.Line("end")
	return result
}

func renderMethodDeclaration(state *RenderState, node *MethodDeclaration) string {
	result := ""
	result += state.Line("function " + Render(state, node.Expression) + ":" + node.Name + "(" + renderParameters(state, node.Parameters, node.HasDotDotDot) + ")")
	result += renderBlock(state, node.Statements)
	result += state.Line("end")
	return result
}

func renderNumericForStatement(state *RenderState, node *NumericForStatement) string {
	idStr := Render(state, node.Id)
	startStr := Render(state, node.Start)
	endStr := Render(state, node.End)

	predicateStr := startStr + ", " + endStr

	// step of 1 can be omitted
	if node.Step != nil {
		step, isNumber := node.Step.(*NumberLiteral)
		if !isNumber || jsnum.FromString(step.Value) != 1 {
			predicateStr += ", " + Render(state, node.Step)
		}
	}

	result := ""
	result += state.Line("for " + idStr + " = " + predicateStr + " do")
	result += renderBlock(state, node.Statements)
	result += state.Line("end")
	return result
}

func renderRepeatStatement(state *RenderState, node *RepeatStatement) string {
	result := ""
	result += state.Line("repeat")
	result += renderBlock(state, node.Statements)
	result += state.Line("until " + Render(state, node.Condition))
	return result
}

func renderReturnStatement(state *RenderState, node *ReturnStatement) string {
	var expStr string
	if l, ok := node.Expression.(*List); ok {
		expStr = strings.Join(renderList(state, l), ", ")
	} else {
		expStr = Render(state, node.Expression.(Node))
	}
	return state.Line("return " + expStr)
}

func renderVariableDeclaration(state *RenderState, node *VariableDeclaration) string {
	leftStr := renderListOrNode(state, node.Left)
	if node.Right != nil {
		rightStr := renderListOrNode(state, node.Right)
		return state.Line("local "+leftStr+" = "+rightStr, node)
	}
	return state.Line("local "+leftStr, node)
}

func renderWhileStatement(state *RenderState, node *WhileStatement) string {
	result := ""
	result += state.Line("while " + Render(state, node.Condition) + " do")
	result += renderBlock(state, node.Statements)
	result += state.Line("end")
	return result
}
