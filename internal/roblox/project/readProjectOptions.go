package project

// Port of Project/functions/readProjectOptions.ts, Project/functions/getExtendedConfigPaths.ts and
// Shared/classes/LogService.ts.

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/collections"
	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/roblox/transform"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// LogService is Shared/classes/LogService.ts.
var LogService = &logService{out: os.Stdout}

type logService struct {
	mu      sync.Mutex
	out     io.Writer
	Verbose bool
	partial bool
}

func (l *logService) Write(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.write(message)
}

func (l *logService) write(message string) {
	l.partial = !strings.HasSuffix(message, "\n")
	fmt.Fprint(l.out, message)
}

func (l *logService) WriteLine(messages ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.partial {
		l.write("\n")
	}
	for _, message := range messages {
		l.write(message + "\n")
	}
}

func (l *logService) WriteLineIfVerbose(messages ...string) {
	if l.Verbose {
		l.WriteLine(messages...)
	}
}

func (l *logService) Warn(message string) {
	l.WriteLine(diagnostics.Yellow("Compiler Warning:") + " " + message)
}

// PartialProjectOptions is Partial<ProjectOptions>: the options that were set, by their roblox-ts name.
type PartialProjectOptions map[string]any

var booleanOptions = map[string]bool{
	"noInclude":              true,
	"luau":                   true,
	"optimizedLoops":         true,
	"allowCommentDirectives": true,
	"logTruthyChanges":       true,
	"writeOnlyChanged":       true,
	"writeTransformedFiles":  true,
}

var pathOptions = map[string]bool{"rojo": true, "includePath": true}

// ApplyProjectOptions is `{ ...options, ...partial }`.
func ApplyProjectOptions(options *transform.ProjectOptions, partial PartialProjectOptions) {
	for key, value := range partial {
		switch key {
		case "includePath":
			options.IncludePath = value.(string)
		case "rojo":
			options.Rojo = value.(string)
		case "type":
			options.Type = transform.ProjectType(value.(string))
		case "logTruthyChanges":
			options.LogTruthyChanges = value.(bool)
		case "noInclude":
			options.NoInclude = value.(bool)
		case "noCompilerHeader":
			options.NoCompilerHeader = value.(bool)
		case "usePolling":
			options.UsePolling = value.(bool)
		case "verbose":
			options.Verbose = value.(bool)
		case "watch":
			options.Watch = value.(bool)
		case "writeOnlyChanged":
			options.WriteOnlyChanged = value.(bool)
		case "writeTransformedFiles":
			options.WriteTransformedFiles = value.(bool)
		case "optimizedLoops":
			options.OptimizedLoops = value.(bool)
		case "allowCommentDirectives":
			options.AllowCommentDirectives = value.(bool)
		case "luau":
			options.Luau = value.(bool)
		}
	}
}

func readOwnOptions(configPath string, value any) (PartialProjectOptions, error) {
	options := PartialProjectOptions{}
	if value == nil {
		return options, nil
	}

	invalid := func(message string) error {
		return &diagnostics.DiagnosticError{Diagnostics: []*ast.Diagnostic{
			diagnostics.CreateTextDiagnostic(`Invalid "rbxts" configuration in "`+configPath+`": `+message, tsdiagnostics.CategoryError),
		}}
	}
	object, ok := value.(*collections.OrderedMap[string, any])
	if !ok {
		return nil, invalid("expected an object")
	}

	for key, setting := range object.Entries() {
		if booleanOptions[key] {
			if _, ok := setting.(bool); !ok {
				return nil, invalid(`"` + key + `" must be a boolean`)
			}
			options[key] = setting
		} else if pathOptions[key] {
			s, ok := setting.(string)
			if !ok {
				return nil, invalid(`"` + key + `" must be a string`)
			}
			if s != "" {
				s = nodepath.Resolve(nodepath.Dirname(configPath), s)
			}
			options[key] = s
		} else if key == "type" {
			s, _ := setting.(string)
			if s != string(transform.ProjectTypeGame) && s != string(transform.ProjectTypeModel) && s != string(transform.ProjectTypePackage) {
				return nil, invalid(`"type" must be "game", "model", or "package"`)
			}
			options[key] = s
		} else {
			LogService.Warn(`Ignoring unsupported "rbxts" option "` + key + `" in "` + configPath + `".`)
		}
	}
	return options, nil
}

// readProjectOptions reads the "rbxts" options of configPath and of the configs it extends.
func readProjectOptions(host *parseHost, configPath string, cache map[string]PartialProjectOptions) (PartialProjectOptions, error) {
	key := tspath.GetCanonicalFileName(nodepath.Normalize(configPath), host.fs.UseCaseSensitiveFileNames())
	if cached, ok := cache[key]; ok {
		return cached, nil
	}

	text, ok := host.fs.ReadFile(configPath)
	if !ok {
		return nil, &diagnostics.DiagnosticError{Diagnostics: []*ast.Diagnostic{
			ast.NewCompilerDiagnostic(tsdiagnostics.Cannot_read_file_0, configPath),
		}}
	}
	raw, errs := tsoptions.ParseConfigFileTextToJson(configPath, tspath.ToPath(configPath, host.cwd, host.fs.UseCaseSensitiveFileNames()), text)
	if len(errs) > 0 {
		return nil, &diagnostics.DiagnosticError{Diagnostics: errs}
	}
	config, _ := raw.(*collections.OrderedMap[string, any])

	options := PartialProjectOptions{}
	var extends []string
	var rbxts any
	if config != nil {
		switch e := config.GetOrZero("extends").(type) {
		case string:
			extends = []string{e}
		case []any:
			for _, v := range e {
				if s, ok := v.(string); ok {
					extends = append(extends, s)
				}
			}
		}
		rbxts = config.GetOrZero("rbxts")
	}
	for _, value := range extends {
		extendedPath := resolveExtendedConfig(host, configPath, value)
		if extendedPath == "" {
			return nil, &diagnostics.DiagnosticError{Diagnostics: []*ast.Diagnostic{
				ast.NewCompilerDiagnostic(tsdiagnostics.File_0_not_found, value),
			}}
		}
		extended, err := readProjectOptions(host, nodepath.Normalize(extendedPath), cache)
		if err != nil {
			return nil, err
		}
		for k, v := range extended {
			options[k] = v
		}
	}
	own, err := readOwnOptions(configPath, rbxts)
	if err != nil {
		return nil, err
	}
	for k, v := range own {
		options[k] = v
	}
	cache[key] = options
	return options, nil
}
