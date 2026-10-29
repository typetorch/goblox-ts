package nodepath

import "testing"

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
