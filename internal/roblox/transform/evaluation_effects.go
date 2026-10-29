package transform

// Port of TSTransformer/util/evaluation/effects.ts.

import (
	"github.com/microsoft/typescript-go/internal/jsnum"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// bindingWildcard is the TypeScript `symbol` Binding: a sentinel for every binding user code can access.
type bindingWildcard struct{ description string }

var (
	USER_READS  = &bindingWildcard{"bindings readable by user code"}
	USER_WRITES = &bindingWildcard{"bindings writable by user code"}
)

// EvaluationEffects are the effects of executing emitted code, not of merely constructing its AST.
type EvaluationEffects struct {
	// bindings whose current values may be observed
	Reads map[Binding]bool
	// bindings that may be reassigned, not mutations to their referenced objects
	Writes map[Binding]bool
	// may read mutable shared state, such as table entries or Instance properties
	ReadsHeap bool
	// may mutate shared state without reassigning a binding
	WritesHeap bool
	// may error or fail to terminate, preventing later evaluations
	Throws bool
	// may create a fresh identity, such as a table or closure, that cannot be duplicated
	Allocates bool
}

// The binding sets of shared effects are never mutated: every function that adds bindings builds new sets.

var NO_EFFECTS = &EvaluationEffects{
	Reads:  map[Binding]bool{},
	Writes: map[Binding]bool{},
}

var UNKNOWN_EFFECTS = &EvaluationEffects{
	Reads:      map[Binding]bool{USER_READS: true},
	Writes:     map[Binding]bool{USER_WRITES: true},
	ReadsHeap:  true,
	WritesHeap: true,
	Throws:     true,
	Allocates:  true,
}

func withEffects(base *EvaluationEffects, update func(effects *EvaluationEffects)) *EvaluationEffects {
	effects := *base
	update(&effects)
	return &effects
}

var (
	MAY_THROW  = withEffects(NO_EFFECTS, func(e *EvaluationEffects) { e.Throws = true })
	READ_HEAP  = withEffects(MAY_THROW, func(e *EvaluationEffects) { e.ReadsHeap = true })
	WRITE_HEAP = withEffects(READ_HEAP, func(e *EvaluationEffects) { e.WritesHeap = true })
	ALLOCATION = withEffects(NO_EFFECTS, func(e *EvaluationEffects) { e.Allocates = true })
)

func isNumberBinding(key Binding) bool {
	_, ok := key.(int)
	return ok
}

// getFunctionEffects is TSTransformer/util/evaluation/effects.ts getFunctionEffects: summarize the actual
// lowered body; parameters and locals belong to this invocation.
func getFunctionEffects(parameters *luau.List, statements *luau.List) *EvaluationEffects {
	locals := map[Binding]bool{}
	parameters.ForEach(func(parameter luau.Node) { locals[getBinding(parameter)] = true })
	var collect func(node luau.Node)
	collect = func(node luau.Node) {
		if luau.IsFunctionExpression(node) || luau.IsFunctionDeclaration(node) {
			return
		}
		if declaration, ok := node.(*luau.VariableDeclaration); ok {
			if ids, ok := declaration.Left.(*luau.List); ok {
				ids.ForEach(func(id luau.Node) { locals[getBinding(id)] = true })
			} else {
				locals[getBinding(declaration.Left.(luau.Node))] = true
			}
		}
		if numericFor, ok := node.(*luau.NumericForStatement); ok {
			locals[getBinding(numericFor.Id)] = true
		}
		if forStatement, ok := node.(*luau.ForStatement); ok {
			forStatement.Ids.ForEach(func(id luau.Node) { locals[getBinding(id)] = true })
		}
		forEachChild(node, collect)
	}
	statements.ForEach(collect)
	effects := getStatementsEffects(statements)
	result := *effects
	result.Reads = map[Binding]bool{}
	for key := range effects.Reads {
		if !locals[key] && !isNumberBinding(key) {
			result.Reads[key] = true
		}
	}
	result.Writes = map[Binding]bool{}
	for key := range effects.Writes {
		if !locals[key] && !isNumberBinding(key) {
			result.Writes[key] = true
		}
	}
	return &result
}

func unionBindings(a map[Binding]bool, b map[Binding]bool) map[Binding]bool {
	if len(b) == 0 {
		return a
	}
	if len(a) == 0 {
		return b
	}
	union := make(map[Binding]bool, len(a)+len(b))
	for key := range a {
		union[key] = true
	}
	for key := range b {
		union[key] = true
	}
	return union
}

// joinEffects is TSTransformer/util/evaluation/effects.ts joinEffects.
func joinEffects(effects ...*EvaluationEffects) *EvaluationEffects {
	result := NO_EFFECTS
	for _, b := range effects {
		a := result
		if a == NO_EFFECTS {
			result = b
			continue
		}
		if b == NO_EFFECTS || a == b {
			continue
		}
		result = &EvaluationEffects{
			Reads:      unionBindings(a.Reads, b.Reads),
			Writes:     unionBindings(a.Writes, b.Writes),
			ReadsHeap:  a.ReadsHeap || b.ReadsHeap,
			WritesHeap: a.WritesHeap || b.WritesHeap,
			Throws:     a.Throws || b.Throws,
			Allocates:  a.Allocates || b.Allocates,
		}
	}
	return result
}

func isExposedBinding(key Binding, wildcard *bindingWildcard) bool {
	if bindingKey, ok := key.(*BindingKey); ok {
		if wildcard == USER_WRITES {
			return bindingKey.WrittenByClosure
		}
		return bindingKey.Captured
	}
	// compiler temporaries are inaccessible to user code; untracked names fail closed
	return !isNumberBinding(key)
}

func hasExposedBinding(bindings map[Binding]bool, wildcard *bindingWildcard) bool {
	for key := range bindings {
		if isExposedBinding(key, wildcard) {
			return true
		}
	}
	return false
}

func intersects(a map[Binding]bool, b map[Binding]bool) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	for _, wildcard := range []*bindingWildcard{USER_READS, USER_WRITES} {
		if a[wildcard] && hasExposedBinding(b, wildcard) {
			return true
		}
		if b[wildcard] && hasExposedBinding(a, wildcard) {
			return true
		}
	}
	for key := range a {
		if b[key] {
			return true
		}
	}
	return false
}

