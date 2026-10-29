package luau

import (
	"math"
	"sync/atomic"

	"github.com/microsoft/typescript-go/internal/jsnum"
)

// Port of LuauAST/impl/create.ts (except `create` itself, which is Create in node.go).

var lastTempId atomic.Int64

// TempId creates a new temporary identifier for a node.
func TempId(name ...string) *TemporaryIdentifier {
	n := ""
	if len(name) > 0 {
		n = name[0]
	}
	return Create(&TemporaryIdentifier{Name: n, Id: int(lastTempId.Add(1) - 1)})
}

// NewNone creates a new `None` node.
func NewNone() *None { return Create(&None{}) }

// Nil creates a new `nil` literal node.
func Nil() *NilLiteral { return Create(&NilLiteral{}) }

// Bool creates a new `boolean` literal node.
func Bool(value bool) Expression {
	if value {
		return Create(&TrueLiteral{})
	}
	return Create(&FalseLiteral{})
}

// Number creates a new `number` literal node, rendering value like JavaScript's String(value).
func Number(value float64) Expression {
	if value >= 0 {
		return Create(&NumberLiteral{Value: jsnum.Number(value).String()})
	}
	return Create(&UnaryExpression{
		Operator:   "-",
		Expression: Number(math.Abs(value)),
	})
}

// String creates a new `string` literal node. quote is an explicit delimiter (`"` or `'`); the value
// must already be escaped for it.
func String(value string, quote ...string) *StringLiteral {
	q := ""
	if len(quote) > 0 {
		q = quote[0]
	}
	return Create(&StringLiteral{Value: value, Quote: q})
}

// Id creates a new identifier node.
func Id(name string) *Identifier { return Create(&Identifier{Name: name}) }

// NewComment creates a new comment node.
func NewComment(text string) *Comment { return Create(&Comment{Text: text}) }

// NewArray creates a new array node.
func NewArray(members ...Expression) *Array {
	return Create(&Array{Members: MakeList(members...)})
}

// NewSet creates a new set node.
func NewSet(members ...Expression) *Set {
	return Create(&Set{Members: MakeList(members...)})
}

// MapEntry is one `[index, value]` pair for NewMap and NewMixedTable.
type MapEntry struct {
	Index Expression
	Value Expression
}

// NewMap creates a new map node.
func NewMap(fields ...MapEntry) *Map {
	nodes := make([]Node, len(fields))
	for i, f := range fields {
		nodes[i] = Create(&MapField{Index: f.Index, Value: f.Value})
	}
	return Create(&Map{Fields: MakeList(nodes...)})
}

// NewMixedTable creates a new mixed table node. Each field is either an Expression or a MapEntry.
func NewMixedTable(fields ...any) *MixedTable {
	nodes := make([]Node, len(fields))
	for i, f := range fields {
		switch f := f.(type) {
		case MapEntry:
			nodes[i] = Create(&MapField{Index: f.Index, Value: f.Value})
		case Node:
			nodes[i] = f
		default:
			Assert(false, "NewMixedTable field must be a Node or a MapEntry")
		}
	}
	return Create(&MixedTable{Fields: MakeList(nodes...)})
}

// Binary creates a new binary expression node.
func Binary(left Expression, operator BinaryOperator, right Expression) *BinaryExpression {
	return Create(&BinaryExpression{Left: left, Operator: operator, Right: right})
}

// Unary creates a new unary expression node.
func Unary(operator UnaryOperator, expression Expression) *UnaryExpression {
	return Create(&UnaryExpression{Operator: operator, Expression: expression})
}

// Property creates a new property access expression node.
func Property(expression IndexableExpression, name string) *PropertyAccessExpression {
	return Create(&PropertyAccessExpression{Expression: expression, Name: name})
}

// Call creates a new call expression node.
func Call(expression IndexableExpression, args ...Expression) *CallExpression {
	return Create(&CallExpression{Expression: expression, Args: MakeList(args...)})
}
