package nodepath

import (
	"strings"
	"testing"
)

// Expected values are Node.js 26 results (node -p 'require("path").posix.<fn>(...)').
func TestNodeSemantics(t *testing.T) {
	Cwd = func() string { return "/cwd" }
	cases := []struct{ got, want string }{
		{Join("a", "b"), "a/b"},
		{Join("a", "", "b/"), "a/b/"},
		{Join(), "."},
		{Join("", ""), "."},
		{Join("/a", "../../b"), "/b"},
		{Join("a", "../.."), ".."},
		{Join("./a"), "a"},
		{Resolve("a"), "/cwd/a"},
		{Resolve("/x", "y", "../z"), "/x/z"},
		{Resolve("/x/"), "/x"},
		{Resolve(""), "/cwd"},
		{Relative("/a/b", "/a/b"), ""},
		{Relative("/a/b", "/a/c/d"), "../c/d"},
		{Relative("/a/b/c", "/a"), "../.."},
		{Relative("/a", "/a/b"), "b"},
		{Relative("/", "/a"), "a"},
		{Dirname("/a/b/"), "/a"},
		{Dirname("/a"), "/"},
		{Dirname("a"), "."},
		{Dirname("a/b"), "a"},
		{Dirname("/"), "/"},
		{Basename("/a/b.ts"), "b.ts"},
		{Basename("/a/b.ts", ".ts"), "b"},
		{Basename("/a/b/"), "b"},
		{Basename(".ts", ".ts"), ""},
		{Basename("aaa/bbb", "bbb"), "bbb"},
		{Basename("/a/.ts", ".ts"), ".ts"},
		{Extname("..a"), ".a"},
		{Extname("a/..b"), ".b"},
		{Extname("a.ts"), ".ts"},
		{Extname("a.d.ts"), ".ts"},
		{Extname(".bashrc"), ""},
		{Extname(".a.b"), ".b"},
		{Extname("a."), "."},
		{Extname(".."), ""},
		{Extname("dir/"), ""},
		{Extname("a.b/"), ".b"},
		{Normalize("a//b/../c/."), "a/c"},
		{Normalize("./"), "./"},
		{Normalize("/.."), "/"},
	}
	for i, c := range cases {
		if c.got != c.want {
			t.Errorf("case %d: got %q, want %q", i, c.got, c.want)
		}
	}
}

func TestWindowsRoots(t *testing.T) {
	Cwd = func() string { return "C:/cwd" }
	cases := []struct{ got, want string }{
		{Resolve("a"), "C:/cwd/a"},
		{Resolve("C:/x", "y", "../z"), "C:/x/z"},
		{Resolve("/x", "D:/y"), "D:/y"},
		{Resolve("C:/.."), "C:/"},
		{Normalize("C:/a/../b/"), "C:/b/"},
		{Relative("C:/a/b", "C:/a/c/d"), "../c/d"},
		{Relative("C:/A/b", "c:/a/b/x"), "x"},
		{Relative("C:/a", "D:/a"), "D:/a"},
		{Dirname("C:/a/b"), "C:/a"},
		{Dirname("C:/a"), "C:/"},
		{Dirname("C:/"), "C:/"},
		{Join("C:/a", "../b"), "C:/b"},
	}
	for i, c := range cases {
		if c.got != c.want {
			t.Errorf("case %d: got %q, want %q", i, c.got, c.want)
		}
	}
}

// isPathDescendantOfReference is the original, allocating implementation that IsPathDescendantOf must match.
func isPathDescendantOfReference(filePath, dirPath string) bool {
	return dirPath == filePath || !strings.HasPrefix(Relative(dirPath, filePath), "..")
}