func hasWrites(effects *EvaluationEffects) bool {
	if effects.WritesHeap {
		return true
	}
	for key := range effects.Writes {
		if !isNumberBinding(key) {
			return true
		}
	}
	return false
}

// effectsCommute is TSTransformer/util/evaluation/effects.ts effectsCommute, including errors: a failed
// operand must not allow a later mutation to happen.
func effectsCommute(a *EvaluationEffects, b *EvaluationEffects) bool {
	return !(intersects(a.Reads, b.Writes) ||
		intersects(a.Writes, b.Reads) ||
		intersects(a.Writes, b.Writes) ||
		(a.ReadsHeap && b.WritesHeap) ||
		(a.WritesHeap && (b.ReadsHeap || b.WritesHeap)) ||
		(a.Throws && (b.Throws || hasWrites(b))) ||
		(b.Throws && hasWrites(a)))
}

// canDiscard is TSTransformer/util/evaluation/effects.ts canDiscard.
func canDiscard(effects *EvaluationEffects) bool {
	return !effects.Throws && !hasWrites(effects)
}

// canRepeat is TSTransformer/util/evaluation/effects.ts canRepeat.
func canRepeat(effects *EvaluationEffects) bool {
	return canDiscard(effects) && !effects.Allocates
}

func init() {
	// `self` is compiler-owned and cannot be rebound by TypeScript source code.
	markConstantReference(luau.Globals.Self)

	// only compiler-owned references receive these contracts. Calls not listed here
	// may execute user code, yield, throw, or allocate a new identity.
	for _, expression := range []luau.Expression{luau.Globals.Type, luau.Globals.Typeof} {
		markBuiltinReference(expression, NO_EFFECTS)
	}
	for _, expression := range []luau.Expression{luau.Globals.Assert, luau.Globals.Error, luau.Globals.Select} {
		markBuiltinReference(expression, MAY_THROW)
	}
	for _, expression := range []luau.Expression{luau.Globals.Next, luau.Globals.Table.Unpack, luau.Globals.Unpack} {
		markBuiltinReference(expression, READ_HEAP)
	}
	for _, expression := range []luau.Expression{
		luau.Globals.Table.Insert,
		luau.Globals.Table.Remove,
		luau.Globals.Table.Clear,
		luau.Globals.Table.Move,
	} {
		markBuiltinReference(expression, WRITE_HEAP)
	}
	for _, expression := range []luau.Expression{luau.Globals.Table.Create, luau.Globals.Table.Pack} {
		markBuiltinReference(expression, withEffects(MAY_THROW, func(e *EvaluationEffects) { e.Allocates = true }))
	}
	markBuiltinReference(luau.Globals.Table.Concat, READ_HEAP)
	// the function lookup is stable, but searching can invoke user equality code.
	markConstantReference(luau.Globals.Table.Find)
	// stable lookup does not imply a pure call (e.g. gsub can invoke a callback).
	s := &luau.Globals.String
	for _, expression := range []luau.Expression{
		s.Byte, s.Find, s.Format, s.Gmatch, s.Gsub, s.Lower, s.Match, s.Rep, s.Reverse, s.Split, s.Sub, s.Upper,
	} {
		markConstantReference(expression)
	}
	markBuiltinReference(s.Split, withEffects(MAY_THROW, func(e *EvaluationEffects) { e.Allocates = true }))
	for _, expression := range []luau.Expression{s.Byte, s.Find, s.Lower, s.Match, s.Rep, s.Reverse, s.Sub, s.Upper} {
		markBuiltinReference(expression, MAY_THROW)
	}
}

