// Command goblox is the goblox-ts CLI: a port of the roblox-ts compiler (TypeScript -> Luau) on top of
// typescript-go.
//
//	goblox [build] [flags]   build a project, like `rbxtsc build`
//	goblox check [flags]     type check only, with timings (the Phase 0 benchmark)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/microsoft/typescript-go/internal/ast"
	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
	"github.com/microsoft/typescript-go/internal/diagnosticwriter"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/project"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "check" {
		os.Exit(runCheck(args[1:]))
	}
	if len(args) > 0 && args[0] == "build" {
		args = args[1:]
	}
	os.Exit(runBuild(args))
}

// runBuild is CLI/commands/build.ts.
func runBuild(args []string) int {
	flags := flag.NewFlagSet("goblox build", flag.ExitOnError)
	projectPath := flags.String("project", ".", "project path")
	flags.StringVar(projectPath, "p", ".", "project path (alias of -project)")
	flags.Bool("watch", false, "enable watch mode (not supported yet)")
	flags.Bool("w", false, "enable watch mode (not supported yet)")
	flags.Bool("usePolling", false, "use polling for watch mode")
	flags.Bool("verbose", false, "enable verbose logs")
	flags.Bool("noInclude", false, "do not copy include files")
	flags.Bool("logTruthyChanges", false, "logs changes to truthiness evaluation from Lua truthiness rules")
	flags.Bool("writeOnlyChanged", false, "only write output files whose contents changed")
	flags.Bool("writeTransformedFiles", false, "writes resulting TypeScript ASTs after transformers to out directory")
	flags.Bool("optimizedLoops", true, "optimize simple loops")
	flags.String("type", "", `override project type ("game", "model" or "package")`)
	includePath := flags.String("includePath", "", "folder to copy runtime files to")
	flags.StringVar(includePath, "i", "", "folder to copy runtime files to (alias of -includePath)")
	flags.String("rojo", "", "manually select Rojo project file")
	flags.Bool("allowCommentDirectives", false, "allow @ts-ignore and similar comment directives")
	flags.Bool("luau", true, "emit files with .luau extension")
	checkers := flags.Int("checkers", 0, "number of type checkers (0 = typescript-go default of 4)")
	timing := flags.Bool("timing", false, "print the build time")
	_ = flags.Parse(args)

	start := time.Now()
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cwd = tspath.NormalizePath(cwd)
	fs := project.DefaultFS()

	// DO NOT PROVIDE DEFAULTS, USE DEFAULT_PROJECT_OPTIONS: only the flags that were given override
	overrides := project.PartialProjectOptions{}
	watch := false
	flags.Visit(func(f *flag.Flag) {
		getter := f.Value.(flag.Getter)
		switch f.Name {
		case "project", "p", "checkers", "timing":
		case "w", "watch":
			watch = getter.Get().(bool)
		case "i":
			overrides["includePath"] = nodePathResolve(cwd, getter.Get().(string))
		case "includePath", "rojo":
			overrides[f.Name] = nodePathResolve(cwd, getter.Get().(string))
		default:
			overrides[f.Name] = getter.Get()
		}
	})

	formatOpts := &diagnosticwriter.FormattingOptions{
		NewLine: "\n",
		ComparePathsOptions: tspath.ComparePathsOptions{
			CurrentDirectory:          cwd,
			UseCaseSensitiveFileNames: fs.UseCaseSensitiveFileNames(),
		},
	}
	report := func(diags []*ast.Diagnostic) {
		for _, d := range diags {
			diagnosticwriter.FormatDiagnosticWithColorAndContext(os.Stdout, diagnosticwriter.WrapASTDiagnostic(d), formatOpts)
			fmt.Fprint(os.Stdout, "\n")
		}
	}

	if t, ok := overrides["type"].(string); ok && t != "game" && t != "model" && t != "package" {
		report([]*ast.Diagnostic{diagnostics.CreateTextDiagnostic("Invalid values:\n  Argument: type, Given: \""+t+"\", Choices: \"game\", \"model\", \"package\"", tsdiagnostics.CategoryError)})
		return 1
	}
	if watch {
		report([]*ast.Diagnostic{diagnostics.CreateTextDiagnostic("goblox does not support watch mode yet", tsdiagnostics.CategoryError)})
		return 1
	}

	result, err := project.Build(context.Background(), project.BuildOptions{
		Project:          *projectPath,
		CurrentDirectory: cwd,
		FS:               fs,
		Overrides:        overrides,
		Checkers:         *checkers,
	})
	if err != nil {
		var projectError *diagnostics.ProjectError
		var cliError *project.CLIError
		switch {
		case errors.As(err, &projectError):
			report([]*ast.Diagnostic{diagnostics.CreateTextDiagnostic(projectError.Message, tsdiagnostics.CategoryError)})
		case errors.As(err, &cliError):
			report([]*ast.Diagnostic{diagnostics.CreateTextDiagnostic(cliError.Message, tsdiagnostics.CategoryError)})
		default:
			fmt.Fprintln(os.Stderr, err)
		}
		return 1
	}

	report(result.Diagnostics)
	if len(result.Panics) > 0 {
		files := make([]string, 0, len(result.Panics))
		for file := range result.Panics {
			files = append(files, file)
		}
		sort.Strings(files)
		for _, file := range files {
			fmt.Fprintf(os.Stderr, "goblox: %s: %s\n", file, strings.TrimSpace(result.Panics[file]))
		}
	}
	if *timing {
		fmt.Fprintf(os.Stderr, "Build time: %v\n", time.Since(start).Round(time.Millisecond))
	}
	for _, d := range result.Diagnostics {
		if d.Category() == tsdiagnostics.CategoryError {
			return 1
		}
	}
	return 0
}

func nodePathResolve(cwd, p string) string {
	if p == "" {
		return p
	}
	return tspath.GetNormalizedAbsolutePath(p, cwd)
}
