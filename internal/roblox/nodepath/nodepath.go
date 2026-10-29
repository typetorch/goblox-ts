// Package nodepath implements the parts of Node.js's `path` module (POSIX flavour) that roblox-ts,
// rojo-resolver and path-translator use, with Node's exact semantics: Go's path/filepath differs on
// empty results ("" vs "."), trailing separators, dot files and relative paths.
package nodepath

import (
	"os"
	"strings"
)

// Sep is path.sep.
const Sep = "/"

// Normalize is path.normalize.
func Normalize(p string) string {
	if p == "" {
		return "."
	}
	isAbsolute := strings.HasPrefix(p, "/")
	trailingSeparator := strings.HasSuffix(p, "/")
	p = normalizeString(p, !isAbsolute)
	if p == "" {
		if isAbsolute {
			return "/"
		}
		if trailingSeparator {
			return "./"
		}
		return "."
	}
	if trailingSeparator {
		p += "/"
	}
	if isAbsolute {
		return "/" + p
	}
	return p
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
	return cwd
}

// Resolve is path.resolve.
func Resolve(paths ...string) string {
	resolved := ""
	resolvedAbsolute := false
	for i := len(paths) - 1; i >= -1 && !resolvedAbsolute; i-- {
		var p string
		if i >= 0 {
			p = paths[i]
		} else {
			p = Cwd()
		}
		if p == "" {
			continue
		}
		resolved = p + "/" + resolved
		resolvedAbsolute = strings.HasPrefix(p, "/")
	}
	resolved = normalizeString(resolved, !resolvedAbsolute)
	if resolvedAbsolute {
		return "/" + resolved
	}
	if resolved == "" {
		return "."
	}
	return resolved
}

// IsAbsolute is path.isAbsolute.
func IsAbsolute(p string) bool { return strings.HasPrefix(p, "/") }

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
	fromParts := splitNonEmpty(from)
	toParts := splitNonEmpty(to)
	i := 0
	for i < len(fromParts) && i < len(toParts) && fromParts[i] == toParts[i] {
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
