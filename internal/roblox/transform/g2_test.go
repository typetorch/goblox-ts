package transform

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func TestG2ValueToIdStr(t *testing.T) {
	cases := []struct {
		value    luau.Expression
		expected string
	}{
		{luau.Id("Foo"), "foo"},
		{luau.Property(luau.Id("game"), "Workspace"), "workspace"},
		{luau.Call(luau.Property(luau.Id("Vector3"), "new")), "vector3"},
		{luau.Call(luau.Id("foo")), ""},
		{luau.Id("end"), ""},
		{luau.Number(1), ""},
	}
	for _, c := range cases {
		if actual := valueToIdStr(c.value); actual != c.expected {
			t.Errorf("valueToIdStr(%s) = %q, want %q", luau.RenderAST(luau.MakeList(luau.Create(&luau.CallStatement{Expression: luau.Call(luau.Id("f"), c.value)}))), actual, c.expected)
		}
	}
}

func TestG2Assignment(t *testing.T) {
	writable := luau.Id("a")
	if actual := luau.RenderAST(luau.MakeList(createAssignmentStatement(writable, "+=", luau.Number(1), writable))); actual != "a += 1\n" {
		t.Errorf("same readable: %q", actual)
	}
	readable := luau.TempId("readable")
	actual := luau.RenderAST(luau.MakeList(
		luau.Create(&luau.VariableDeclaration{Left: readable, Right: luau.Id("a")}),
		createAssignmentStatement(writable, "..=", luau.String("x"), readable),
	))
	if actual != "local _readable = a\na = _readable .. \"x\"\n" {
		t.Errorf("separate readable: %q", actual)
	}
	if op := getSimpleAssignmentOperator(nil, ast.KindBarEqualsToken, nil); op != "" {
		t.Errorf("|= should have no simple operator, got %q", op)
	}
	if op := getSimpleAssignmentOperator(nil, ast.KindAsteriskAsteriskEqualsToken, nil); op != "^=" {
		t.Errorf("**= got %q", op)
	}
}

func TestG2Bitwise(t *testing.T) {
	if !isBitwiseOperator(ast.KindCaretToken) || !isBitwiseOperator(ast.KindGreaterThanGreaterThanGreaterThanEqualsToken) || isBitwiseOperator(ast.KindPlusToken) {
		t.Error("isBitwiseOperator")
	}
	call := createBitwiseCall(ast.KindGreaterThanGreaterThanToken, []luau.Expression{luau.Id("a"), luau.Number(2)})
	actual := luau.RenderAST(luau.MakeList(luau.Create(&luau.VariableDeclaration{Left: luau.Id("x"), Right: call})))
	if actual != "local x = bit32.arshift(a, 2)\n" {
		t.Errorf("createBitwiseCall: %q", actual)
	}
}

func TestG2IsSafeInteger(t *testing.T) {
	for value, expected := range map[float64]bool{0: true, -3: true, 1.5: false, 9007199254740991: true, 9007199254740992: false} {
		if isSafeInteger(value) != expected {
			t.Errorf("isSafeInteger(%v) != %v", value, expected)
		}
	}
}
