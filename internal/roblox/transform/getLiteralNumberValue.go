package transform

// Port of TSTransformer/util/getLiteralNumberValue.ts.

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/jsnum"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// getLiteralNumberValue is TSTransformer/util/getLiteralNumberValue.ts getLiteralNumberValue.
func getLiteralNumberValue(expression luau.Expression) (float64, bool) {
	switch e := expression.(type) {
	case *luau.ParenthesizedExpression:
		return getLiteralNumberValue(e.Expression)
	case *luau.NumberLiteral:
		return float64(jsnum.FromString(strings.ReplaceAll(e.Value, "_", ""))), true
	case *luau.UnaryExpression:
		if e.Operator == "-" {
			if innerValue, ok := getLiteralNumberValue(e.Expression); ok {
				return -innerValue, true
			}
		}
	}
	return 0, false
}
