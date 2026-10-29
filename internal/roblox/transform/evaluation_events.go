package transform

// Port of TSTransformer/util/evaluation/events.ts.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

type EvaluationReference struct {
	OperandIndex int
	Expression   luau.Expression
	Effects      *EvaluationEffects
}

type EvaluationEvent struct {
	Effects      *EvaluationEffects
	OperandIndex int // -1 when absent
	Conditional  bool
}

var MACRO_STATEMENTS = map[luau.SyntaxKind]bool{
	luau.KindVariableDeclaration: true,
	luau.KindAssignment:          true,
	luau.KindIfStatement:         true,
	luau.KindForStatement:        true,
	luau.KindNumericForStatement: true,
	luau.KindCallStatement:       true,
	luau.KindComment:             true,
	luau.KindBreakStatement:      true,
}

// getEvaluationEvents is TSTransformer/util/evaluation/events.ts getEvaluationEvents.
//
// field order differs from execution order when Luau reads a local's register
// directly; conditional uses must still preserve eager argument evaluation
func getEvaluationEvents(statements *luau.List, result luau.Expression, references map[int]*EvaluationReference) []*EvaluationEvent {
	var events []*EvaluationEvent
	mayExit := false
	getReference := func(node luau.Node) *EvaluationReference {
		if temporary, ok := node.(*luau.TemporaryIdentifier); ok {
			return references[temporary.Id]
		}
		return nil
	}
	isLateReference := func(node luau.Expression) bool {
		reference := getReference(node)
		if reference == nil {
			return isLateRead(node)
		}
		return isLateRead(reference.Expression)
	}

	var visit func(node luau.Node, conditional bool)
	visit = func(node luau.Node, conditional bool) {
		if reference := getReference(node); reference != nil {
			events = append(events, &EvaluationEvent{
				Effects:      reference.Effects,
				OperandIndex: reference.OperandIndex,
				Conditional:  conditional || mayExit,
			})
			return
		}
		if isConstantReference(node) {
			return
		}
		// source callbacks are opaque operands. New macro-owned control flow needs
		// an ordering rule here before it can safely contain operand references.
		assert(!luau.IsStatement(node) || MACRO_STATEMENTS[node.Kind()])
		assert(!luau.IsFunctionExpression(node))
		visitExpression := func(child luau.Node) { visit(child, conditional) }
		visitStatements := func(nodes *luau.List) {
			nodes.ForEach(func(child luau.Node) { visit(child, true) })
		}
		switch n := node.(type) {
		case *luau.BinaryExpression:
			shortCircuit := n.Operator == "and" || n.Operator == "or"
			if !shortCircuit && n.Operator != ".." && isLateReference(n.Left) {
				visitExpression(n.Right)
				visitExpression(n.Left)
			} else {
				visitExpression(n.Left)
				visit(n.Right, conditional || shortCircuit)
			}
		case *luau.ComputedIndexExpression:
			if isLateReference(n.Expression) {
				visitExpression(n.Index)
				visitExpression(n.Expression)
			} else {
				visitExpression(n.Expression)
				visitExpression(n.Index)
			}
		case *luau.CallExpression:
			visitExpression(n.Expression)
			n.Args.ForEach(visitExpression)
		case *luau.MethodCallExpression:
			visitExpression(n.Expression)
			n.Args.ForEach(visitExpression)
		case *luau.VariableDeclaration:
			assert(!luau.IsList(n.Left))
			assert(!luau.IsList(n.Right))
			assert(getReference(n.Left.(luau.Node)) == nil, "A macro must not redeclare an operand reference")
			if n.Right != nil {
				visitExpression(n.Right.(luau.Node))
			}
		case *luau.Assignment:
			assert(!luau.IsList(n.Left))
			assert(!luau.IsList(n.Right))
			target := n.Left.(luau.Node)
			var delayed []luau.Expression
			assert(getReference(target) == nil, "A macro must not assign to an operand reference")
			if !luau.IsAnyIdentifier(target) {
				index, ok := target.(*luau.ComputedIndexExpression)
				assert(ok)
				if isLateReference(index.Expression) {
					delayed = append(delayed, index.Expression)
				} else {
					visitExpression(index.Expression)
				}
				if isLateReference(index.Index) {
					delayed = append(delayed, index.Index)
				} else {
					visitExpression(index.Index)
				}
			}
			if n.Operator != "=" {
				visitExpression(target)
			}
			visitExpression(n.Right.(luau.Node))
			for _, expression := range delayed {
				visitExpression(expression)
			}
		case *luau.IfExpression:
			visitExpression(n.Condition)
			visit(n.Expression, true)
			visit(n.Alternative, true)
		case *luau.IfStatement:
			elseBody, ok := n.ElseBody.(*luau.List)
			assert(ok)
			visitExpression(n.Condition)
			visitStatements(n.Statements)
			visitStatements(elseBody)
		case *luau.ForStatement:
			visitExpression(n.Expression)
			visitStatements(n.Statements)
		case *luau.NumericForStatement:
			assert(n.Step == nil)
			visitExpression(n.Start)
			visitExpression(n.End)
			visitStatements(n.Statements)
		default:
			forEachChild(node, visitExpression)
		}
		if luau.IsBreakStatement(node) {
			mayExit = true
		}
		events = append(events, &EvaluationEvent{Effects: getIntrinsicEffects(node), OperandIndex: -1})
	}
	statements.ForEach(func(statement luau.Node) { visit(statement, false) })
	visit(result, false)

	return events
}