// getIntrinsicEffects is TSTransformer/util/evaluation/effects.ts getIntrinsicEffects: direct effect,
// excluding child evaluation. Unknown syntax fails closed.
func getIntrinsicEffects(node luau.Node) *EvaluationEffects {
	if luau.IsAnyIdentifier(node) {
		return withEffects(NO_EFFECTS, func(e *EvaluationEffects) { e.Reads = map[Binding]bool{getBinding(node): true} })
	}
	if luau.IsSimplePrimitive(node) || luau.IsNone(node) || luau.IsVarArgsLiteral(node) {
		return NO_EFFECTS
	}
	switch n := node.(type) {
	case *luau.CallExpression:
		if effects := getBuiltinCallEffects(n.Expression); effects != nil {
			return effects
		}
		if effects := getCallEffects(n.Expression); effects != nil {
			return effects
		}
		return UNKNOWN_EFFECTS
	case *luau.MethodCallExpression:
		return UNKNOWN_EFFECTS
	case *luau.Array, *luau.Map, *luau.MixedTable:
		return ALLOCATION
	case *luau.Set:
		return withEffects(ALLOCATION, func(e *EvaluationEffects) { e.Throws = true })
	case *luau.MapField:
		if luau.IsStringLiteral(n.Index) {
			return NO_EFFECTS
		}
		if literal, ok := n.Index.(*luau.NumberLiteral); ok {
			value := jsnum.FromString(literal.Value)
			if !value.IsNaN() && !value.IsInf() {
				return NO_EFFECTS
			}
		}
		return MAY_THROW
	case *luau.PropertyAccessExpression, *luau.ComputedIndexExpression:
		if isStableLookup(node) {
			return NO_EFFECTS
		}
		return UNKNOWN_EFFECTS
	case *luau.InterpolatedString:
		if n.Parts.Every(func(part luau.Node) bool { return luau.IsInterpolatedStringPart(part) || isPrimitiveValue(part) }) {
			return NO_EFFECTS
		}
		return UNKNOWN_EFFECTS
	case *luau.UnaryExpression:
		if n.Operator == "#" && !isPrimitiveValue(n.Expression) {
			return UNKNOWN_EFFECTS
		}
		return NO_EFFECTS
	case *luau.BinaryExpression:
		if (n.Operator == "==" || n.Operator == "~=") && !isPrimitiveValue(n.Left) && !isPrimitiveValue(n.Right) {
			return UNKNOWN_EFFECTS
		}
		return NO_EFFECTS
	case *luau.ParenthesizedExpression, *luau.IfExpression:
		return NO_EFFECTS
	case *luau.Assignment:
		return getTargetsEffects(n.Left)
	case *luau.VariableDeclaration:
		return getTargetsEffects(n.Left)
	case *luau.ForStatement:
		// generalized iteration can call __iter
		return UNKNOWN_EFFECTS
	case *luau.WhileStatement, *luau.RepeatStatement:
		// moving an operand into a nonterminating loop can prevent its evaluation
		return UNKNOWN_EFFECTS
	case *luau.NumericForStatement:
		// non-finite bounds can fail to terminate
		return MAY_THROW
	case *luau.CallStatement,
		*luau.ReturnStatement,
		*luau.IfStatement,
		*luau.DoStatement,
		*luau.Comment,
		*luau.BreakStatement,
		*luau.ContinueStatement,
		*luau.InterpolatedStringPart:
		return NO_EFFECTS
	}
	return UNKNOWN_EFFECTS
}

