package luau

// Port of LuauRenderer/solveTempIds.ts.

func isFullyScopedNode(node Node) bool {
	return IsForStatement(node) || IsNumericForStatement(node) || IsFunctionLike(node)
}

func isScopeEdge(node Node, head bool) bool {
	edge := func(list *List) *ListNode {
		if head {
			return list.Head
		}
		return list.Tail
	}
	if parent := node.Parent(); parent != nil {
		// is the first statement in a block that creates scope
		if HasStatements(parent) {
			if e := edge(GetStatements(parent)); e != nil && node == e.Value {
				return true
			}
		}

		// non-list elseBody would have the elseBody itself as a parent,
		// which would be a luau.IfStatement and handled above
		if ifStatement, ok := parent.(*IfStatement); ok {
			if elseBody, ok := ifStatement.ElseBody.(*List); ok {
				if e := edge(elseBody); e != nil && node == e.Value {
					return true
				}
			}
		}
	}
	return false
}

func isScopeStart(node Node) bool { return isScopeEdge(node, true) }
func isScopeEnd(node Node) bool   { return isScopeEdge(node, false) }

type scope struct {
	ids     map[string]bool
	lastTry map[string]int
	parent  *scope
}

func createScope(parent *scope) *scope {
	return &scope{ids: map[string]bool{}, lastTry: map[string]int{}, parent: parent}
}

func scopeHasId(s *scope, id string) bool {
	for ; s != nil; s = s.parent {
		if s.ids[id] {
			return true
		}
	}
	return false
}

// SolveTempIds assigns a name to every TemporaryIdentifier in ast (a *List or a Node) that does not
// collide with the identifiers declared in its scope, and records it in state.SeenTempNodes.
func SolveTempIds(state *RenderState, ast ListOrNode) {
	var tempIdsToProcess []*TemporaryIdentifier
	nodesToScopes := map[*TemporaryIdentifier]*scope{}

	scopeStack := []*scope{createScope(nil)}

	peekScopeStack := func() *scope {
		Assert(len(scopeStack) > 0)
		return scopeStack[len(scopeStack)-1]
	}
	pushScopeStack := func() {
		scopeStack = append(scopeStack, createScope(peekScopeStack()))
	}
	popScopeStack := func() {
		if len(scopeStack) > 0 {
			scopeStack = scopeStack[:len(scopeStack)-1]
		}
	}
	registerId := func(name string) {
		peekScopeStack().ids[name] = true
	}

	Visit(ast, &Visitor{
		Before: func(node Node) {
			if isFullyScopedNode(node) {
				pushScopeStack()
			}
			if isScopeStart(node) {
				pushScopeStack()
			}

			switch n := node.(type) {
			case *TemporaryIdentifier:
				nodesToScopes[n] = peekScopeStack()
				tempIdsToProcess = append(tempIdsToProcess, n)
			case *VariableDeclaration:
				if l, ok := n.Left.(*List); ok {
					l.ForEach(func(node Node) {
						if id, ok := node.(*Identifier); ok {
							registerId(id.Name)
						}
					})
				} else if id, ok := n.Left.(*Identifier); ok {
					registerId(id.Name)
				}
			default:
				if IsFunctionLike(node) {
					parameters, _ := GetParameters(node)
					parameters.ForEach(func(node Node) {
						if id, ok := node.(*Identifier); ok {
							registerId(id.Name)
						}
					})
				}
			}
		},
		After: func(node Node) {
			if isFullyScopedNode(node) {
				popScopeStack()
			}
			if isScopeEnd(node) {
				popScopeStack()
			}
		},
	})

	for _, tempId := range tempIdsToProcess {
		if _, seen := state.SeenTempNodes[tempId.Id]; !seen {
			s := nodesToScopes[tempId]
			Assert(s != nil)

			separator := "_"
			if tempId.Name == "" {
				separator = ""
			}

			input := "_" + tempId.Name
			i, ok := s.lastTry[input]
			if !ok {
				i = 1
			}
			for scopeHasId(s, input) {
				input = "_" + tempId.Name + separator + itoa(i)
				i++
			}
			s.lastTry[input] = i
			s.ids[input] = true

			state.SeenTempNodes[tempId.Id] = input
		}
	}
}
