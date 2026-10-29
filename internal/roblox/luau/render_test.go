package luau

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

// render.json holds hand-written ASTs and their rendering by the reference TypeScript renderer
// (@roblox-ts/luau-ast 2.1.2). Regenerate it with testdata/oracle/generate.sh.

var nodeTypes = map[string]reflect.Type{}

func init() {
	for _, n := range []Node{
		&Identifier{}, &TemporaryIdentifier{}, &ComputedIndexExpression{}, &PropertyAccessExpression{},
		&CallExpression{}, &MethodCallExpression{}, &ParenthesizedExpression{}, &None{}, &NilLiteral{},
		&FalseLiteral{}, &TrueLiteral{}, &NumberLiteral{}, &StringLiteral{}, &VarArgsLiteral{},
		&FunctionExpression{}, &BinaryExpression{}, &UnaryExpression{}, &IfExpression{}, &InterpolatedString{},
		&Array{}, &Map{}, &Set{}, &MixedTable{}, &Assignment{}, &BreakStatement{}, &CallStatement{},
		&ContinueStatement{}, &DoStatement{}, &WhileStatement{}, &RepeatStatement{}, &IfStatement{},
		&NumericForStatement{}, &ForStatement{}, &FunctionDeclaration{}, &MethodDeclaration{},
		&VariableDeclaration{}, &ReturnStatement{}, &Comment{}, &MapField{}, &InterpolatedStringPart{},
	} {
		nodeTypes[GetKindName(n.Kind())] = reflect.TypeOf(n).Elem()
	}
}

var (
	nodeType       = reflect.TypeFor[Node]()
	listOrNodeType = reflect.TypeFor[ListOrNode]()
	listType       = reflect.TypeFor[*List]()
)

// decode builds the AST bottom-up with Create, so parents are set exactly as luau.create sets them.
func decode(t *testing.T, value any) ListOrNode {
	t.Helper()
	obj, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("not an object: %v", value)
	}
	if items, ok := obj["list"]; ok {
		list := MakeList()
		for _, item := range items.([]any) {
			list.Push(decode(t, item).(Node))
		}
		return list
	}
	typ, ok := nodeTypes[obj["kind"].(string)]
	if !ok {
		t.Fatalf("unknown kind %v", obj["kind"])
	}
	ptr := reflect.New(typ)
	for key, field := range obj {
		if key == "kind" {
			continue
		}
		name := strings.ToUpper(key[:1]) + key[1:]
		f := ptr.Elem().FieldByName(name)
		if !f.IsValid() {
			t.Fatalf("%s has no field %s", typ.Name(), name)
		}
		switch f.Type() {
		case nodeType, listOrNodeType, listType:
			f.Set(reflect.ValueOf(decode(t, field)))
		default:
			switch v := field.(type) {
			case string:
				f.SetString(v)
			case bool:
				f.SetBool(v)
			case float64:
				f.SetInt(int64(v))
			default:
				t.Fatalf("%s.%s: unexpected %T", typ.Name(), name, field)
			}
		}
	}
	return Create(ptr.Interface().(Node))
}

func TestRenderMatchesReference(t *testing.T) {
	data, err := os.ReadFile("testdata/render.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name   string
		Ast    any
		Output string
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures")
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			ast := decode(t, fixture.Ast).(*List)
			got := RenderAST(ast)
			if got != fixture.Output {
				t.Errorf("render mismatch\n--- got\n%s\n--- want\n%s", got, fixture.Output)
			}
		})
	}
}

func TestCreateClonesParentedChildren(t *testing.T) {
	id := Id("x")
	first := Call(id)
	second := Call(id)
	if first.Expression != Node(id) || id.Parent() != Node(first) {
		t.Fatal("first use should adopt the child")
	}
	if second.Expression == Node(id) || second.Expression.Parent() != Node(second) {
		t.Fatal("second use should adopt a clone")
	}
	// shared globals are never adopted, so concurrent transforms never write to them
	call := Call(Globals.TS)
	if call.Expression == Node(Globals.TS) || Globals.TS.Parent() != sharedParent {
		t.Fatal("globals must be cloned")
	}
}

func TestListOperations(t *testing.T) {
	a, b, c := Id("a"), Id("b"), Id("c")
	list := MakeList(a)
	list.Push(b)
	list.Unshift(c)
	if got := ListMapToArray(list, func(n Node) string { return n.(*Identifier).Name }); strings.Join(got, ",") != "c,a,b" {
		t.Fatalf("got %v", got)
	}
	other := MakeList(Id("d"))
	list.PushList(other)
	if !other.Readonly || list.Size() != 4 || list.Tail.Value.(*Identifier).Name != "d" {
		t.Fatal("PushList")
	}
	front := MakeList(Id("e"))
	list.UnshiftList(front)
	if list.Shift().(*Identifier).Name != "e" || list.Head.Prev != nil || list.Size() != 4 {
		t.Fatal("UnshiftList/Shift")
	}
	if MakeList().Shift() != nil {
		t.Fatal("Shift on empty list")
	}
}
