package diagnostics

import (
	"reflect"

	"github.com/microsoft/typescript-go/internal/core"
	"testing"

	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
)

// roblox-ts assigns ids in declaration order; the diagnostic fixtures refer to them.
func TestIdsFollowDeclarationOrder(t *testing.T) {
	v := reflect.ValueOf(Errors)
	next := 0
	for i := range v.NumField() {
		f, ok := v.Field(i).Interface().(*DiagnosticFactory)
		if !ok {
			continue // text diagnostics have no id
		}
		if f.Id != next {
			t.Fatalf("%s has id %d, want %d", v.Type().Field(i).Name, f.Id, next)
		}
		next++
	}
	if Warnings.RuntimeLibUsedInReplicatedFirst.Id != next {
		t.Fatalf("RuntimeLibUsedInReplicatedFirst has id %d, want %d", Warnings.RuntimeLibUsedInReplicatedFirst.Id, next)
	}
	if next != 65 {
		t.Fatalf("%d error factories, roblox-ts 68648b0 has 65", next)
	}
}

func TestMessages(t *testing.T) {
	ColorEnabled = false
	d := Errors.NoRojoData.CreateWithRange(nil, core.NewTextRange(0, 0), "src/a.ts", true)
	want := "Could not find Rojo data. There is no $path in your Rojo config that covers src/a.ts\nSuggestion: Did you forget to add a custom npm scope to your default.project.json?"
	if d.MessageText() != want {
		t.Fatalf("got %q", d.MessageText())
	}
	if d.Source() != Source || GetDiagnosticId(d) != Errors.NoRojoData.Id || d.Category() != tsdiagnostics.CategoryError {
		t.Fatal("source, id or category")
	}
	f := Errors.FailedSymlinkResolve.messageText([]any{"/p/node_modules", "/x/index.d.ts", "", "@rbxts/x"})
	if f != "Cannot resolve package entry point inside the project node_modules!\nCheck the package main field and any package symlinks.\nPackages are expected to be installed under /p/node_modules\nPackage declarations resolved to /x/index.d.ts\nThe path from node_modules to the package would be @rbxts/x" {
		t.Fatalf("got %q", f)
	}
}