// getTargetsEffects is the Assignment / VariableDeclaration case of getIntrinsicEffects.
func getTargetsEffects(left luau.ListOrNode) *EvaluationEffects {
	var targets []luau.Node
	if list, ok := left.(*luau.List); ok {
		targets = list.ToArray()
	} else {
		targets = []luau.Node{left.(luau.Node)}
	}
	// property writes can invoke __newindex or immediate Instance signal handlers
	effects := make([]*EvaluationEffects, len(targets))
	for i, target := range targets {
		if luau.IsAnyIdentifier(target) {
			effects[i] = withEffects(NO_EFFECTS, func(e *EvaluationEffects) {
				e.Writes = map[Binding]bool{getBinding(target): true}
			})
		} else {
			effects[i] = UNKNOWN_EFFECTS
		}
	}
	return joinEffects(effects...)
}

// forEachChild is TSTransformer/util/evaluation/effects.ts forEachChild. The TypeScript version walks
// the fields of the node object in creation order; this walks them in the luau-ast declaration order.
// Field order is sufficient for effect summaries, but not for evaluation planning.
func forEachChild(node luau.Node, callback func(luau.Node)) {
	visitNode := func(child luau.Node) {
		if child != nil {
			callback(child)
		}
	}
	visitList := func(list *luau.List) {
		if list != nil {
			list.ForEach(callback)
		}
	}
	visitListOrNode := func(value luau.ListOrNode) {
		switch v := value.(type) {
		case nil:
		case *luau.List:
			visitList(v)
		case luau.Node:
			visitNode(v)
		}
	}
	switch n := node.(type) {
	case *luau.ComputedIndexExpression:
		visitNode(n.Expression)
		visitNode(n.Index)
	case *luau.PropertyAccessExpression:
		visitNode(n.Expression)
	case *luau.CallExpression:
		visitNode(n.Expression)
		visitList(n.Args)
	case *luau.MethodCallExpression:
		visitNode(n.Expression)
		visitList(n.Args)
	case *luau.ParenthesizedExpression:
		visitNode(n.Expression)
	case *luau.FunctionExpression:
		visitList(n.Parameters)
		visitList(n.Statements)
	case *luau.BinaryExpression:
		visitNode(n.Left)
		visitNode(n.Right)
	case *luau.UnaryExpression:
		visitNode(n.Expression)
	case *luau.IfExpression:
		visitNode(n.Condition)
		visitNode(n.Expression)
		visitNode(n.Alternative)
	case *luau.InterpolatedString:
		visitList(n.Parts)
	case *luau.Array:
		visitList(n.Members)
	case *luau.Map:
		visitList(n.Fields)
	case *luau.Set:
		visitList(n.Members)
	case *luau.MixedTable:
		visitList(n.Fields)
	case *luau.Assignment:
		visitListOrNode(n.Left)
		visitListOrNode(n.Right)
	case *luau.CallStatement:
		visitNode(n.Expression)
	case *luau.DoStatement:
		visitList(n.Statements)
	case *luau.WhileStatement:
		visitNode(n.Condition)
		visitList(n.Statements)
	case *luau.RepeatStatement:
		visitList(n.Statements)
		visitNode(n.Condition)
	case *luau.IfStatement:
		visitNode(n.Condition)
		visitList(n.Statements)
		visitListOrNode(n.ElseBody)
	case *luau.NumericForStatement:
		visitNode(n.Id)
		visitNode(n.Start)
		visitNode(n.End)
		visitNode(n.Step)
		visitList(n.Statements)
	case *luau.ForStatement:
		visitList(n.Ids)
		visitNode(n.Expression)
		visitList(n.Statements)
	case *luau.FunctionDeclaration:
		visitNode(n.Name)
		visitList(n.Parameters)
		visitList(n.Statements)
	case *luau.MethodDeclaration:
		visitNode(n.Expression)
		visitList(n.Parameters)
		visitList(n.Statements)
	case *luau.VariableDeclaration:
		visitListOrNode(n.Left)
		visitListOrNode(n.Right)
	case *luau.ReturnStatement:
		visitListOrNode(n.Expression)
	case *luau.MapField:
		visitNode(n.Index)
		visitNode(n.Value)
	}
}

