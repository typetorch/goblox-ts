package transform

// Port of TSTransformer/util/pointer.ts.

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
	return &MapPointer{Name: name, Value: luau.NewMap()}
}

// createArrayPointer is TSTransformer/util/pointer.ts createArrayPointer.
func createArrayPointer(name string) *ArrayPointer {
	return &ArrayPointer{Name: name, Value: luau.NewArray()}
}

// assignToMapPointer is TSTransformer/util/pointer.ts assignToMapPointer.
func assignToMapPointer(prereqs *Prereqs, ptr *Pointer, left luau.Expression, right luau.Expression) {
	if m, ok := ptr.Value.(*luau.Map); ok {
		m.Fields.Push(luau.Create(&luau.MapField{
			Index: left,
			Value: right,
		}))
	} else {
		prereqs.Push(luau.Create(&luau.Assignment{
			Left: luau.Create(&luau.ComputedIndexExpression{
				Expression: ptr.Value,
				Index:      left,
			}),
			Operator: "=",
			Right:    right,
		}))
	}
}

// disableMapInline is TSTransformer/util/pointer.ts disableMapInline.
func disableMapInline(prereqs *Prereqs, ptr *MapPointer) {
	if luau.IsMap(ptr.Value) {
		ptr.Value = prereqs.PushToVar(ptr.Value, ptr.Name)
	}
}

// disableArrayInline is TSTransformer/util/pointer.ts disableArrayInline.
func disableArrayInline(prereqs *Prereqs, ptr *ArrayPointer) {
	if luau.IsArray(ptr.Value) {
		ptr.Value = prereqs.PushToVar(ptr.Value, ptr.Name)
	}
}
