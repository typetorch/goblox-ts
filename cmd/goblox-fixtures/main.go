// Command goblox-fixtures is the fixture diff harness: it builds the roblox-ts test project of an oracle
// checkout with goblox and diffs every emitted .luau file against the oracle's own output.
//
//	go run ./cmd/goblox-fixtures -oracle <oracle dir> [-v] [-run <regexp>]
//
// With -lune it compiles the project and runs every spec under Lune instead (see lune.go):
//
//	go run ./cmd/goblox-fixtures -oracle <oracle dir> -lune [-compiler rbxtsc] [-optimizedLoops]
//
// With -diagnostic-fixtures it compiles each diagnostic fixture with both compilers and compares what
// they print (see diagnostics.go):
//
//	go run ./cmd/goblox-fixtures -oracle <oracle dir> -diagnostic-fixtures [-v] [-run <regexp>]
//
// With -cli it compares both compilers on CLI invocations, including the files written (see cli.go):
//
//	go run ./cmd/goblox-fixtures -oracle <oracle dir> -cli [-v] [-run <regexp>]
//
// The test project (<oracle>/tests) is copied to a temporary directory first, without its out/, so the
// oracle's expected output is never touched and the Rojo paths (out/...) still resolve. Each file prints
// one line: `match`, `differs` (a unified diff follows with -v), `missing` (goblox wrote no file),
// `extra` (goblox wrote a file the oracle didn't), or `panic: <message>` when its transform hit an
// unported stub (the other files still compile).
package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/diagnosticwriter"
	"github.com/microsoft/typescript-go/internal/roblox/project"
	"github.com/microsoft/typescript-go/internal/stringutil"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/peter-evans/patience"
)

func main() {
	os.Exit(run())
}

