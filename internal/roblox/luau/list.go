package luau

// Port of LuauAST/impl/List.ts: a doubly linked list of nodes.

// ListNode is one element of a List.
type ListNode struct {
	Prev  *ListNode
	Next  *ListNode
	Value Node
}

// List is `luau.List<T>`.
type List struct {
	Head     *ListNode
	Tail     *ListNode
	Readonly bool
}

func (*List) isListOrNode() {}

// MakeListNode is `luau.list.makeNode`.
func MakeListNode(value Node) *ListNode {
	Assert(value != nil)
	return &ListNode{Value: value}
}

// MakeList is `luau.list.make`.
func MakeList(values ...Node) *List {
	list := &List{}
	if len(values) > 0 {
		head := MakeListNode(values[0])
		tail := head
		for i := 1; i < len(values); i++ {
			node := MakeListNode(values[i])
			tail.Next = node
			node.Prev = tail
			tail = node
		}
		list.Head = head
		list.Tail = tail
	}
	return list
}

// MakeListOf is MakeList for a slice of a concrete node type.
func MakeListOf[T Node](values []T) *List {
	nodes := make([]Node, len(values))
	for i, v := range values {
		nodes[i] = v
	}
	return MakeList(nodes...)
}

// IsList is `luau.list.isList`.
func IsList(value ListOrNode) bool {
	l, ok := value.(*List)
	return ok && l != nil
}

// Clone is `luau.list.clone`: a new list of shallow copies of the elements.
func (list *List) Clone() *List {
	newList := MakeList()
	list.ForEach(func(element Node) {
		newList.Push(cloneNode(element))
	})
	return newList
}

// Push is `luau.list.push`.
func (list *List) Push(value Node) {
	Assert(value != nil)
	Assert(!list.Readonly)
	node := MakeListNode(value)
	if list.Tail != nil {
		list.Tail.Next = node
		node.Prev = list.Tail
	} else {
		list.Head = node
	}
	list.Tail = node
}

// PushList is `luau.list.pushList`. other becomes read-only.
func (list *List) PushList(other *List) {
	Assert(!list.Readonly)
	Assert(!other.Readonly)
	other.Readonly = true

	if other.Head != nil && other.Tail != nil {
		if list.Head != nil && list.Tail != nil {
			list.Tail.Next = other.Head
			other.Head.Prev = list.Tail
			list.Tail = other.Tail
		} else {
			list.Head = other.Head
			list.Tail = other.Tail
		}
	}
}

// Shift is `luau.list.shift`. It returns nil for an empty list.
func (list *List) Shift() Node {
	Assert(!list.Readonly)
	if list.Head != nil {
		head := list.Head
		if head.Next != nil {
			list.Head = head.Next
			head.Next.Prev = nil
		} else {
			list.Tail = nil
			list.Head = nil
		}
		return head.Value
	}
	return nil
}

// Unshift is `luau.list.unshift`.
func (list *List) Unshift(value Node) {
	Assert(value != nil)
	Assert(!list.Readonly)
	node := MakeListNode(value)
	if list.Head != nil {
		list.Head.Prev = node
		node.Next = list.Head
	} else {
		list.Tail = node
	}
	list.Head = node
}

// UnshiftList is `luau.list.unshiftList`. other becomes read-only.
func (list *List) UnshiftList(other *List) {
	Assert(!list.Readonly)
	Assert(!other.Readonly)
	other.Readonly = true

	if other.Head != nil && other.Tail != nil {
		if list.Head != nil && list.Tail != nil {
			list.Head.Prev = other.Tail
			other.Tail.Next = list.Head
			list.Head = other.Head
		} else {
			list.Head = other.Head
			list.Tail = other.Tail
		}
	}
}

// IsEmpty is `luau.list.isEmpty`.
func (list *List) IsEmpty() bool { return list.Head == nil }

// IsNonEmpty is `luau.list.isNonEmpty`.
func (list *List) IsNonEmpty() bool { return list.Head != nil }

// ForEach is `luau.list.forEach`.
func (list *List) ForEach(callback func(value Node)) {
	for node := list.Head; node != nil; node = node.Next {
		callback(node.Value)
	}
}

// ForEachListNode is `luau.list.forEachListNode`.
func (list *List) ForEachListNode(callback func(value *ListNode)) {
	for node := list.Head; node != nil; node = node.Next {
		callback(node)
	}
}

// ListMapToArray is `luau.list.mapToArray`.
func ListMapToArray[U any](list *List, callback func(value Node) U) []U {
	var result []U
	list.ForEach(func(value Node) { result = append(result, callback(value)) })
	return result
}

// ToArray is `luau.list.toArray`.
func (list *List) ToArray() []Node {
	var result []Node
	list.ForEach(func(value Node) { result = append(result, value) })
	return result
}

// Every is `luau.list.every`.
func (list *List) Every(callback func(value Node) bool) bool {
	for node := list.Head; node != nil; node = node.Next {
		if !callback(node.Value) {
			return false
		}
	}
	return true
}

// Some is `luau.list.some`.
func (list *List) Some(callback func(value Node) bool) bool {
	for node := list.Head; node != nil; node = node.Next {
		if callback(node.Value) {
			return true
		}
	}
	return false
}

// Size is `luau.list.size`.
func (list *List) Size() int {
	size := 0
	for node := list.Head; node != nil; node = node.Next {
		size++
	}
	return size
}
