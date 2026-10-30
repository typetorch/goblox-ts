package incremental

// The part of the incremental engine goblox's build uses (internal/roblox/project/incremental.go). tsc asks
// a Program for its semantic diagnostics; goblox computes its own (it checks a rebound proxy program and adds
// its own diagnostics), so it only asks which files have no cached result and reports back which are clean.

import (
	"context"
	"fmt"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/json"
	"github.com/microsoft/typescript-go/internal/tsoptions"
)

// SetSignatureSalt adds text to the d.ts signature of a file. Use it for what a file's dependents can see
// but its d.ts does not say (decorators). It must be set before the signatures are computed.
func (p *Program) SetSignatureSalt(salt func(file *ast.SourceFile) string) {
	p.signatureSalt = salt
}

// CollectAffectedFiles compares the d.ts signatures of the changed files and forgets the clean state of
// every file a change can reach. Afterwards IsClean is false for the files that need processing.
func (p *Program) CollectAffectedFiles(ctx context.Context) {
	collectAllAffectedFiles(ctx, p)
}

// IsClean reports whether the file has a cached result: it did not change, nothing it depends on changed
// shape, and it was marked clean by an earlier build.
func (p *Program) IsClean(file *ast.SourceFile) bool {
	_, ok := p.snapshot.semanticDiagnosticsPerFile.Load(file.Path())
	return ok
}

// MarkClean records that the file was processed without any diagnostic.
func (p *Program) MarkClean(file *ast.SourceFile) {
	p.snapshot.semanticDiagnosticsPerFile.Store(file.Path(), &DiagnosticsOrBuildInfoDiagnosticsWithFileName{})
}

// MarkNotClean makes the next build process the file again.
func (p *Program) MarkNotClean(file *ast.SourceFile) {
	p.snapshot.semanticDiagnosticsPerFile.Delete(file.Path())
}

// ComputeMissingSignatures computes the d.ts signature of every file that has dependents and still uses its
// version as its signature. Without it the first edit of a file looks like a change of its shape, because a
// new signature never equals a version. Dependents of nothing don't need one: no file is invalidated by them.
func (p *Program) ComputeMissingSignatures(ctx context.Context) {
	handler := affectedFilesHandler{ctx: ctx, program: p}
	wg := core.NewWorkGroup(p.program.SingleThreaded())
	for _, file := range p.program.GetSourceFiles() {
		if file.IsDeclarationFile {
			continue
		}
		info, ok := p.snapshot.fileInfos.Load(file.Path())
		if !ok || (info.signature != info.version && info.signature != "") {
			continue
		}
		hasDependents := false
		for range p.snapshot.referencedMap.getReferencedBy(file.Path()) {
			hasDependents = true
			break
		}
		if !hasDependents {
			continue
		}
		wg.Queue(func() {
			if signature := handler.computeDtsSignature(file); signature != "" {
				info.signature = signature
			}
		})
	}
	wg.RunAndWait()
}

// SameFiles reports whether program has exactly the files of this state, with the same text.
func (p *Program) SameFiles(program *compiler.Program) bool {
	files := program.GetSourceFiles()
	if p.snapshot.fileInfos.Size() != len(files) || p.snapshot.changedFilesSet.Size() != 0 {
		return false
	}
	same := make([]bool, len(files))
	wg := core.NewWorkGroup(program.SingleThreaded())
	for i, file := range files {
		wg.Queue(func() {
			info, ok := p.snapshot.fileInfos.Load(file.Path())
			same[i] = ok && info.version == p.snapshot.computeHash(file.Text())
		})
	}
	wg.RunAndWait()
	for _, ok := range same {
		if !ok {
			return false
		}
	}
	return true
}

// Detached is the state without the compiler program, so the old program's checkers can be collected.
func (p *Program) Detached() *Program {
	p.snapshot.allFilesExcludingDefaultLibraryFile = nil
	return &Program{snapshot: p.snapshot}
}

// MarshalState serializes the state in the .tsbuildinfo format. It needs the program, so call it before
// Detached. Paths are relative to buildInfoFileName.
func (p *Program) MarshalState(buildInfoFileName string) ([]byte, error) {
	buildInfo, err := snapshotToBuildInfo(p.snapshot, p.program, buildInfoFileName)
	if err != nil {
		return nil, err
	}
	return json.Marshal(buildInfo)
}

// UnmarshalState is the inverse of MarshalState. A state it cannot read is an error, never a partial state.
func UnmarshalState(data []byte, config *tsoptions.ParsedCommandLine, host compiler.CompilerHost) (program *Program, err error) {
	defer func() {
		if r := recover(); r != nil {
			program, err = nil, fmt.Errorf("corrupt incremental state: %v", r)
		}
	}()
	var buildInfo BuildInfo
	if err := json.Unmarshal(data, &buildInfo); err != nil {
		return nil, err
	}
	if !buildInfo.IsValidVersion() || !buildInfo.IsIncremental() {
		return nil, fmt.Errorf("unusable incremental state, version %q", buildInfo.Version)
	}
	return &Program{snapshot: buildInfoToSnapshot(&buildInfo, config, host)}, nil
}
