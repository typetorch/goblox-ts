package rojo

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/internal/json"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs/osvfs"
)

// expected.json is the output of the reference rojo-resolver 1.3.0 and path-translator 1.1.1 for
// queries.json (testdata/oracle/generate.sh). This test runs the same queries through the port.

type queries struct {
	Projects []struct {
		Config string   `json:"config"`
		Files  []string `json:"files"`
	} `json:"projects"`
	Find      []string `json:"find"`
	Synthetic struct {
		Base  string   `json:"base"`
		Files []string `json:"files"`
	} `json:"synthetic"`
	Relative   [][2][]string `json:"relative"`
	RbxPaths   [][]string    `json:"rbxPaths"`
	Translator struct {
		RootDir string   `json:"rootDir"`
		OutDir  string   `json:"outDir"`
		Files   []string `json:"files"`
		Outputs []string `json:"outputs"`
	} `json:"translator"`
}

func TestMatchesReference(t *testing.T) {
	t.Parallel()
	rootAbs, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	root := tspath.NormalizePath(rootAbs)
	var q queries
	readJSON(t, "testdata/queries.json", &q)
	var expected any
	readJSON(t, "testdata/expected.json", &expected)

	fs := osvfs.FS()
	abs := func(p string) string { return root + "/" + p }
	rel := func(p string) string { return strings.ReplaceAll(p, root, "<root>") }
	relAll := func(ps []string) []string {
		out := []string{}
		for _, p := range ps {
			out = append(out, rel(p))
		}
		return out
	}
	describe := func(r *RojoResolver, files []string) map[string]any {
		partitions := []any{}
		for _, p := range r.GetPartitions() {
			partitions = append(partitions, map[string]any{"fsPath": rel(p.FsPath), "rbxPath": p.RbxPath})
		}
		fileResults := []any{}
		for _, f := range files {
			rbxPath := r.GetRbxPathFromFilePath(abs(f))
			entry := map[string]any{"file": f, "rbxPath": nil, "rbxType": r.GetRbxTypeFromFilePath(abs(f)), "networkType": nil, "isolated": nil}
			if rbxPath != nil {
				entry["rbxPath"] = rbxPath
				entry["networkType"] = r.GetNetworkType(rbxPath)
				entry["isolated"] = r.IsIsolated(rbxPath)
			}
			fileResults = append(fileResults, entry)
		}
		return map[string]any{"warnings": relAll(r.GetWarnings()), "isGame": r.IsGame, "partitions": partitions, "files": fileResults}
	}

	result := map[string]any{}
	projects := []any{}
	for _, p := range q.Projects {
		projects = append(projects, describe(FromPath(fs, abs(p.Config)), p.Files))
	}
	result["projects"] = projects

	find := []any{}
	for _, p := range q.Find {
		path, warnings := FindRojoConfigFilePath(fs, abs(p))
		var pathValue any
		if path != "" {
			pathValue = rel(path)
		}
		find = append(find, map[string]any{"path": pathValue, "warnings": relAll(warnings)})
	}
	result["find"] = find
	result["synthetic"] = describe(Synthetic(fs, abs(q.Synthetic.Base)), q.Synthetic.Files)

	relative := []any{}
	for _, pair := range q.Relative {
		parts := []string{}
		for _, part := range Relative(pair[0], pair[1]) {
			if part.IsParent {
				parts = append(parts, "<parent>")
			} else {
				parts = append(parts, part.Name)
			}
		}
		relative = append(relative, parts)
	}
	result["relative"] = relative

	game := FromPath(fs, abs(q.Projects[0].Config))
	fileRelations := []any{}
	networkTypes := []any{}
	for _, a := range q.RbxPaths {
		row := []any{}
		for _, b := range q.RbxPaths {
			row = append(row, game.GetFileRelation(a, b))
		}
		fileRelations = append(fileRelations, row)
		networkTypes = append(networkTypes, game.GetNetworkType(a))
	}
	result["fileRelations"] = fileRelations
	result["networkTypes"] = networkTypes

	translator := []any{}
	for _, declaration := range []bool{false, true} {
		for _, luau := range []bool{false, true} {
			tr := NewPathTranslator(abs(q.Translator.RootDir), abs(q.Translator.OutDir), "", declaration, luau)
			each := func(files []string, f func(string) string) []string {
				out := []string{}
				for _, file := range files {
					out = append(out, rel(f(abs(file))))
				}
				return out
			}
			inputPaths := []any{}
			for _, f := range q.Translator.Outputs {
				inputPaths = append(inputPaths, relAll(tr.GetInputPaths(abs(f))))
			}
			translator = append(translator, map[string]any{
				"declaration":           declaration,
				"luau":                  luau,
				"outputPath":            each(q.Translator.Files, tr.GetOutputPath),
				"outputDeclarationPath": each(q.Translator.Files, tr.GetOutputDeclarationPath),
				"outputTransformedPath": each(q.Translator.Files, tr.GetOutputTransformedPath),
				"importPath":            each(q.Translator.Files, func(f string) string { return tr.GetImportPath(f, false) }),
				"importPathNodeModule":  each(q.Translator.Files, func(f string) string { return tr.GetImportPath(f, true) }),
				"inputPaths":            inputPaths,
			})
		}
	}
	result["translator"] = translator

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var got any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"projects", "find", "synthetic", "relative", "fileRelations", "networkTypes", "translator"} {
		g, w := got.(map[string]any)[key], expected.(map[string]any)[key]
		if !reflect.DeepEqual(g, w) {
			gj, err := json.MarshalIndent(g, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			wj, err := json.MarshalIndent(w, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			t.Errorf("%s mismatch\n--- got\n%s\n--- want\n%s", key, gj, wj)
		}
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatal(err)
	}
}
