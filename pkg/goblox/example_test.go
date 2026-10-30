package goblox_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/pkg/goblox"
)

// doublePlugin replaces every call `double(x)` with `x * 2`. It asks the checker which function is
// being called, so a local function that happens to be named `double` is left alone.
type doublePlugin struct{}

func (doublePlugin) Name() string { return "double" }

func (doublePlugin) TransformSourceFile(ctx *goblox.TransformContext, file *goblox.SourceFile) *goblox.SourceFile {
	var visitor *goblox.NodeVisitor
	visitor = ctx.EmitContext.NewNodeVisitor(func(node *goblox.Node) *goblox.Node {
		if node.Kind == goblox.KindCallExpression {
			call := node.AsCallExpression()
			// Query the checker with the original node, before anything replaced it.
			symbol := ctx.Checker.GetSymbolAtLocation(call.Expression)
			if symbol != nil && symbol.Name == "double" && len(symbol.Declarations) > 0 &&
				goblox.GetSourceFileOfNode(symbol.Declarations[0]).IsDeclarationFile {
				arg := visitor.VisitNode(call.Arguments.Nodes[0])
				f := ctx.Factory
				return f.NewBinaryExpression(nil, arg, nil, f.NewToken(goblox.KindAsteriskToken), f.NewNumericLiteral("2", goblox.TokenFlagsNone))
			}
		}
		return visitor.VisitEachChild(node)
	})
	return visitor.VisitSourceFile(file)
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, text := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExamplePlugin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"tsconfig.json":   `{ "compilerOptions": { "strict": true, "target": "esnext", "module": "commonjs" }, "include": ["src"] }`,
		"src/macros.d.ts": `declare function double(value: number): number;`,
		"src/index.ts": "export const a = double(21);\n" +
			"function shadow() { const double = (v: number) => v; return double(1); }\n" +
			"export const b: number = a + shadow();\n",
	})

	ctx := context.Background()
	program, errs := goblox.LoadProgram(goblox.LoadOptions{Project: dir, CurrentDirectory: dir, Checkers: 1})
	if program == nil {
		t.Fatalf("LoadProgram: %v", errs)
	}

	result := goblox.RunPlugins(ctx, program, []goblox.Plugin{doublePlugin{}}, goblox.PluginOptions{CurrentDirectory: dir})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("plugin diagnostics: %v", result.Diagnostics)
	}
	if result.Program == program {
		t.Fatal("expected a new program")
	}

	index := filepath.ToSlash(filepath.Join(dir, "src/index.ts"))
	text, ok := result.Transformed[goblox.NormalizePath(index)]
	if !ok {
		t.Fatalf("index.ts was not transformed: %v", result.Transformed)
	}
	if !strings.Contains(text, "export const a = 21 * 2;") || !strings.Contains(text, "return double(1);") {
		t.Fatalf("unexpected output:\n%s", text)
	}

	// The transformed file is re-parsed and bound: its nodes are type-checkable.
	file := result.Program.GetSourceFile(goblox.NormalizePath(index))
	if diags := goblox.GetPreEmitDiagnostics(ctx, result.Program, file); len(diags) != 0 {
		var b strings.Builder
		goblox.WriteDiagnostics(&b, diags, dir)
		t.Fatalf("transformed program has diagnostics:\n%s", b.String())
	}
	checker, done := result.Program.GetTypeCheckerForFile(ctx, file)
	defer done()
	initializer := file.Statements.Nodes[0].AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes[0].Initializer()
	if got := checker.TypeToString(checker.GetTypeAtLocation(initializer)); got != "number" {
		t.Fatalf("type of `21 * 2` = %q, want number", got)
	}

	// Unchanged files are reused, not parsed again.
	macros := goblox.NormalizePath(filepath.Join(dir, "src/macros.d.ts"))
	if program.GetSourceFile(macros) != result.Program.GetSourceFile(macros) {
		t.Fatal("unchanged file was parsed again")
	}
}
