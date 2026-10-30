package transform

import (
	"reflect"
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/ls/lsutil"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// lastTokenBySearching is getLastToken without its shortcuts.
func lastTokenBySearching(file *ast.SourceFile, parent *ast.Node, statements []*ast.Node) *ast.Node {
	if len(statements) > 0 {
		lastStatement := statements[len(statements)-1]
		lastToken := lsutil.GetLastToken(lastStatement.Parent, file)
		if lastToken != nil && !isNodeDescendantOf(lastToken, lastStatement) {
			return lastToken
		}
		return nil
	}
	return lsutil.GetLastToken(parent, file)
}

func commentTexts(comments *luau.List) []string {
	texts := []string{}
	comments.ForEach(func(comment luau.Node) { texts = append(texts, comment.(*luau.Comment).Text) })
	return texts
}

func TestLastTokenShortcuts(t *testing.T) {
	t.Parallel()
	for _, source := range []string{
		`
// a
class A extends B {
	constructor() { // b
		super(); // c
		// d
		f(); // e
		// f
	}
}
function g() { /* g */ }
function h() {}
namespace N { f(); /* n */ }
namespace M { /* m */ }
switch (x) {
	case 1: f(); /* i */
	case 2: /* j */
	default: { f(); /* k */ }
}
if (x) f(); /* l */
// end
`,
		"// only a comment\n",
		"",
	} {
		file := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts", Path: "/test.ts"}, source, core.ScriptKindTS)
		state := &TransformState{SourceFile: file, sourceFileText: source}

		check := func(parent *ast.Node, statements []*ast.Node) {
			t.Helper()
			want := []string{}
			if token := lastTokenBySearching(file, parent, statements); token != nil {
				want = commentTexts(state.GetLeadingComments(token))
			}
			got := []string{}
			if pos, ok := getLastTokenPos(statements); ok {
				got = commentTexts(state.getLeadingCommentsAt(pos))
			} else if token := getLastToken(state, parent, statements); token != nil {
				got = commentTexts(state.GetLeadingComments(token))
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%v with %d statements at %d: comments %q, want %q", parent.Kind, len(statements), parent.Pos(), got, want)
			}
		}

		visited := 0
		var visit func(node *ast.Node) bool
		visit = func(node *ast.Node) bool {
			switch node.Kind {
			case ast.KindSourceFile, ast.KindBlock, ast.KindModuleBlock, ast.KindCaseClause, ast.KindDefaultClause:
				visited++
				statements := node.Statements()
				// every split, as a constructor renders the statements around its super call separately
				for i := range len(statements) + 1 {
					check(node, statements[:i])
					check(node, statements[i:])
				}
			}
			node.ForEachChild(visit)
			return false
		}
		visit(file.AsNode())
		if visited == 0 {
			t.Errorf("no statement lists in %q", source)
		}
	}
}
