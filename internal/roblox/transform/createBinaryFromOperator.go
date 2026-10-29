package transform

// Port of TSTransformer/util/createBinaryFromOperator.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// binaryOperatorMap is OPERATOR_MAP of createBinaryFromOperator.ts.
var binaryOperatorMap = map[ast.Kind]luau.BinaryOperator{
	// comparison
	ast.KindLessThanToken:                "<",
	ast.KindGreaterThanToken:             ">",
	ast.KindLessThanEqualsToken:          "<=",
	ast.KindGreaterThanEqualsToken:       ">=",
	ast.KindEqualsEqualsEqualsToken:      "==",
	ast.KindExclamationEqualsEqualsToken: "~=",

	// math
	ast.KindMinusToken:            "-",
	ast.KindAsteriskToken:         "*",
	ast.KindSlashToken:            "/",
	ast.KindAsteriskAsteriskToken: "^",
	ast.KindPercentToken:          "%",
}

func createBinaryAdd(left luau.Expression, leftType *checker.Type, right luau.Expression, rightType *checker.Type) luau.Expression {
	leftIsString := isDefinitelyType(leftType, isStringType)
	rightIsString := isDefinitelyType(rightType, isStringType)
	if leftIsString || rightIsString {
		if !leftIsString {
			left = luau.Call(luau.Globals.Tostring, left)
		}
		if !rightIsString {
			right = luau.Call(luau.Globals.Tostring, right)
		}
		return luau.Binary(left, "..", right)
	} else {
		return luau.Binary(left, "+", right)
	}
}

// createBinaryFromOperator is TSTransformer/util/createBinaryFromOperator.ts createBinaryFromOperator.
func createBinaryFromOperator(prereqs *Prereqs, left luau.Expression, leftType *checker.Type, operatorKind ast.Kind, right luau.Expression, rightType *checker.Type) luau.Expression {
	// arithmetic and comparison instructions can read a local after the RHS call ran
	if isLateRead(left) && !effectsCommute(getEffects(left), getEffects(right)) {
		left = prereqs.PushToVar(left, "left")
	}

	// simple
	if operator, ok := binaryOperatorMap[operatorKind]; ok {
		return luau.Binary(left, operator, right)
	}

	// plus
	if operatorKind == ast.KindPlusToken || operatorKind == ast.KindPlusEqualsToken {
		return createBinaryAdd(left, leftType, right, rightType)
	}

	// bitwise assignment
	if isBitwiseOperator(operatorKind) {
		return createBitwiseCall(operatorKind, []luau.Expression{left, right})
	}

	if operatorKind == ast.KindCommaToken {
		prereqs.PushList(wrapExpressionStatement(left))
		return right
	}

	assert(false, "createBinaryFromOperator unknown operator: "+getKindName(operatorKind))
	return nil
}
