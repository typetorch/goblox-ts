package diagnostics

import "github.com/microsoft/typescript-go/internal/ast"

// ProjectError is Shared/errors/ProjectError.ts: a fatal error of the project setup. The compiler
// panics with it and the CLI prints its message.
type ProjectError struct {
	Message string
}

func (e *ProjectError) Error() string { return e.Message }

// DiagnosticError is Shared/errors/DiagnosticError.ts: fatal diagnostics.
type DiagnosticError struct {
	Diagnostics []*ast.Diagnostic
}

func (e *DiagnosticError) Error() string { return "DiagnosticError" }
