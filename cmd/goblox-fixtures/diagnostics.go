package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/internal/stringutil"
	"github.com/peter-evans/patience"
)

var tsconfigReferencesPattern = regexp.MustCompile(`(?m)^\s*"references":.*$`)

// compilerRun is what one compiler did with one project: its exit code and everything it printed.
type compilerRun struct {
	exitCode int
	stdout   string
	stderr   string
}

func runCompiler(dir string, env []string, name string, args ...string) (compilerRun, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	run := compilerRun{stdout: stdout.String(), stderr: stderr.String()}
	if exitError, ok := errors.AsType[*exec.ExitError](err); ok {
		run.exitCode = exitError.ExitCode()
	} else if err != nil {
		return run, err
	}
	return run, nil
}

// buildGoblox builds cmd/goblox into dir and returns the executable's path.
func buildGoblox(dir string) (string, error) {
	name := filepath.Join(dir, "goblox")
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	output, err := exec.Command("go", "build", "-o", name, "./cmd/goblox").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go build ./cmd/goblox: %w\n%s", err, output)
	}
	return name, nil
}

// compareCompilers compiles the project in dir with both compilers and describes the differences, or
// returns "" if they exit with the same code and print the same bytes (the printed diagnostics include
// the code, the message and, through the source context, the span).
func compareCompilers(dir string, oracle string, goblox string, flags ...string) (string, compilerRun, error) {
	reset := func() error { return os.RemoveAll(filepath.Join(dir, "out")) }
	if err := reset(); err != nil {
		return "", compilerRun{}, err
	}
	want, err := runCompiler(dir, append(os.Environ(), "NODE_PATH="+filepath.Join(oracle, "out")), "node",
		append([]string{filepath.Join(oracle, "out", "CLI", "cli.js")}, flags...)...)
	if err != nil {
		return "", want, err
	}
	if resetErr := reset(); resetErr != nil {
		return "", want, resetErr
	}
	got, err := runCompiler(dir, os.Environ(), goblox, flags...)
	if err != nil {
		return "", want, err
	}
	return describeRunDifferences(want, got), want, nil
}

func describeRunDifferences(want compilerRun, got compilerRun) string {
	var differences strings.Builder
	if want.exitCode != got.exitCode {
		fmt.Fprintf(&differences, "exit code: rbxtsc %d, goblox %d\n", want.exitCode, got.exitCode)
	}
	if want.stdout != got.stdout {
		differences.WriteString(unifiedDiff("rbxtsc stdout", "goblox stdout", want.stdout, got.stdout))
	}
	if want.stderr != got.stderr {
		differences.WriteString(unifiedDiff("rbxtsc stderr", "goblox stderr", want.stderr, got.stderr))
	}
	return differences.String()
}

func unifiedDiff(wantHeader string, gotHeader string, want string, got string) string {
	lines := patience.Diff(stringutil.SplitLines(want), stringutil.SplitLines(got))
	return patience.UnifiedDiffTextWithOptions(lines, patience.UnifiedDiffOptions{
		Precontext:  3,
		Postcontext: 3,
		SrcHeader:   wantHeader,
		DstHeader:   gotHeader,
	})
}

// runDiagnosticFixtures compiles every fixture of <oracle>/tests/diagnostics.off (roblox-ts's
// tests/src/diagnostics), alone, with goblox and with the oracle's roblox-ts, and compares what they print.
// compile.test.ts compiles them one by one too, with the same flags.
func runDiagnosticFixtures(oracle string, filter *regexp.Regexp, verbose bool) int {
	fixturesDir := filepath.Join(oracle, "tests", "diagnostics.off")
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}
	var fixtures []string
	for _, entry := range entries {
		name := entry.Name()
		if (strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx")) && (filter == nil || filter.MatchString(name)) {
			fixtures = append(fixtures, name)
		}
	}
	sort.Strings(fixtures)

	tempRoot, err := os.MkdirTemp("", "goblox-fixtures-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}
	// expand 8.3 short names (C:/Users/PHASEN~1) so paths match the ones the program sees
	if long, evalSymlinksErr := filepath.EvalSymlinks(tempRoot); evalSymlinksErr == nil {
		tempRoot = long
	}
	defer os.RemoveAll(tempRoot)

	goblox, err := buildGoblox(tempRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}

	type outcome struct {
		differences string
		codes       []string
		err         error
	}
	outcomes := make([]outcome, len(fixtures))
	next := make(chan int)
	var wg sync.WaitGroup
	for worker := range max(1, runtime.NumCPU()-2) {
		dir := filepath.Join(tempRoot, fmt.Sprintf("project%d", worker))
		if err := prepareFixtureProject(oracle, dir); err != nil {
			fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
			return 2
		}
		wg.Go(func() {
			for i := range next {
				result := &outcomes[i]
				fixture := filepath.Join(dir, "src", "diagnostics", fixtures[i])
				if result.err = copyFile(filepath.Join(fixturesDir, fixtures[i]), fixture); result.err != nil {
					continue
				}
				var want compilerRun
				result.differences, want, result.err = compareCompilers(dir, oracle, goblox, "--allowCommentDirectives", "--optimizedLoops")
				result.codes = diagnosticCodePattern.FindAllString(want.stdout, -1)
				if removeErr := os.Remove(fixture); removeErr != nil && result.err == nil {
					result.err = removeErr
				}
			}
		})
	}
	for i := range fixtures {
		next <- i
	}
	close(next)
	wg.Wait()

	matched := 0
	for i, fixture := range fixtures {
		result := outcomes[i]
		switch {
		case result.err != nil:
			fmt.Printf("%-50s error: %v\n", fixture, result.err)
		case result.differences != "":
			fmt.Printf("%-50s differs\n", fixture)
			if verbose {
				fmt.Println(strings.TrimRight(result.differences, "\n"))
			}
		case len(result.codes) == 0:
			// a fixture must report a diagnostic; if roblox-ts reports none, this checks nothing
			fmt.Printf("%-50s no diagnostics\n", fixture)
		default:
			matched++
			fmt.Printf("%-50s match %s\n", fixture, strings.Join(result.codes, " "))
		}
	}
	fmt.Printf("\n%d of %d diagnostic fixtures match\n", matched, len(fixtures))
	if matched != len(fixtures) {
		return 1
	}
	return 0
}

var diagnosticCodePattern = regexp.MustCompile(`TS(?: roblox-ts|\d+)`)

// prepareFixtureProject copies the tests project to dir, without compiler output, specs and references:
// fixtures only import src/helpers.
func prepareFixtureProject(oracle string, dir string) error {
	testsDir := filepath.Join(oracle, "tests")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	err := copyTree(testsDir, dir, func(rel string) bool {
		return rel == "out" || rel == "diagnostics.off" || rel == "compiler"
	})
	if err != nil {
		return err
	}
	for _, name := range []string{filepath.Join("src", "tests"), filepath.Join("src", "main.server.ts")} {
		if removeAllErr := os.RemoveAll(filepath.Join(dir, name)); removeAllErr != nil {
			return removeAllErr
		}
	}
	if mkdirAllErr := os.MkdirAll(filepath.Join(dir, "src", "diagnostics"), 0o755); mkdirAllErr != nil {
		return mkdirAllErr
	}
	tsconfigPath := filepath.Join(dir, "tsconfig.json")
	tsconfig, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return err
	}
	return os.WriteFile(tsconfigPath, tsconfigReferencesPattern.ReplaceAll(tsconfig, nil), 0o644)
}

func copyFile(src string, dst string) error {
	contents, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, contents, 0o644)
}