func run() int {
	oracle := flag.String("oracle", "", "oracle directory (the built roblox-ts checkout that contains tests/)")
	verbose := flag.Bool("v", false, "print a unified diff for every differing file and the full panic stacks")
	runPattern := flag.String("run", "", "only report files whose path (relative to out/) matches this regexp")
	keep := flag.Bool("keep", false, "keep the temporary project directory and print its path")
	showDiagnostics := flag.Bool("diagnostics", false, "print the build diagnostics")
	runSpecs := flag.Bool("lune", false, "compile the tests project and run every spec under Lune, like roblox-ts's `npm test`")
	luneBin := flag.String("lune-bin", "lune", "lune 0.8.8 executable (-lune)")
	rojoBin := flag.String("rojo-bin", "rojo", "rojo 7.4.4 executable (-lune)")
	compiler := flag.String("compiler", "goblox", `compiler for -lune: "goblox", or "rbxtsc" for the oracle's roblox-ts (the baseline)`)
	optimizedLoops := flag.Bool("optimizedLoops", false, "compile with --optimizedLoops, as compile.test.ts does (-lune)")
	diagnosticFixtures := flag.Bool("diagnostic-fixtures", false, "compile each diagnostic fixture with goblox and rbxtsc and compare their output")
	cli := flag.Bool("cli", false, "compare goblox and rbxtsc on CLI invocations: exit codes, output and written files")
	flag.Parse()
	if *oracle == "" {
		fmt.Fprintln(os.Stderr, "goblox-fixtures: -oracle is required")
		return 2
	}
	if *runSpecs {
		return runLune(luneOptions{
			oracle:         *oracle,
			luneBin:        *luneBin,
			rojoBin:        *rojoBin,
			compiler:       *compiler,
			optimizedLoops: *optimizedLoops,
			verbose:        *verbose,
			keep:           *keep,
		})
	}
	var filter *regexp.Regexp
	if *runPattern != "" {
		filter = regexp.MustCompile(*runPattern)
	}
	if *cli {
		return runCliScenarios(*oracle, filter, *verbose)
	}
	if *diagnosticFixtures {
		return runDiagnosticFixtures(*oracle, filter, *verbose)
	}

	testsDir := filepath.Join(*oracle, "tests")
	expectedDir := filepath.Join(testsDir, "out")
	if _, err := os.Stat(expectedDir); err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}

	tempDir, err := os.MkdirTemp("", "goblox-fixtures-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}
	// expand 8.3 short names (C:/Users/PHASEN~1) so paths match the ones the program sees
	if long, evalSymlinksErr := filepath.EvalSymlinks(tempDir); evalSymlinksErr == nil {
		tempDir = long
	}
	if *keep {
		fmt.Fprintln(os.Stderr, "project copy:", tempDir)
	} else {
		defer os.RemoveAll(tempDir)
	}
	if copyTreeErr := copyTree(testsDir, tempDir, func(rel string) bool {
		return rel == "out" || rel == "diagnostics.off"
	}); copyTreeErr != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}

	cwd := tspath.NormalizePath(tempDir)
	start := time.Now()
	result, err := project.Build(context.Background(), project.BuildOptions{
		Project:          cwd,
		CurrentDirectory: cwd,
		Overrides:        project.PartialProjectOptions{"allowCommentDirectives": true},
		KeepGoing:        true,
	})
	buildTime := time.Since(start)
	if err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures: build failed:", err)
		return 1
	}
	if *showDiagnostics {
		printDiagnostics(result, cwd)
	}

	// panics by output path
	panics := map[string]string{}
	for fileName, message := range result.Panics {
		rel, ok := strings.CutPrefix(fileName, cwd+"/src/")
		if !ok {
			continue
		}
		rel = strings.TrimSuffix(strings.TrimSuffix(rel, ".tsx"), ".ts") + ".luau"
		panics[rel] = message
	}

	expected, err := listLuau(expectedDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}
	actualDir := filepath.Join(tempDir, "out")
	actual, _ := listLuau(actualDir)

	names := map[string]bool{}
	for name := range expected {
		names[name] = true
	}
	for name := range actual {
		names[name] = true
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)

	counts := map[string]int{}
	for _, name := range sorted {
		if filter != nil && !filter.MatchString(name) {
			continue
		}
		status := ""
		detail := ""
		_, inExpected := expected[name]
		_, inActual := actual[name]
		switch {
		case panics[name] != "":
			message := panics[name]
			if *verbose {
				detail = message
			}
			if index := strings.IndexByte(message, '\n'); index >= 0 {
				message = message[:index]
			}
			status = "panic: " + message
		case !inActual:
			status = "missing"
		case !inExpected:
			status = "extra"
		default:
			want, _ := os.ReadFile(filepath.Join(expectedDir, name))
			got, _ := os.ReadFile(filepath.Join(actualDir, name))
			if string(want) == string(got) {
				status = "match"
			} else {
				status = "differs"
				if *verbose {
					lines := patience.Diff(stringutil.SplitLines(string(want)), stringutil.SplitLines(string(got)))
					detail = patience.UnifiedDiffTextWithOptions(lines, patience.UnifiedDiffOptions{
						Precontext:  3,
						Postcontext: 3,
						SrcHeader:   "oracle/" + name,
						DstHeader:   "goblox/" + name,
					})
				}
			}
		}
		key := status
		if strings.HasPrefix(key, "panic") {
			key = "panic"
		}
		counts[key]++
		fmt.Printf("%-50s %s\n", name, status)
		if detail != "" {
			fmt.Println(strings.TrimRight(detail, "\n"))
		}
	}

	errorCount := 0
	for _, d := range result.Diagnostics {
		if isError(d) {
			errorCount++
		}
	}
	fmt.Printf("\n%d match, %d differ, %d panic, %d missing, %d extra; %d diagnostics (%d errors); build %v\n",
		counts["match"], counts["differs"], counts["panic"], counts["missing"], counts["extra"],
		len(result.Diagnostics), errorCount, buildTime.Round(time.Millisecond))
	if counts["match"] == len(sorted) && filter == nil {
		return 0
	}
	return 1
}

func printDiagnostics(result *project.EmitResult, cwd string) {
	formatOpts := &diagnosticwriter.FormattingOptions{
		NewLine: "\n",
		ComparePathsOptions: tspath.ComparePathsOptions{
			CurrentDirectory: cwd,
		},
	}
	for _, d := range result.Diagnostics {
		diagnosticwriter.WriteFormatDiagnostic(os.Stdout, diagnosticwriter.WrapASTDiagnostic(d), formatOpts)
	}
}

func isError(d *ast.Diagnostic) bool {
	return d.Category().Name() == "error"
}

// listLuau returns the .luau files under dir, by forward-slash path relative to dir.
func listLuau(dir string) (map[string]bool, error) {
	files := map[string]bool{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".luau") {
			rel, _ := filepath.Rel(dir, path)
			files[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	return files, err
}

// copyTree copies src to dst, skipping the top-level entries for which skip returns true.
func copyTree(src, dst string, skip func(rel string) bool) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if rel == "." {
			return nil
		}
		if !strings.ContainsRune(filepath.ToSlash(rel), '/') && skip(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, contents, 0o644)
	})
}
