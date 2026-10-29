package rojo

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
)

// Port of @roblox-ts/path-translator 1.1.1.

const (
	tsExt          = ".ts"
	tsxExt         = ".tsx"
	dExt           = ".d"
	dtsExt         = dExt + tsExt
	transformedExt = ".transformed"
	indexName      = "index"
)

type pathInfo struct {
	dirName  string
	fileName string
	exts     []string
}

func pathInfoFrom(filePath string) *pathInfo {
	dirName := nodepath.Dirname(filePath)
	rest := ""
	if start := len(dirName) + len(nodepath.Sep); start <= len(filePath) {
		rest = filePath[start:]
	}
	parts := strings.Split(rest, ".")
	exts := make([]string, 0, len(parts)-1)
	for _, v := range parts[1:] {
		exts = append(exts, "."+v)
	}
	return &pathInfo{dirName: dirName, fileName: parts[0], exts: exts}
}

func (p *pathInfo) extsPeek(depth int) string {
	i := len(p.exts) - (depth + 1)
	if i < 0 || i >= len(p.exts) {
		return ""
	}
	return p.exts[i]
}

func (p *pathInfo) pop() string {
	if len(p.exts) == 0 {
		return ""
	}
	last := p.exts[len(p.exts)-1]
	p.exts = p.exts[:len(p.exts)-1]
	return last
}

func (p *pathInfo) push(ext string) { p.exts = append(p.exts, ext) }

func (p *pathInfo) join() string {
	return nodepath.Join(p.dirName, p.fileName+strings.Join(p.exts, ""))
}

// PathTranslator maps between source (rootDir) and output (outDir) paths.
type PathTranslator struct {
	RootDir             string
	OutDir              string
	BuildInfoOutputPath string
	Declaration         bool
	UseLuauExtension    bool
}

// NewPathTranslator is `new PathTranslator(...)`.
func NewPathTranslator(rootDir, outDir, buildInfoOutputPath string, declaration bool, useLuauExtension bool) *PathTranslator {
	return &PathTranslator{
		RootDir:             rootDir,
		OutDir:              outDir,
		BuildInfoOutputPath: buildInfoOutputPath,
		Declaration:         declaration,
		UseLuauExtension:    useLuauExtension,
	}
}

func (t *PathTranslator) getLuauExt() string {
	if t.UseLuauExtension {
		return luauExt
	}
	return luaExt
}

func makeRelativeFactory(from, to string) func(*pathInfo) string {
	return func(p *pathInfo) string { return nodepath.Join(to, nodepath.Relative(from, p.join())) }
}

func isTsExt(ext string) bool { return ext == tsExt || ext == tsxExt }

// GetOutputPath maps an input path to an output path:
// `.ts(x)` && !`.d.ts(x)` -> `.lua(u)`, `index` -> `init`, `src/*` -> `out/*`.
func (t *PathTranslator) GetOutputPath(filePath string) string {
	makeRelative := makeRelativeFactory(t.RootDir, t.OutDir)
	p := pathInfoFrom(filePath)

	if isTsExt(p.extsPeek(0)) && p.extsPeek(1) != dExt {
		p.pop() // pop .ts(x)

		// index -> init
		if p.fileName == indexName {
			p.fileName = initName
		}

		p.push(t.getLuauExt())
	}

	return makeRelative(p)
}

// GetOutputDeclarationPath maps an input path to an output .d.ts path.
func (t *PathTranslator) GetOutputDeclarationPath(filePath string) string {
	makeRelative := makeRelativeFactory(t.RootDir, t.OutDir)
	p := pathInfoFrom(filePath)

	if isTsExt(p.extsPeek(0)) && p.extsPeek(1) != dExt {
		p.pop() // pop .ts(x)
		p.push(dtsExt)
	}

	return makeRelative(p)
}

// GetOutputTransformedPath maps an input path to an output .transformed.ts(x) path.
func (t *PathTranslator) GetOutputTransformedPath(filePath string) string {
	makeRelative := makeRelativeFactory(t.RootDir, t.OutDir)
	p := pathInfoFrom(filePath)

	at := len(p.exts) - 1
	if p.extsPeek(1) == dExt {
		// Transformers currently never get a chance to transform .d.ts files
		// But case is covered anyways
		at = len(p.exts) - 2
	}
	at = max(at, 0)
	p.exts = append(p.exts[:at], append([]string{transformedExt}, p.exts[at:]...)...)

	return makeRelative(p)
}

// GetInputPaths maps an output path to the possible input paths it came from.
func (t *PathTranslator) GetInputPaths(filePath string) []string {
	makeRelative := makeRelativeFactory(t.OutDir, t.RootDir)
	var possiblePaths []string
	p := pathInfoFrom(filePath)

	// index.*.lua(u) cannot come from a .ts file
	if p.extsPeek(0) == t.getLuauExt() && p.fileName != indexName {
		p.pop() // pop .lua(u)

		// ts
		p.push(tsExt)
		possiblePaths = append(possiblePaths, makeRelative(p))
		p.pop()

		// tsx
		p.push(tsxExt)
		possiblePaths = append(possiblePaths, makeRelative(p))
		p.pop()

		// init -> index
		if p.fileName == initName {
			originalFileName := p.fileName
			p.fileName = indexName

			// index.*.ts
			p.push(tsExt)
			possiblePaths = append(possiblePaths, makeRelative(p))
			p.pop()

			// index.*.tsx
			p.push(tsxExt)
			possiblePaths = append(possiblePaths, makeRelative(p))
			p.pop()

			p.fileName = originalFileName
		}

		p.push(t.getLuauExt())
	}

	if t.Declaration {
		if isTsExt(p.extsPeek(0)) && p.extsPeek(1) == dExt {
			tsExtension := p.pop() // pop .ts(x)
			p.pop()                // pop .d

			// .ts
			p.push(tsExt)
			possiblePaths = append(possiblePaths, makeRelative(p))
			p.pop()

			// .tsx
			p.push(tsxExt)
			possiblePaths = append(possiblePaths, makeRelative(p))
			p.pop()

			p.push(dExt)
			p.push(tsExtension)
		}
	}

	possiblePaths = append(possiblePaths, makeRelative(p))
	return possiblePaths
}

// GetImportPath maps a src path to an import path. Import paths are passed to RojoResolver and used
// virtually to resolve RbxPaths, so the import path may not actually exist:
// `.d.ts(x)` -> `.ts(x)` -> `.lua(u)`, `index` -> `init`.
func (t *PathTranslator) GetImportPath(filePath string, isNodeModule bool) string {
	makeRelative := makeRelativeFactory(t.RootDir, t.OutDir)
	p := pathInfoFrom(filePath)

	if isTsExt(p.extsPeek(0)) {
		p.pop() // pop .ts(x)
		if p.extsPeek(0) == dExt {
			p.pop() // pop .d
		}

		// index -> init
		if p.fileName == indexName {
			p.fileName = initName
		}

		p.push(t.getLuauExt()) // push .lua(u)
	}

	// inside of node_modules, we assume compiled file is sibling of filePath
	// outside, we check relative to outDir
	if isNodeModule {
		return p.join()
	}
	return makeRelative(p)
}
