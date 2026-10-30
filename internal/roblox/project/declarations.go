package project

// Port of the declaration steps of Project/functions/compileFiles.ts, of
// Project/functions/transformDeclarationFile.ts and of Project/transformers/builtin. typescript-go's Emit has
// no afterDeclarations hook, so the transformers edit the printed text, which only changes module specifiers
// and type references. Mappings of a .d.ts.map after a changed specifier on the same line are off by the
// difference in length.

import (
	"context"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/printer"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/scanner"
	"github.com/microsoft/typescript-go/internal/stringutil"
	"github.com/microsoft/typescript-go/internal/tspath"
)

type declarationWrite struct {
	fileName string
	text     string
}

// emitDeclarations is proxyProgram.emit(sourceFile, writer, undefined, true, { afterDeclarations }) for every
// file. roblox-ts holds the writes back, as declaration errors must not leave Luau and declaration outputs from
// different builds.
func emitDeclarations(ctx context.Context, program *compiler.Program, proxyProgram *compiler.Program, sourceFiles []*ast.SourceFile) ([]declarationWrite, []*ast.Diagnostic) {
	var mu sync.Mutex
	var writes []declarationWrite
	result := proxyProgram.Emit(ctx, compiler.EmitOptions{
		TargetSourceFiles: sourceFiles,
		EmitOnly:          compiler.EmitOnlyDts,
		WriteFile: func(fileName string, text string, data *compiler.WriteFileData) error {
			if tspath.IsDeclarationFileName(fileName) {
				text = transformDeclarationOutput(program, fileName, data.SourceFile.FileName(), text)
			}
			mu.Lock()
			defer mu.Unlock()
			writes = append(writes, declarationWrite{fileName, text})
			return nil
		},
	})
	slices.SortFunc(writes, func(a, b declarationWrite) int { return strings.Compare(a.fileName, b.fileName) })
	return writes, result.Diagnostics
}

// textEdit replaces the text between pos and end.
type textEdit struct {
	pos  int
	end  int
	text string
}

func applyTextEdits(text string, edits []textEdit) string {
	slices.SortFunc(edits, func(a, b textEdit) int { return a.pos - b.pos })
	var result strings.Builder
	last := 0
	for _, edit := range edits {
		result.WriteString(text[last:edit.pos])
		result.WriteString(edit.text)
		last = edit.end
	}
	result.WriteString(text[last:])
	return result.String()
}

func parseDeclarationText(fileName string, text string) *ast.SourceFile {
	return parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: fileName, Path: tspath.Path(fileName)}, text, core.ScriptKindTS)
}

// transformDeclarationOutput runs the afterDeclarations transformers over what the emit printed for sourceFileName:
// transformTypeReferenceDirectives, then transformPaths.
func transformDeclarationOutput(program *compiler.Program, fileName string, sourceFileName string, text string) string {
	file := parseDeclarationText(fileName, text)
	edits := transformPaths(program, file, sourceFileName)
	for _, directive := range file.TypeReferenceDirectives {
		if directive.FileName == "types" {
			edits = append(edits, textEdit{directive.Pos(), directive.End(), RBXTS_SCOPE + "/types"})
		}
	}
	return applyTextEdits(text, edits)
}

// transformDeclarationFile is transformDeclarationFile.ts: a declaration asset is printed again, not copied.
func transformDeclarationFile(program *compiler.Program, fileName string) (string, error) {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	emitContext, done := printer.GetEmitContext()
	defer done()
	source := parseDeclarationText(fileName, stringutil.RemoveByteOrderMark(string(contents)))
	p := printer.NewPrinter(printer.PrinterOptions{NewLine: core.NewLineKindLF}, printer.PrintHandlers{}, emitContext)
	text := p.EmitSourceFile(source)
	return applyTextEdits(text, transformPaths(program, parseDeclarationText(fileName, text), fileName)), nil
}

// transformPaths is the visitor of transformPaths.ts, with a module specifier's edit for each updated node.
// sourceFileName is the file that the module specifiers are resolved from.
func transformPaths(program *compiler.Program, file *ast.SourceFile, sourceFileName string) []textEdit {
	compilerOptions := program.Options()
	if compilerOptions.BaseUrl == "" && compilerOptions.Paths == nil {
		return nil
	}

	implicitExtensions := []string{tspath.ExtensionTs, tspath.ExtensionDts}
	allowJs := compilerOptions.GetAllowJS()
	allowJsx := compilerOptions.Jsx != core.JsxEmitNone
	if allowJs {
		implicitExtensions = append(implicitExtensions, tspath.ExtensionJs)
	}
	if allowJsx {
		implicitExtensions = append(implicitExtensions, tspath.ExtensionTsx)
	}
	if allowJs && allowJsx {
		implicitExtensions = append(implicitExtensions, tspath.ExtensionJsx)
	}
	if compilerOptions.GetResolveJsonModule() {
		implicitExtensions = append(implicitExtensions, tspath.ExtensionJson)
	}

	var edits []textEdit
	update := func(literal *ast.Node) {
		moduleName := literal.Text()
		resolved := program.ResolveModuleName(moduleName, sourceFileName, core.ResolutionModeNone)
		if resolved == nil || !resolved.IsResolved() || resolved.IsExternalLibraryImport {
			return
		}
		fileDir := nodepath.Dirname(sourceFileName)
		modulePath := nodepath.Dirname(resolved.ResolvedFileName)
		p := nodepath.Join(nodepath.Relative(fileDir, modulePath), nodepath.Basename(resolved.ResolvedFileName))
		if slices.Contains(implicitExtensions, resolved.Extension) {
			p = strings.TrimSuffix(p, resolved.Extension)
		}
		if p == "" {
			return
		}
		if p[0] != '.' {
			p = "./" + p
		}
		edits = append(edits, textEdit{scanner.SkipTrivia(file.Text(), literal.Pos()), literal.End(), `"` + p + `"`})
	}

	var visit func(node *ast.Node) bool
	visit = func(node *ast.Node) bool {
		switch node.Kind {
		case ast.KindExternalModuleReference:
			if expression := node.AsExternalModuleReference().Expression; ast.IsStringLiteral(expression) {
				update(expression)
				return false
			}
		case ast.KindImportDeclaration, ast.KindJSImportDeclaration, ast.KindExportDeclaration:
			if specifier := node.ModuleSpecifier(); specifier != nil && ast.IsStringLiteral(specifier) {
				update(specifier)
				return false
			}
		case ast.KindImportType:
			argument := node.AsImportTypeNode().Argument
			if ast.IsLiteralTypeNode(argument) {
				if literal := argument.AsLiteralTypeNode().Literal; ast.IsStringLiteral(literal) && literal.Text() != "" {
					update(literal)
				}
			}
			return false
		}
		node.ForEachChild(visit)
		return false
	}
	file.AsNode().ForEachChild(visit)
	return edits
}
