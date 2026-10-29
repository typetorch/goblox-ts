package luau

import "strconv"

// Port of LuauRenderer/RenderState.ts.

const indentCharacter = "\t"

// RenderState is the state of a rendering process.
type RenderState struct {
	indent         string
	SeenTempNodes  map[int]string
	listNodesStack []*ListNode
	tempIdFallback int
}

// NewRenderState is `new RenderState()`.
func NewRenderState() *RenderState {
	return &RenderState{SeenTempNodes: map[int]string{}}
}

func (s *RenderState) pushIndent() { s.indent += indentCharacter }
func (s *RenderState) popIndent()  { s.indent = s.indent[len(indentCharacter):] }

// GetTempName returns an unique identifier that is unused in the current scope. SeenTempNodes should
// already be fully populated by this point; this is a fallback for when solveTempIds() does not catch
// something properly.
func (s *RenderState) GetTempName(node *TemporaryIdentifier) string {
	name, ok := s.SeenTempNodes[node.Id]
	if !ok {
		name = "_" + strconv.Itoa(s.tempIdFallback)
		s.tempIdFallback++
		s.SeenTempNodes[node.Id] = name
	}
	Assert(name != "")
	return name
}

// PushListNode pushes a statement list node to the top of the list node stack.
func (s *RenderState) PushListNode(listNode *ListNode) {
	s.listNodesStack = append(s.listNodesStack, listNode)
}

// PeekListNode returns the top of the list node stack, or nil.
func (s *RenderState) PeekListNode() *ListNode {
	if len(s.listNodesStack) == 0 {
		return nil
	}
	return s.listNodesStack[len(s.listNodesStack)-1]
}

// PopListNode pops the top list node off the stack.
func (s *RenderState) PopListNode() *ListNode {
	top := s.PeekListNode()
	if top != nil {
		s.listNodesStack = s.listNodesStack[:len(s.listNodesStack)-1]
	}
	return top
}

// Newline adds a newline to the end of the string.
func (s *RenderState) Newline(text string) string { return text + "\n" }

// Indented prefixes the text with the current indent.
func (s *RenderState) Indented(text string) string { return s.indent + text }

// Line renders a line, adding the current indent, a semicolon if necessary, and "\n". endNode is used
// to determine if a semicolon should be added; nil means no semicolon is added.
func (s *RenderState) Line(text string, endNode ...Statement) string {
	result := s.Indented(text)
	if len(endNode) > 0 && endNode[0] != nil {
		result += getEnding(s, endNode[0])
	}
	return s.Newline(result)
}

// Block renders callback one indent level deeper.
func (s *RenderState) Block(callback func() string) string {
	s.pushIndent()
	result := callback()
	s.popIndent()
	return result
}
