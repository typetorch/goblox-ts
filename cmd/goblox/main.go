// Command goblox is the goblox-ts CLI: a port of the roblox-ts compiler (TypeScript -> Luau) on top of
// typescript-go. It is a drop-in for `rbxtsc` (see docs/goblox/CLI.md).
//
//	goblox [build] [flags]   build a project, like `rbxtsc build`
//	goblox watch [flags]     goblox build -w
//	goblox check [flags]     type check only, with timings (the Phase 0 benchmark)
//	goblox --version         goblox 0.1.0 (roblox-ts 3.0.0 compatible, typescript 7.x)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"time"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
	"github.com/microsoft/typescript-go/internal/diagnosticwriter"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/project"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// version is goblox's own version. Release builds set it with -ldflags "-X main.version=<npm version>".
var version = "0.1.0"

// ROBLOX_TS_VERSION is the roblox-ts release whose output goblox reproduces.
const ROBLOX_TS_VERSION = "3.0.0"

func versionString() string {
	return fmt.Sprintf("goblox %s (roblox-ts %s compatible, typescript %s)", version, ROBLOX_TS_VERSION, core.Version())
}

func main() {
	args := os.Args[1:]
	// yargs' .version(): -v/--version wins over everything else, with any command
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if arg == "-v" || arg == "--version" || arg == "-version" {
			fmt.Println(versionString())
			os.Exit(0)
		}
	}
	if len(args) > 0 {
		switch args[0] {
		case "check":
			os.Exit(runCheck(args[1:]))
		case "build":
			args = args[1:]
		case "watch":
			args = append([]string{"-watch"}, args[1:]...)
		}
	}
	os.Exit(runBuild(args))
}

// buildFlags are CLI/commands/build.ts's options. Go's flag package takes -name and --name alike.
type buildFlags struct {
	set         *flag.FlagSet
	projectPath *string
	checkers    *int
	timing      *bool
	cpuProfile  *string
	memProfile  *string
}

func newBuildFlags() *buildFlags {
	flags := flag.NewFlagSet("goblox", flag.ContinueOnError)
	f := &buildFlags{set: flags}
	f.projectPath = flags.String("project", ".", "project path")
	flags.StringVar(f.projectPath, "p", ".", "project path (alias of --project)")
	flags.Bool("watch", false, "enable watch mode")
	flags.Bool("w", false, "enable watch mode (alias of --watch)")
	flags.Bool("usePolling", false, "use polling for watch mode (goblox always polls)")
	flags.Bool("verbose", false, "enable verbose logs")
	flags.Bool("noInclude", false, "do not copy include files")
	flags.Bool("logTruthyChanges", false, "logs changes to truthiness evaluation from Lua truthiness rules")
	flags.Bool("logTruncatedPaths", false, "accepted for compatibility; goblox never truncates paths")
	flags.Bool("writeOnlyChanged", false, "only write output files whose contents changed (on by default in watch mode)")
	flags.Bool("writeTransformedFiles", false, "writes resulting TypeScript ASTs after transformers to out directory")
	flags.Bool("optimizedLoops", false, "optimize simple loops")
	flags.String("type", "", `override project type ("game", "model" or "package")`)
	includePath := flags.String("includePath", "", "folder to copy runtime files to")
	flags.StringVar(includePath, "i", "", "folder to copy runtime files to (alias of --includePath)")
	flags.String("rojo", "", "manually select Rojo project file")
	flags.Bool("allowCommentDirectives", false, "allow @ts-ignore and similar comment directives")
	flags.Bool("luau", false, "emit files with .luau extension")
	f.checkers = flags.Int("checkers", 0, "number of type checkers (0 = typescript-go default of 4)")
	f.timing = flags.Bool("timing", false, "print the build time")
	f.cpuProfile = flags.String("cpuprofile", "", "write a Go CPU profile to this file (goblox only)")
	f.memProfile = flags.String("memprofile", "", "write a Go heap profile to this file at exit (goblox only)")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "%s\n\nUsage: goblox [build|watch] [flags]\n       goblox check [flags]\n\nFlags:\n", versionString())
		flags.PrintDefaults()
	}
	return f
}

// runBuild is CLI/commands/build.ts.
func runBuild(args []string) int {
	f := newBuildFlags()
	flags := f.set
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		// yargs' .fail(): exit code 1, not Go's 2
		return 1
	}
	if flags.NArg() > 0 {
		// yargs' .strict()
		fmt.Fprintf(os.Stderr, "Unknown argument: %s\n", flags.Arg(0))
		return 1
	}

	if *f.cpuProfile != "" {
		out, err := os.Create(*f.cpuProfile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := pprof.StartCPUProfile(out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer func() {
			pprof.StopCPUProfile()
			out.Close()
		}()
	}
	if *f.memProfile != "" {
		defer func() {
			out, err := os.Create(*f.memProfile)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			defer out.Close()
			runtime.GC()
			if err := pprof.Lookup("allocs").WriteTo(out, 0); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}()
	}

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
		case "project", "p", "checkers", "timing", "cpuprofile", "memprofile", "logTruncatedPaths", "usePolling":
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

	report := newReporter(cwd, fs.UseCaseSensitiveFileNames())

	if t, ok := overrides["type"].(string); ok && t != "game" && t != "model" && t != "package" {
		report([]*ast.Diagnostic{diagnostics.CreateTextDiagnostic("Invalid values:\n  Argument: type, Given: \""+t+"\", Choices: \"game\", \"model\", \"package\"", tsdiagnostics.CategoryError)})
		return 1
	}

	opts := project.BuildOptions{
		Project:          *f.projectPath,
		CurrentDirectory: cwd,
		FS:               fs,
		Overrides:        overrides,
		Checkers:         *f.checkers,
	}
	if watch {
		return runWatch(opts, report)
	}

	_, errorCount, ok := buildOnce(opts, report)
	if *f.timing {
		fmt.Fprintf(os.Stderr, "Build time: %v\n", time.Since(start).Round(time.Millisecond))
	}
	if !ok || errorCount > 0 {
		return 1
	}
	return 0
}

// newReporter is ts.createDiagnosticReporter(ts.sys, true).
func newReporter(cwd string, useCaseSensitiveFileNames bool) func([]*ast.Diagnostic) {
	formatOpts := &diagnosticwriter.FormattingOptions{
		NewLine: "\n",
		ComparePathsOptions: tspath.ComparePathsOptions{
			CurrentDirectory:          cwd,
			UseCaseSensitiveFileNames: useCaseSensitiveFileNames,
		},
	}
	return func(diags []*ast.Diagnostic) {
		for _, d := range diags {
			diagnosticwriter.FormatDiagnosticWithColorAndContext(os.Stdout, diagnosticwriter.WrapASTDiagnostic(d), formatOpts)
			fmt.Fprint(os.Stdout, "\n")
		}
	}
}

// buildOnce runs one build and reports its diagnostics, and its panics on stderr. ok is false when the
// build failed before compiling (a CLIError, a ProjectError or an I/O error), which was reported as one
// error and is in err; errorCount counts error diagnostics.
func buildOnce(opts project.BuildOptions, report func([]*ast.Diagnostic)) (result *project.EmitResult, errorCount int, ok bool) {
	result, err := project.Build(context.Background(), opts)
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
		return nil, 1, false
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
	for _, d := range result.Diagnostics {
		if d.Category() == tsdiagnostics.CategoryError {
			errorCount++
		}
	}
	return result, errorCount, true
}

func nodePathResolve(cwd, p string) string {
	if p == "" {
		return p
	}
	return tspath.GetNormalizedAbsolutePath(p, cwd)
}
