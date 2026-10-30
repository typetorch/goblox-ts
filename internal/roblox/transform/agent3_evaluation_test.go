package transform

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func TestFactsSurviveCreateClones(t *testing.T) {
	t.Parallel()
	// luau.Globals.Self is shared, so every use is a clone; it must keep its CONSTANT fact.
	access := luau.Property(luau.Globals.Self, "x")
	if access.Expression == luau.Node(luau.Globals.Self) {
		t.Fatal("expected the shared global to be cloned")
	}
	if !isConstantReference(access.Expression) {
		t.Error("clone of self lost its constant reference fact")
	}
	if getEffects(access.Expression) != NO_EFFECTS {
		t.Error("expected self to have no effects")
	}
	// marking the clone must not mark the original's other clones
	id := luau.Id("a")
	parent := luau.Create(&luau.ParenthesizedExpression{Expression: id})
	other := luau.Create(&luau.ParenthesizedExpression{Expression: id})
	markPrimitiveValue(parent.Expression)
	if !isPrimitiveValue(parent.Expression) || isPrimitiveValue(other.Expression) {
		t.Error("facts must be per node, copied only at clone time")
	}
}

func TestEffects(t *testing.T) {
	t.Parallel()
	x := luau.Id("x")
	insert := luau.Call(luau.Globals.Table.Insert, x, luau.Number(1))
	effects := getEffects(insert)
	if !effects.WritesHeap || !effects.ReadsHeap || !effects.Throws || !effects.Reads["x"] || effects.Allocates {
		t.Errorf("table.insert(x, 1): unexpected effects %+v", effects)
	}
	unknown := getEffects(luau.Call(luau.Id("f")))
	if !unknown.Reads[USER_READS] || !unknown.Writes[USER_WRITES] {
		t.Errorf("f(): expected unknown effects, got %+v", unknown)
	}
	// a local temporary is invisible to user code, so an unknown call commutes with reading it
	temp := luau.TempId("t")
	if effectsCommute(getEffects(luau.Id("y")), unknown) {
		t.Error("an untracked user binding must not commute with an unknown call")
	}
	if !effectsCommute(getEffects(temp), unknown) {
		t.Error("a temporary must commute with an unknown call")
	}
	if joinEffects() != NO_EFFECTS || joinEffects(NO_EFFECTS, unknown) != unknown {
		t.Error("NO_EFFECTS must be the identity of joinEffects")
	}
	if !canRepeat(getEffects(luau.Binary(luau.Id("a"), "+", luau.Number(1)))) {
		t.Error("a + 1 must be repeatable")
	}
	if canRepeat(getEffects(luau.NewMap())) {
		t.Error("{} allocates and must not be repeatable")
	}
}

func TestSubstitute(t *testing.T) {
	t.Parallel()
	ref := luau.TempId("operand")
	expression := luau.Binary(ref, "+", luau.Number(1))
	got := substitute(expression, map[int]luau.Expression{ref.Id: luau.Binary(luau.Id("n"), "-", luau.Number(1))})
	if rendered := luau.RenderAST(luau.MakeList(luau.Create(&luau.ReturnStatement{Expression: got}))); rendered != "return n\n" {
		t.Errorf("expected offset folding to give `return n`, got %q", rendered)
	}
	call := luau.Call(ref)
	got = substitute(call, map[int]luau.Expression{ref.Id: luau.Create(&luau.FunctionExpression{
		Parameters: luau.MakeList(), Statements: luau.MakeList(),
	})})
	if !luau.IsParenthesizedExpression(got.(*luau.CallExpression).Expression) {
		t.Error("a substituted callee must be made indexable")
	}
}

func TestPlanEvaluation(t *testing.T) {
	t.Parallel()
	render := func(prereqs *Prereqs, result luau.Expression) string {
		statements := luau.MakeList()
		statements.PushList(prereqs.Statements)
		statements.Push(luau.Create(&luau.ReturnStatement{Expression: result}))
		return luau.RenderAST(statements)
	}
	// an operand used twice is captured when it cannot be repeated
	prereqs := NewPrereqs()
	result := planEvaluation(prereqs, []*EvaluationOperand{{Expression: luau.Call(luau.Id("f")), Prereqs: luau.MakeList()}},
		func(_ *Prereqs, references []luau.Expression) luau.Expression {
			return luau.Binary(references[0], "+", references[0])
		})
	if got, want := render(prereqs, result), "local _exp = f()\nreturn _exp + _exp\n"; got != want {
		t.Errorf("repeated call:\ngot:\n%swant:\n%s", got, want)
	}
	// a repeatable operand is inlined
	prereqs = NewPrereqs()
	result = planEvaluation(prereqs, []*EvaluationOperand{{Expression: luau.Id("a"), Prereqs: luau.MakeList()}},
		func(_ *Prereqs, references []luau.Expression) luau.Expression {
			return luau.Binary(references[0], "*", references[0])
		})
	if got, want := render(prereqs, result), "return a * a\n"; got != want {
		t.Errorf("repeated identifier:\ngot:\n%swant:\n%s", got, want)
	}
	// an unused operand with effects is kept as a statement
	prereqs = NewPrereqs()
	result = planEvaluation(prereqs, []*EvaluationOperand{{Expression: luau.Call(luau.Id("f")), Prereqs: luau.MakeList()}},
		func(_ *Prereqs, _ []luau.Expression) luau.Expression { return luau.Nil() })
	if got, want := render(prereqs, result), "f()\nreturn nil\n"; got != want {
		t.Errorf("unused call:\ngot:\n%swant:\n%s", got, want)
	}
}
