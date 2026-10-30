package rojo

import "testing"

func TestParseRojoFileJson(t *testing.T) {
	t.Parallel()
	tree := `"tree":{"$className":"DataModel"}`
	cases := []struct{ name, text, wantError string }{
		{"valid", `{"name":"a",` + tree + `}`, ""},
		{"duplicate name, the last one wins", `{"name":1,"name":"a",` + tree + `}`, ""},
		{"text after the value is ignored", `{"name":"a",` + tree + `} trailing`, ""},
		{"integer servePort", `{"name":"a","servePort":34872,` + tree + `}`, ""},
		{"fractional servePort", `{"name":"a","servePort":1.5,` + tree + `}`, "data/servePort must be integer"},
		{"exponent servePort", `{"name":"a","servePort":1e3,` + tree + `}`, "data/servePort must be integer"},
		{"string servePort", `{"name":"a","servePort":"1",` + tree + `}`, "data/servePort must be integer"},
		{"null name", `{"name":null,` + tree + `}`, "data/name must be string"},
		{"not an object", `[]`, "data must be object"},
		{"syntax error", `{"name":"a",}`, "data must be object"},
		{"unterminated", `{"name":"a"`, "data must be object"},
		{"empty", ``, "data must be object"},
		{"missing tree", `{"name":"a"}`, "data must have required property 'tree'"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if _, got := parseRojoFile(c.text); got != c.wantError {
				t.Errorf("got %q, want %q", got, c.wantError)
			}
		})
	}
}
