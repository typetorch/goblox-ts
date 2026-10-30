package rojo

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/internal/json"
)

// RojoTree is a node of a Rojo project's "tree". Children keep JavaScript's Object.keys order.
type RojoTree struct {
	ClassName string
	// Path is `$path`, or `$path.optional`. nil when absent.
	Path     *string
	Children []RojoTreeChild
}

// RojoTreeChild is a named child of a RojoTree.
type RojoTreeChild struct {
	Name string
	Tree *RojoTree
}

// RojoFile is a parsed *.project.json.
type RojoFile struct {
	Name string
	Tree *RojoTree
}

// orderedObject is a JSON object with its keys in JavaScript's Object.keys order.
type orderedObject struct {
	keys   []string
	values map[string]any
}

// jsonNumber is a JSON number's source text.
type jsonNumber string

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return nil, err
	}
	switch tok.Kind() {
	case '{':
		obj := &orderedObject{values: map[string]any{}}
		for dec.PeekKind() != '}' {
			keyTok, err := dec.ReadToken()
			if err != nil {
				return nil, err
			}
			key := keyTok.String()
			value, err := decodeValue(dec)
			if err != nil {
				return nil, err
			}
			if _, seen := obj.values[key]; !seen {
				obj.keys = append(obj.keys, key)
			}
			obj.values[key] = value
		}
		if _, err := dec.ReadToken(); err != nil {
			return nil, err
		}
		jsKeyOrder(obj.keys)
		return obj, nil
	case '[':
		var arr []any
		for dec.PeekKind() != ']' {
			value, err := decodeValue(dec)
			if err != nil {
				return nil, err
			}
			arr = append(arr, value)
		}
		if _, err := dec.ReadToken(); err != nil {
			return nil, err
		}
		return arr, nil
	case '"':
		return tok.String(), nil
	case '0':
		return jsonNumber(tok.String()), nil
	case 't', 'f':
		return tok.Bool(), nil
	}
	return nil, nil
}

// isArrayIndex reports whether key is a canonical array index ("0", "1", ... up to 2^32 - 2).
func isArrayIndex(key string) (uint64, bool) {
	n, err := strconv.ParseUint(key, 10, 32)
	if err != nil || n == 1<<32-1 || strconv.FormatUint(n, 10) != key {
		return 0, false
	}
	return n, true
}

// jsKeyOrder sorts keys the way JavaScript orders an object's own keys: array indices ascending
// first, then the other keys in insertion order.
func jsKeyOrder(keys []string) {
	sort.SliceStable(keys, func(i, j int) bool {
		a, aIsIndex := isArrayIndex(keys[i])
		b, bIsIndex := isArrayIndex(keys[j])
		if aIsIndex && bIsIndex {
			return a < b
		}
		return aIsIndex && !bIsIndex
	})
}

// parseRojoFile parses and validates a *.project.json against rojo-resolver's rojo-schema.json. Like
// ajv (without allErrors) it stops at the first failing keyword, in schema order, and returns the
// message ajv's errorsText would.
func parseRojoFile(text string) (*RojoFile, string) {
	// duplicate names are fine: the last one wins, as in JSON.parse
	reader := bytes.NewReader([]byte(text))
	dec := json.NewDecoder(reader)
	dec.Reset(reader, json.AllowDuplicateNames(true))
	value, err := decodeValue(dec)
	if err != nil {
		return nil, "data must be object"
	}
	root, ok := value.(*orderedObject)
	if !ok {
		return nil, "data must be object"
	}
	for _, required := range []string{"name", "tree"} {
		if _, present := root.values[required]; !present {
			return nil, fmt.Sprintf("data must have required property '%s'", required)
		}
	}
	name, ok := root.values["name"].(string)
	if !ok {
		return nil, "data/name must be string"
	}
	if port, ok := root.values["servePort"]; ok {
		if n, isNumber := port.(jsonNumber); !isNumber || strings.ContainsAny(string(n), ".eE") {
			return nil, "data/servePort must be integer"
		}
	}
	tree, errText := validateTree(root.values["tree"], "data/tree")
	if errText != "" {
		return nil, errText
	}
	return &RojoFile{Name: name, Tree: tree}, ""
}

func jsonPointerEscape(key string) string {
	return strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}

func validateTree(raw any, at string) (*RojoTree, string) {
	obj, ok := raw.(*orderedObject)
	if !ok {
		return nil, at + " must be object"
	}
	tree := &RojoTree{}
	// properties, in schema order
	if value, ok := obj.values["$className"]; ok {
		s, ok := value.(string)
		if !ok {
			return nil, at + "/$className must be string"
		}
		tree.ClassName = s
	}
	if value, ok := obj.values["$ignoreUnknownInstances"]; ok {
		if _, ok := value.(bool); !ok {
			return nil, at + "/$ignoreUnknownInstances must be boolean"
		}
	}
	if value, ok := obj.values["$path"]; ok {
		switch v := value.(type) {
		case string:
			tree.Path = &v
		case *orderedObject:
			if optional, ok := v.values["optional"]; ok {
				s, ok := optional.(string)
				if !ok {
					return nil, at + "/$path must be string, " + at + "/$path/optional must be string, " + at + "/$path must match a schema in anyOf"
				}
				tree.Path = &s
			}
		default:
			return nil, at + "/$path must be string, " + at + "/$path must be object, " + at + "/$path must match a schema in anyOf"
		}
	}
	if value, ok := obj.values["$properties"]; ok {
		if _, ok := value.(*orderedObject); !ok {
			return nil, at + "/$properties must be object"
		}
	}
	// patternProperties "^[^\$].*$", in data order
	for _, key := range obj.keys {
		if key == "" || strings.HasPrefix(key, "$") {
			continue
		}
		child, errText := validateTree(obj.values[key], at+"/"+jsonPointerEscape(key))
		if errText != "" {
			return nil, errText
		}
		tree.Children = append(tree.Children, RojoTreeChild{Name: key, Tree: child})
	}
	return tree, ""
}
