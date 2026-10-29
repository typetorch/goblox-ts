package transform

// Port of TSTransformer/macros/types.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

type MacroList[T any] = map[string]T

type IdentifierMacro = func(state *TransformState, node *ast.Node) luau.Expression

type ConstructorMacro = func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression

// operands are opaque values: assigning to them or transforming their source again
// bypasses evaluation planning. Use pushToVar only for the macro's own working state.
type CallMacro = func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression

// PropertyCallMacro's node is a CallExpression whose expression is a PropertyAccessExpression or an
// ElementAccessExpression.
type PropertyCallMacro = func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression
