package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/internal/roblox/project"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// rokitManifest pins what roblox-ts's foreman.toml pins. A rokit shim on PATH (rojo.exe, lune.exe) only runs
// inside a project with a manifest, so the temporary project gets one. Install the tools with:
//
//	rokit add --global rojo-rbx/rojo@7.4.4 && rokit add --global lune-org/lune@0.8.8
const rokitManifest = "[tools]\nrojo = \"rojo-rbx/rojo@7.4.4\"\nlune = \"lune-org/lune@0.8.8\"\n"

var luneSummaryPattern = regexp.MustCompile(`(?m)^(\d+) passed, (\d+) failed, (\d+) skipped\s*$`)

type luneOptions struct {
	oracle         string
	luneBin        string
	rojoBin        string
	compiler       string // "goblox", or "rbxtsc" for the baseline (the oracle's built roblox-ts)
	optimizedLoops bool
	verbose        bool
	keep           bool
}

// runLune is package.json's `npm test` minus jest: compile the tests project (test-compile), `rojo build`
// (test-rojo), `lune run runTestsWithLune.luau` (test-run). The project is a temporary copy of
// <oracle>/tests, so the oracle's out/ is never touched. Its referenced projects (projects/common and
// projects/shared) keep the oracle's outputs.
func runLune(opts luneOptions) int {
	tempDir, cleanup, err := copyTestsProject(opts.oracle, opts.keep)
	if err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}
	defer cleanup()

	if err := compileTestsProject(opts, tempDir); err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 1
	}

	// default.project.json maps out/diagnostics, and the fixtures that would fill it never emit
	if err := os.MkdirAll(filepath.Join(tempDir, "out", "diagnostics"), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}
	if err := os.WriteFile(filepath.Join(tempDir, "rokit.toml"), []byte(rokitManifest), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}

	rojo := exec.Command(opts.rojoBin, "build", ".", "-o", "test.rbxl")
	rojo.Dir = tempDir
	if output, err := rojo.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "goblox-fixtures: rojo build: %v\n%s", err, output)
		return 1
	}

	lune := exec.Command(opts.luneBin, "run", "./runTestsWithLune.luau", "./test.rbxl")
	lune.Dir = tempDir
	output, luneErr := lune.CombinedOutput()
	text := string(output)
	if opts.verbose || luneErr != nil {
		fmt.Print(text)
	}

	matches := luneSummaryPattern.FindStringSubmatch(text)
	if matches == nil {
		fmt.Fprintf(os.Stderr, "goblox-fixtures: no TestEZ summary in the Lune output (%v)\n", luneErr)
		return 1
	}
	failed, _ := strconv.Atoi(matches[2])
	fmt.Printf("%s: %s passed, %s failed, %s skipped\n", opts.compiler, matches[1], matches[2], matches[3])
	if luneErr != nil || failed > 0 {
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, "[-]") && !opts.verbose {
				fmt.Println(strings.TrimRight(line, "\r"))
			}
		}
		return 1
	}
	return 0
}

// copyTestsProject copies <oracle>/tests, without any compiler output, to a temporary directory.
func copyTestsProject(oracle string, keep bool) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "goblox-fixtures-")
	if err != nil {
		return "", nil, err
	}
	// expand 8.3 short names (C:/Users/PHASEN~1) so paths match the ones the program sees
	if long, err := filepath.EvalSymlinks(tempDir); err == nil {
		tempDir = long
	}
	cleanup := func() { os.RemoveAll(tempDir) }
	if keep {
		fmt.Fprintln(os.Stderr, "project copy:", tempDir)
		cleanup = func() {}
	}
	err = copyTree(filepath.Join(oracle, "tests"), tempDir, func(rel string) bool {
		return rel == "out" || rel == "diagnostics.off"
	})
	return tempDir, cleanup, err
}

// compileTestsProject builds the tests project, with the options compile.test.ts uses
// (`allowCommentDirectives`, and `optimizedLoops` when opts.optimizedLoops is set).
func compileTestsProject(opts luneOptions, dir string) error {
	if opts.compiler == "rbxtsc" {
		args := []string{filepath.Join(opts.oracle, "out", "CLI", "cli.js"), "--allowCommentDirectives"}
		if opts.optimizedLoops {
			args = append(args, "--optimizedLoops")
		}
		node := exec.Command("node", args...)
		node.Dir = dir
		node.Env = append(os.Environ(), "NODE_PATH="+filepath.Join(opts.oracle, "out"))
		if output, err := node.CombinedOutput(); err != nil {
			return fmt.Errorf("rbxtsc %s: %w\n%s", dir, err, output)
		}
		return nil
	}
	cwd := tspath.NormalizePath(dir)
	overrides := project.PartialProjectOptions{"allowCommentDirectives": true}
	if opts.optimizedLoops {
		overrides["optimizedLoops"] = true
	}
	result, err := project.Build(context.Background(), project.BuildOptions{
		Project:          cwd,
		CurrentDirectory: cwd,
		Overrides:        overrides,
	})
	if err != nil {
		return fmt.Errorf("goblox %s: %w", dir, err)
	}
	if result.EmitSkipped || len(result.Panics) > 0 {
		printDiagnostics(result, cwd)
		return fmt.Errorf("goblox %s: build failed with %d diagnostics, %d panics", dir, len(result.Diagnostics), len(result.Panics))
	}
	return nil
}
