// Package diagnostics is the port of roblox-ts's src/Shared/diagnostics.ts: the catalogue of roblox-ts
// errors and warnings. Messages are byte-identical to roblox-ts's.
package diagnostics

import (
	"strings"
	"sync/atomic"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
	"github.com/microsoft/typescript-go/internal/scanner"
)

// Source is the diagnostic source of every roblox-ts diagnostic. roblox-ts sets the code to
// " roblox-ts", which TypeScript prints as "error TS roblox-ts: ...".
const Source = "roblox-ts"

const repoURL = "https://github.com/roblox-ts/roblox-ts"

func suggestion(text string) string { return "Suggestion: " + Yellow(text) }

// issue is Shared/util/createGithubLink.ts.
func issue(id int) string {
	return "More information: " + Grey(repoURL+"/issues/"+itoa(id))
}

// DiagnosticFactory creates a diagnostic of one kind. Id is the roblox-ts diagnostic id.
type DiagnosticFactory struct {
	Id               int
	category         tsdiagnostics.Category
	messages         []string
	contextFormatter func(context ...any) []string
}

var nextId atomic.Int32

func diagnosticWithContext(category tsdiagnostics.Category, contextFormatter func(context ...any) []string, messages ...string) *DiagnosticFactory {
	return &DiagnosticFactory{
		Id:               int(nextId.Add(1) - 1),
		category:         category,
		messages:         messages,
		contextFormatter: contextFormatter,
	}
}

func diagnostic(category tsdiagnostics.Category, messages ...string) *DiagnosticFactory {
	return diagnosticWithContext(category, nil, messages...)
}

func errorFactory(messages ...string) *DiagnosticFactory {
	return diagnostic(tsdiagnostics.CategoryError, messages...)
}

func errorWithContext(contextFormatter func(context ...any) []string, messages ...string) *DiagnosticFactory {
	return diagnosticWithContext(tsdiagnostics.CategoryError, contextFormatter, messages...)
}

func warning(messages ...string) *DiagnosticFactory {
	return diagnostic(tsdiagnostics.CategoryWarning, messages...)
}

// joinMessages joins messages with "\n", skipping the `false` entries (empty strings here).
func joinMessages(messages []string) string {
	var parts []string
	for _, m := range messages {
		if m != "" {
			parts = append(parts, m)
		}
	}
	return strings.Join(parts, "\n")
}

func (f *DiagnosticFactory) messageText(context []any) string {
	messages := f.messages
	if f.contextFormatter != nil {
		messages = append(append([]string{}, messages...), f.contextFormatter(context...)...)
	}
	return joinMessages(messages)
}

// Create is calling the factory with a node: the diagnostic spans the node without its leading trivia.
func (f *DiagnosticFactory) Create(node *ast.Node, context ...any) *ast.Diagnostic {
	file := ast.GetSourceFileOfNode(node)
	start := scanner.GetTokenPosOfNode(node, file, false)
	return createDiagnosticWithLocation(f.Id, f.messageText(context), f.category, file, start, node.End()-start)
}

// CreateWithRange is calling the factory with a SourceFileWithTextRange. Like roblox-ts, the length is
// range.End (not End - Pos).
func (f *DiagnosticFactory) CreateWithRange(file *ast.SourceFile, textRange core.TextRange, context ...any) *ast.Diagnostic {
	return createDiagnosticWithLocation(f.Id, f.messageText(context), f.category, file, textRange.Pos(), textRange.End())
}

// createDiagnosticWithLocation is Shared/util/createDiagnosticWithLocation.ts. The roblox-ts id is
// stored as the diagnostic's code (see GetDiagnosticId).
func createDiagnosticWithLocation(id int, messageText string, category tsdiagnostics.Category, file *ast.SourceFile, start int, length int) *ast.Diagnostic {
	d := ast.NewDiagnosticFromSerialized(file, core.NewTextRange(start, start+length), int32(id), category, "", nil, nil, nil, false, false, false)
	return d.SetExternalData(Source, messageText)
}

// CreateTextDiagnostic is Shared/util/createTextDiagnostic.ts: a diagnostic without a location.
func CreateTextDiagnostic(messageText string, category tsdiagnostics.Category) *ast.Diagnostic {
	d := ast.NewDiagnosticFromSerialized(nil, core.NewTextRange(-1, -1), -1, category, "", nil, nil, nil, false, false, false)
	return d.SetExternalData(Source, messageText)
}

func diagnosticText(category tsdiagnostics.Category, messages ...string) *ast.Diagnostic {
	return CreateTextDiagnostic(joinMessages(messages), category)
}

func errorText(messages ...string) *ast.Diagnostic {
	return diagnosticText(tsdiagnostics.CategoryError, messages...)
}

func warningText(messages ...string) *ast.Diagnostic {
	return diagnosticText(tsdiagnostics.CategoryWarning, messages...)
}

// IsRobloxTsDiagnostic reports whether d was created by this package.
func IsRobloxTsDiagnostic(d *ast.Diagnostic) bool { return d.Source() == Source }

// GetDiagnosticId returns the roblox-ts id of a diagnostic created by a DiagnosticFactory.
func GetDiagnosticId(d *ast.Diagnostic) int { return int(d.Code()) }