// getEffects is TSTransformer/util/evaluation/effects.ts getEffects for a node. getStatementsEffects is
// the same function for a list of statements.
func getEffects(node luau.Node) *EvaluationEffects {
	return getEffectsOfListOrNode(node)
}

// getStatementsEffects is `getEffects(statements)` for a luau.List: the TypeScript getEffects takes
// `luau.Node | luau.List<luau.Statement>`, the fixed Go signature takes a node.
func getStatementsEffects(statements *luau.List) *EvaluationEffects {
	return getEffectsOfListOrNode(statements)
}

func getEffectsOfListOrNode(value luau.ListOrNode) *EvaluationEffects {
	if list, ok := value.(*luau.List); ok {
		if list.IsEmpty() {
			return NO_EFFECTS
		}
	} else {
		node := value.(luau.Node)
		if isConstantReference(node) || luau.IsSimplePrimitive(node) || luau.IsNone(node) {
			return NO_EFFECTS
		}
		if luau.IsFunctionExpression(node) {
			return ALLOCATION
		}
	}
	// accumulate once per subtree instead of copying growing binding sets at every level
	result := &EvaluationEffects{Reads: map[Binding]bool{}, Writes: map[Binding]bool{}}
	var visit func(node luau.Node)
	visit = func(node luau.Node) {
		if isConstantReference(node) {
			return
		}
		// creating a closure does not execute its body.
		if luau.IsFunctionExpression(node) {
			result.Allocates = true
			return
		}
		effects := getIntrinsicEffects(node)
		for key := range effects.Reads {
			result.Reads[key] = true
		}
		for key := range effects.Writes {
			result.Writes[key] = true
		}
		result.ReadsHeap = result.ReadsHeap || effects.ReadsHeap
		result.WritesHeap = result.WritesHeap || effects.WritesHeap
		result.Throws = result.Throws || effects.Throws
		result.Allocates = result.Allocates || effects.Allocates
		forEachChild(node, visit)
	}
	if list, ok := value.(*luau.List); ok {
		list.ForEach(visit)
	} else {
		visit(value.(luau.Node))
	}
	return result
}

// isLateRead is TSTransformer/util/evaluation/effects.ts isLateRead: luau can use a local's register
// directly until the consuming instruction.
func isLateRead(expression luau.Expression) bool {
	for {
		parenthesized, ok := expression.(*luau.ParenthesizedExpression)
		if !ok {
			break
		}
		expression = parenthesized.Expression
	}
	return luau.IsAnyIdentifier(expression)
}
