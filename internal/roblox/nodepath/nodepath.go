// Package nodepath implements the parts of Node.js's `path` module (POSIX flavour) that roblox-ts,
// rojo-resolver and path-translator use, with Node's exact semantics: Go's path/filepath differs on
// empty results ("" vs "."), trailing separators, dot files and relative paths.
//
// Paths always use "/". typescript-go normalizes Windows paths to "C:/a/b", so besides "/" a drive
// root ("C:/") is an absolute root too, as in path.win32. Relative compares the segments of drive
// paths case-insensitively, like path.win32.relative.
package nodepath

import (
	"os"
	"strings"
)

// Sep is path.sep.
const Sep = "/"

// rootOf returns the root of an absolute path ("/" or a drive root like "C:/"), "" for a relative path.
func rootOf(p string) string {
	if strings.HasPrefix(p, "/") {
		return "/"
	}
	if len(p) >= 3 && p[1] == ':' && (p[2] == '/' || p[2] == '\\') && isDriveLetter(p[0]) {
		return p[:2] + "/"
	}
	if len(p) == 2 && p[1] == ':' && isDriveLetter(p[0]) {
		return p + "/"
	}
	return ""
}

func isDriveLetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// Normalize is path.normalize.
func Normalize(p string) string {
	if p == "" {
		return "."
	}
	root := rootOf(p)
	isAbsolute := root != ""
	trailingSeparator := strings.HasSuffix(p, "/")
	p = normalizeString(p[min(len(root), len(p)):], !isAbsolute)
	if p == "" {
		if isAbsolute {
			return root
		}
		if trailingSeparator {
			return "./"
		}
		return "."
	}
	if trailingSeparator {
		p += "/"
	}
	return root + p
}

// normalizeString resolves . and .. segments, like Node's internal normalizeString.
func normalizeString(p string, allowAboveRoot bool) string {
	var out []string
	for _, segment := range strings.Split(p, "/") {
		switch segment {
		case "", ".":
			continue
		case "..":
			if len(out) > 0 && out[len(out)-1] != ".." {
				out = out[:len(out)-1]
			} else if allowAboveRoot {
				out = append(out, "..")
			}
		default:
			out = append(out, segment)
		}
	}
	return strings.Join(out, "/")
}

// Join is path.join.
func Join(paths ...string) string {
	var nonEmpty []string
	for _, p := range paths {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	if len(nonEmpty) == 0 {
		return "."
	}
	return Normalize(strings.Join(nonEmpty, "/"))
}

// Cwd is used by Resolve for relative paths. It defaults to the process's working directory.
var Cwd = func() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "/"
	}
	return strings.ReplaceAll(cwd, "\\", "/")
}

// Resolve is path.resolve.
func Resolve(paths ...string) string {
	resolved := ""
	root := ""
	for i := len(paths) - 1; i >= -1 && root == ""; i-- {
		var p string
		if i >= 0 {
			p = paths[i]
		} else {
			p = Cwd()
		}
		if p == "" {
			continue
		}
		root = rootOf(p)
		resolved = p[min(len(root), len(p)):] + "/" + resolved
	}
	resolved = normalizeString(resolved, root == "")
	if root != "" {
		return root + resolved
	}
	if resolved == "" {
		return "."
	}
	return resolved
}

// IsAbsolute is path.isAbsolute.
func IsAbsolute(p string) bool { return rootOf(p) != "" }

// Relative is path.relative.
func Relative(from, to string) string {
	if from == to {
		return ""
	}
	from = Resolve(from)
	to = Resolve(to)
	if from == to {
		return ""
	}
	fromRoot, toRoot := rootOf(from), rootOf(to)
	equal := func(a, b string) bool { return a == b }
	if fromRoot != "/" || toRoot != "/" {
		if !strings.EqualFold(fromRoot, toRoot) {
			return to
		}
		equal = strings.EqualFold
	}
	fromParts := splitNonEmpty(from[len(fromRoot):])
	toParts := splitNonEmpty(to[len(toRoot):])
	i := 0
	for i < len(fromParts) && i < len(toParts) && equal(fromParts[i], toParts[i]) {
		i++
	}
	var out []string
	for range fromParts[i:] {
		out = append(out, "..")
	}
	out = append(out, toParts[i:]...)
	return strings.Join(out, "/")
}

func splitNonEmpty(p string) []string {
	var parts []string
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return parts
}

// Dirname is path.dirname.
func Dirname(p string) string {
	if p == "" {
		return "."
	}
	root := rootOf(p)
	if len(root) > 1 {
		if len(p) <= len(root) {
			return root
		}
		dir := Dirname("/" + p[len(root):])
		if dir == "/" {
			return root
		}
		return root + dir[1:]
	}
	hasRoot := p[0] == '/'
	end := -1
	matchedSlash := true
	for i := len(p) - 1; i >= 1; i-- {
		if p[i] == '/' {
			if !matchedSlash {
				end = i
				break
			}
		} else {
			matchedSlash = false
		}
	}
	if end == -1 {
		if hasRoot {
			return "/"
		}
		return "."
	}
	if hasRoot && end == 1 {
		return "//"
	}
	return p[:end]
}

// Basename is path.basename. When ext is given and the base name ends with it, it is removed.
func Basename(p string, ext ...string) string {
	p = strings.TrimRight(p, "/")
	if p == "" {
		return ""
	}
	slash := strings.LastIndex(p, "/")
	base := p[slash+1:]
	if len(ext) > 0 && ext[0] != "" && strings.HasSuffix(base, ext[0]) {
		if ext[0] != base {
			return base[:len(base)-len(ext[0])]
		}
		// Node quirk: basename("ts", "ts") is "" but basename("a/ts", "ts") is "ts"
		if slash == -1 {
			return ""
		}
	}
	return base
}

// Extname is path.extname.
func Extname(p string) string {
	base := Basename(p)
	// a leading dot does not start an extension (".bashrc", "..")
	start := 0
	for start < len(base) && base[start] == '.' {
		start++
	}
	if start == len(base) {
		return ""
	}
	i := strings.LastIndex(base, ".")
	if i < start {
		if start > 1 && i == start-1 {
			// "..a" -> ".a"
			return base[i:]
		}
		return ""
	}
	return base[i:]
}