func TestIsPathDescendantOf(t *testing.T) {
	Cwd = func() string { return "/cwd" }
	cases := []struct {
		file, dir string
		want      bool
	}{
		{"/a/b", "/a/b", true},        // same path
		{"/a/b/", "/a/b", true},       // trailing slash on the file
		{"/a/b", "/a/b/", true},       // trailing slash on the dir
		{"/a/b/c", "/a/b/", true},     // child, dir with trailing slash
		{"/a/b/c.ts", "/a/b", true},   // child
		{"/a/b//c", "/a//b", true},    // doubled separators
		{"/a/bc", "/a/b", false},      // sibling with a common prefix
		{"/a/b", "/a/bc", false},      // sibling, the other way
		{"/a", "/a/b", false},         // ancestor
		{"/", "/a", false},            // root is an ancestor
		{"/a", "/", true},             // everything is under the root
		{"/a/b/../c", "/a/b", false},  // ".." leaves the dir
		{"/a/b/../b/c", "/a/b", true}, // ".." comes back in
		{"/a/./b", "/a", true},        // "." segment
		{"/a/b", "/a/b/c/..", true},   // ".." in the dir
		{"/a/..b", "/a", false},       // quirk: relative() is "..b", which starts with ".."
		{"/a/x/..b", "/a", true},      // only the first remaining segment counts
		{"/a/...", "/a", false},       // same quirk
		{"/A/b", "/a", false},         // POSIX paths are case-sensitive
		{"a/b", "a", true},            // relative paths resolve against Cwd
		{"/cwd/a/b", "a", true},       // relative dir, absolute file
		{"C:/a/b", "C:/a", true},      // drive paths
		{"c:/A/b", "C:/a", true},      // drive paths compare case-insensitively
		{"C:/ab", "C:/a", false},      // sibling on a drive
		{"C:/a", "C:/a/b", false},     // ancestor on a drive
		{"C:\a/b", "C:/a", true},      // backslash after the drive letter is a root separator
		{"C:/a/b", "C:\a", true},      // same, on the dir
		{"C:/a\b", "C:/a", false},     // other backslashes are file name characters, like path.posix
		{"C:/a\b/c", "C:/a\b", true},  // mixed separators, same on both sides
		{"D:/a", "C:/a", true},        // quirk: different drives, relative() returns the target
		{"/a/b", "C:/a", true},        // quirk: POSIX root vs drive root
		{"C:/", "C:/a", false},        // drive root is an ancestor
		{"C:/a", "C:/", true},         // everything is under the drive root
		{"C:/a", "C:", true},          // bare drive
		{"C:/Ä/b", "C:/ä", true},      // case folding is Unicode-aware like strings.EqualFold
	}
	for _, c := range cases {
		if got := IsPathDescendantOf(c.file, c.dir); got != c.want {
			t.Errorf("IsPathDescendantOf(%q, %q) = %v, want %v", c.file, c.dir, got, c.want)
		}
		if ref := isPathDescendantOfReference(c.file, c.dir); ref != c.want {
			t.Errorf("reference(%q, %q) = %v, want %v (test expectation is wrong)", c.file, c.dir, ref, c.want)
		}
	}
}

// TestIsPathDescendantOfMatchesReference compares against the original implementation on every pair of
// generated paths.
func TestIsPathDescendantOfMatchesReference(t *testing.T) {
	Cwd = func() string { return "C:/cwd" }
	roots := []string{"", "/", "//", "C:/", "c:/", "C:\\", "D:/", "C:"}
	segs := []string{"a", "A", "ab", ".", "..", "..a", "", "a\\b", "cwd", "Ä"}
	var paths []string
	for _, r := range roots {
		paths = append(paths, r)
		for _, s1 := range segs {
			paths = append(paths, r+s1, r+s1+"/")
			for _, s2 := range segs {
				paths = append(paths, r+s1+"/"+s2)
				for _, s3 := range []string{"ä", ".."} {
					paths = append(paths, r+s1+"/"+s2+"/"+s3)
				}
			}
		}
	}
	for _, file := range paths {
		for _, dir := range paths {
			if got, want := IsPathDescendantOf(file, dir), isPathDescendantOfReference(file, dir); got != want {
				t.Fatalf("IsPathDescendantOf(%q, %q) = %v, reference = %v", file, dir, got, want)
			}
		}
	}
}

func BenchmarkIsPathDescendantOf(b *testing.B) {
	file := "C:/Users/me/game/out/client/controllers/grenade.controller.luau"
	dir := "C:/Users/me/game/out/shared"
	b.ReportAllocs()
	for b.Loop() {
		IsPathDescendantOf(file, dir)
	}
}

func BenchmarkIsPathDescendantOfReference(b *testing.B) {
	file := "C:/Users/me/game/out/client/controllers/grenade.controller.luau"
	dir := "C:/Users/me/game/out/shared"
	b.ReportAllocs()
	for b.Loop() {
		isPathDescendantOfReference(file, dir)
	}
}
