package luau

// Port of LuauRenderer/util/needsParentheses.ts.

// https://www.lua.org/manual/5.1/manual.html#2.5.6
/*
	1. or
	2. and
	3. <     >     <=    >=    ~=    ==
	4. ..
	5. +     -
	6. *     /     %
	7. not   #     - (unary)
	8. ^
*/

const ifExpressionPrecedence = 1

var unaryOperatorPrecedence = map[UnaryOperator]int{
	"not": 7,
	"#":   7,
	"-":   7,
}

var binaryOperatorPrecedence = map[BinaryOperator]int{
	"or":  1,
	"and": 2,
	"<":   3,
	">":   3,
	"<=":  3,
	">=":  3,
	"~=":  3,
	"==":  3,
	"..":  4,
	"+":   5,
	"-":   5,
	"*":   6,
	"/":   6,
	"//":  6,
	"%":   6,
	"^":   8,
}

func getPrecedence(node Node) int {
	switch n := node.(type) {
	case *IfExpression:
		return ifExpressionPrecedence
	case *BinaryExpression:
		return binaryOperatorPrecedence[n.Operator]
	case *UnaryExpression:
		return unaryOperatorPrecedence[n.Operator]
	}
	Assert(false)
	return 0
}

func needsParentheses(node Node) bool {
	parent := node.Parent()
	if parent != nil && IsExpressionWithPrecedence(parent) {
		nodePrecedence := getPrecedence(node)
		parentPrecedence := getPrecedence(parent)
		if nodePrecedence < parentPrecedence {
			return true
		} else if nodePrecedence == parentPrecedence {
			if b, ok := parent.(*BinaryExpression); ok {
				return node == b.Right
			}
			return false
		}
	}
	return false
}
