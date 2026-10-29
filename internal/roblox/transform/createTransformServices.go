package transform

// Port of TSTransformer/util/createTransformServices.ts.

import (
	"github.com/microsoft/typescript-go/internal/checker"
)

func createTransformServices(typeChecker *checker.Checker) *TransformServices {
	macroManager := NewMacroManager(typeChecker)

	return &TransformServices{MacroManager: macroManager}
}
