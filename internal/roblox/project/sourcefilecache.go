package project

// SourceFileCache keeps parsed and bound source files between the builds of a watch session, like
// typescript-go's own `tsc -w` (internal/execute/watcher.go watchCompilerHost): a file whose
// modification time and size did not change is not parsed or bound again. With verify (a build handed to
// the watcher, see BuildOptions.VerifySourceFiles) such a file is still read and its text compared, as an
// edit within the file system's time resolution that keeps the size (a git checkout) looks unchanged.

import (
	"sync"
	"time"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// SourceFileCache is safe for concurrent use. The zero value is ready to use.
type SourceFileCache struct {
	files sync.Map // tspath.Path -> *cachedSourceFile
}

type cachedSourceFile struct {
	file    *ast.SourceFile
	modTime time.Time
	size    int64
}

// cachingCompilerHost serves unchanged files from a SourceFileCache.
type cachingCompilerHost struct {
	compiler.CompilerHost
	cache *SourceFileCache
	// verify compares the text of a cached file with the file system's
	verify bool
}

func (h *cachingCompilerHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	info := h.CompilerHost.FS().Stat(opts.FileName)
	if value, ok := h.cache.files.Load(opts.Path); ok {
		cached := value.(*cachedSourceFile)
		if info != nil && info.ModTime().Equal(cached.modTime) && info.Size() == cached.size && cached.file.ParseOptions() == opts {
			if !h.verify {
				return cached.file
			}
			// the text compilerHost.GetSourceFile would parse
			if text, ok := h.CompilerHost.FS().ReadFile(opts.FileName); ok && text == cached.file.Text() {
				return cached.file
			}
		}
	}
	file := h.CompilerHost.GetSourceFile(opts)
	if file != nil && info != nil {
		h.cache.files.Store(opts.Path, &cachedSourceFile{file: file, modTime: info.ModTime(), size: info.Size()})
	} else {
		h.cache.files.Delete(opts.Path)
	}
	return file
}

// prune drops the files that are not in program, so deleted files don't stay in memory.
func (c *SourceFileCache) prune(program *compiler.Program) {
	files := program.FilesByPath()
	c.files.Range(func(key, _ any) bool {
		if _, ok := files[key.(tspath.Path)]; !ok {
			c.files.Delete(key)
		}
		return true
	})
}
