package transform

// Port of TSTransformer/util/pointer.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// Pointer is Pointer<T>; Value is a *luau.Map, *luau.Array or an identifier.
type Pointer struct {
	Name  string
	Value luau.Expression
}

type (
	MapPointer   = Pointer
	ArrayPointer = Pointer
)

// createMapPointer is TSTransformer/util/pointer.ts createMapPointer.
func createMapPointer(name string) *MapPointer {
	panic(todo("util/pointer.ts", "createMapPointer"))
}

// createArrayPointer is TSTransformer/util/pointer.ts createArrayPointer.
func createArrayPointer(name string) *ArrayPointer {
	panic(todo("util/pointer.ts", "createArrayPointer"))
}

// assignToMapPointer is TSTransformer/util/pointer.ts assignToMapPointer.
func assignToMapPointer(prereqs *Prereqs, ptr *Pointer, left luau.Expression, right luau.Expression) {
	panic(todo("util/pointer.ts", "assignToMapPointer"))
}

// disableMapInline is TSTransformer/util/pointer.ts disableMapInline.
func disableMapInline(prereqs *Prereqs, ptr *MapPointer) {
	panic(todo("util/pointer.ts", "disableMapInline"))
}

// disableArrayInline is TSTransformer/util/pointer.ts disableArrayInline.
func disableArrayInline(prereqs *Prereqs, ptr *ArrayPointer) {
	panic(todo("util/pointer.ts", "disableArrayInline"))
}
