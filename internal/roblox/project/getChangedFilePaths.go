package project

// Port of Project/functions/getChangedFilePaths.ts, for a fresh builder (every file has changed), and of
// BuilderState.getReferencedFiles of TypeScript. goblox only needs the order: a file comes right before the
// files that import it, and roblox-ts transforms no further files after an error. A module declared in more
// than one file is referenced by every file, in program order, which doesn't change this order.

import (
	"context"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func getChangedFilePaths(ctx context.Context, program *compiler.Program) []tspath.Path {
	files := program.GetSourceFiles()
	c, done := program.GetTypeChecker(ctx)
	defer done()

	// referencedMap is sourceFile -> files that this file imports
	// but we need sourceFile -> files that import this file
	reversedReferencedMap := make(map[tspath.Path][]tspath.Path)
	seen := make(map[[2]tspath.Path]bool)
	for _, file := range files {
		addReferencedFile := func(referencedPath tspath.Path) {
			key := [2]tspath.Path{referencedPath, file.Path()}
			if !seen[key] {
				seen[key] = true
				reversedReferencedMap[referencedPath] = append(reversedReferencedMap[referencedPath], file.Path())
			}
		}
		addReferenceFromDeclarations := func(declarations []*ast.Node, withOwnFile bool) {
			for _, declaration := range declarations {
				if declarationFile := ast.GetSourceFileOfNode(declaration); declarationFile != nil && (withOwnFile || declarationFile != file) {
					addReferencedFile(declarationFile.Path())
				}
			}
		}
		addReferenceFromFileName := func(fileName string) {
			if redirect := program.GetParseFileRedirect(fileName); redirect != "" {
				fileName = redirect
			}
			addReferencedFile(tspath.ToPath(fileName, tspath.GetDirectoryPath(file.FileName()), program.UseCaseSensitiveFileNames()))
		}

		for _, importName := range file.Imports() {
			if symbol := c.GetSymbolAtLocation(importName); symbol != nil {
				addReferenceFromDeclarations(symbol.Declarations, true)
			}
		}
		for _, referencedFile := range file.ReferencedFiles {
			addReferenceFromFileName(referencedFile.FileName)
		}
		for _, directive := range program.GetResolvedTypeReferenceDirectives()[file.Path()] {
			if directive.ResolvedFileName != "" {
				addReferenceFromFileName(directive.ResolvedFileName)
			}
		}
		for _, moduleName := range file.ModuleAugmentations {
			if ast.IsStringLiteral(moduleName) {
				if symbol := c.GetSymbolAtLocation(moduleName); symbol != nil {
					addReferenceFromDeclarations(symbol.Declarations, false)
				}
			}
		}
	}

	assumeDirect := program.Options().AssumeChangesOnlyAffectDirectDependencies == core.TSTrue
	changedFilesSet := make(map[tspath.Path]bool, len(files))
	changedFiles := make([]tspath.Path, 0, len(files))
	add := func(filePath tspath.Path) {
		if !changedFilesSet[filePath] {
			changedFilesSet[filePath] = true
			changedFiles = append(changedFiles, filePath)
		}
	}
	var search func(filePath tspath.Path)
	search = func(filePath tspath.Path) {
		add(filePath)
		for _, refFilePath := range reversedReferencedMap[filePath] {
			if !changedFilesSet[refFilePath] {
				add(refFilePath)
				if !assumeDirect {
					search(refFilePath)
				}
			}
		}
	}
	for _, file := range files {
		search(file.Path())
	}
	return changedFiles
}

// fileReplayOrder is the indices of sourceFiles in the order of getChangedFilePaths.
func fileReplayOrder(ctx context.Context, program *compiler.Program, sourceFiles []*ast.SourceFile) []int {
	indices := make(map[tspath.Path]int, len(sourceFiles))
	for i, sourceFile := range sourceFiles {
		indices[sourceFile.Path()] = i
	}
	order := make([]int, 0, len(sourceFiles))
	for _, filePath := range getChangedFilePaths(ctx, program) {
		if i, ok := indices[filePath]; ok {
			order = append(order, i)
		}
	}
	return order
}
