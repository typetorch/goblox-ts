package main

// `goblox check`: load a tsconfig, build the program and checkers, report diagnostics and timings
// (the Phase 0 benchmark, see docs/goblox/BENCHMARKS.md).

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/diagnosticwriter"
	"github.com/microsoft/typescript-go/internal/roblox/project"
	"github.com/microsoft/typescript-go/internal/scanner"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func runCheck(args []string) int {
	flag := flag.NewFlagSet("goblox check", flag.ExitOnError)
	projectPath := flag.String("p", ".", "path to tsconfig.json or the folder containing it")
	checkers := flag.Int("checkers", 0, "number of type checkers (0 = typescript-go default of 4)")
	calls := flag.String("calls", "", "print GetTypeAtLocation for every call expression in this source file")
	noCheck := flag.Bool("nocheck", false, "skip the semantic check (program creation only)")
	all := flag.Bool("all", false, "report diagnostics of every file, including declaration files in node_modules (roblox-ts only reports the files it compiles)")
	flag.Parse(args)

	start := time.Now()
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cwd = tspath.NormalizePath(cwd)
	fs := project.DefaultFS()

	formatOpts := &diagnosticwriter.FormattingOptions{
		NewLine: "\n",
		ComparePathsOptions: tspath.ComparePathsOptions{
			CurrentDirectory:          cwd,
			UseCaseSensitiveFileNames: fs.UseCaseSensitiveFileNames(),
		},
	}
	report := func(diags []*ast.Diagnostic) {
		for _, d := range diags {
			diagnosticwriter.WriteFormatDiagnostic(os.Stdout, diagnosticwriter.WrapASTDiagnostic(d), formatOpts)
		}
	}

	programStart := time.Now()
	program, errs := project.LoadProgram(project.LoadOptions{
		Project:          *projectPath,
		CurrentDirectory: cwd,
		FS:               fs,
		Checkers:         *checkers,
	})
	if program == nil {
		report(errs)
		return 1
	}
	programTime := time.Since(programStart)

	ctx := context.Background()
	diags := project.GetConfigDiagnostics(ctx, program)

	bindStart := time.Now()
	program.GetBindDiagnostics(ctx, nil)
	bindTime := time.Since(bindStart)

	var checkTime time.Duration
	if !*noCheck {
		checkStart := time.Now()
		if *all {
			diags = append(diags, project.GetPreEmitDiagnostics(ctx, program, nil)...)
		} else {
			// Check every file in parallel first (as tsc does), then collect the diagnostics of the
			// files roblox-ts would compile.
			program.GetSemanticDiagnostics(ctx, nil)
			diags = append(diags, program.GetGlobalDiagnostics(ctx)...)
			for _, fileName := range program.CommandLine().FileNames() {
				file := program.GetSourceFile(fileName)
				if file == nil || file.IsDeclarationFile {
					continue
				}
				diags = append(diags, program.GetSemanticDiagnostics(ctx, file)...)
			}
			diags = project.FilterCompatDiagnostics(diags)
		}
		checkTime = time.Since(checkStart)
	}
	diags = compiler.SortAndDeduplicateDiagnostics(diags)
	report(diags)

	var callsTime time.Duration
	if *calls != "" {
		callsStart := time.Now()
		file := program.GetSourceFile(tspath.GetNormalizedAbsolutePath(*calls, cwd))
		if file == nil {
			fmt.Fprintf(os.Stderr, "goblox: %s is not in the program\n", *calls)
			return 1
		}
		c, done := program.GetTypeCheckerForFile(ctx, file)
		var visit ast.Visitor
		visit = func(node *ast.Node) bool {
			if ast.IsCallExpression(node) {
				line := scanner.GetECMALineOfPosition(file, scanner.SkipTrivia(file.Text(), node.Pos()))
				fmt.Printf("%d: %s\n", line+1, c.TypeToString(c.GetTypeAtLocation(node)))
			}
			node.ForEachChild(visit)
			return false
		}
		file.AsNode().ForEachChild(visit)
		done()
		callsTime = time.Since(callsStart)
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Fprintf(os.Stderr, "Files:          %d\n", len(program.GetSourceFiles()))
	fmt.Fprintf(os.Stderr, "Diagnostics:    %d\n", len(diags))
	fmt.Fprintf(os.Stderr, "Program time:   %v\n", programTime.Round(time.Millisecond))
	fmt.Fprintf(os.Stderr, "Bind time:      %v\n", bindTime.Round(time.Millisecond))
	fmt.Fprintf(os.Stderr, "Check time:     %v\n", checkTime.Round(time.Millisecond))
	if *calls != "" {
		fmt.Fprintf(os.Stderr, "Calls time:     %v\n", callsTime.Round(time.Millisecond))
	}
	fmt.Fprintf(os.Stderr, "Total time:     %v\n", time.Since(start).Round(time.Millisecond))
	fmt.Fprintf(os.Stderr, "Go heap (sys):  %d MB\n", mem.Sys>>20)
	if len(diags) > 0 {
		return 2
	}
	return 0
}
