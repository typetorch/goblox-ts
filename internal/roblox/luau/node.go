// Package luau is a Go port of @roblox-ts/luau-ast 2.1.2: the Luau AST, its renderer and solveTempIds.
//
// Naming follows the TypeScript sources. `luau.SyntaxKind.X` is KindX, the node interfaces are Go
// structs of the same name, `luau.create(kind, fields)` is Create(&X{...}), `luau.list.*` are methods on
// *List, and the helper constructors (`luau.id`, `luau.call`, ...) are Id, Call, ...
//
// As in the TypeScript version, a node has at most one parent: Create adopts the given children and
// replaces any child that already has a parent with a shallow clone.
package luau

// ListOrNode is a Node or a *List. It types the fields that are a union of a node and a list in the
// TypeScript AST (Assignment.Left, VariableDeclaration.Right, IfStatement.ElseBody, ...).
type ListOrNode interface {
	isListOrNode()
}

// Node is any Luau AST node.
type Node interface {
	ListOrNode
	Kind() SyntaxKind
	Parent() Node
	setParent(parent Node)
	shallowClone() Node
	adoptChildren()
}

// Aliases for the TypeScript category types. They are all Node: the category is checked at runtime
// with IsExpression, IsStatement, ... exactly like the TypeScript type guards.
type (
	Expression          = Node
	IndexableExpression = Node
	Statement           = Node
	Field               = Node
	AnyIdentifier       = Node
	WritableExpression  = Node
)

// NodeBase is embedded in every node.
type NodeBase struct {
	parent Node
}

func (b *NodeBase) Parent() Node          { return b.parent }
func (b *NodeBase) setParent(parent Node) { b.parent = parent }
func (b *NodeBase) isListOrNode()         {}

// Create is `luau.create(kind, fields)`: it sets the parent of every child of node to node, cloning
// children that already have a parent, and returns node.
func Create[T Node](node T) T {
	node.adoptChildren()
	return node
}

func adopt(parent Node, child Node) Node {
	if child == nil {
		return nil
	}
	if child.Parent() != nil {
		clone := child.shallowClone()
		clone.setParent(parent)
		return clone
	}
	child.setParent(parent)
	return child
}

func adoptList(parent Node, list *List) {
	if list == nil {
		return
	}
	for listNode := list.Head; listNode != nil; listNode = listNode.Next {
		listNode.Value = adopt(parent, listNode.Value)
	}
}

func adoptListOrNode(parent Node, value ListOrNode) ListOrNode {
	switch v := value.(type) {
	case nil:
		return nil
	case *List:
		adoptList(parent, v)
		return v
	case Node:
		return adopt(parent, v)
	}
	panic("unreachable")
}

// sharedParent is the parent of the package-level nodes in Globals and Strings. Because they already
// have a parent, Create always clones them instead of adopting (and mutating) the shared node, which
// keeps them safe to use from concurrent transforms.
var sharedParent Node = &None{}

func shared[T Node](node T) T {
	node.setParent(sharedParent)
	return node
}
