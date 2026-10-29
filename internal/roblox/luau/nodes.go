// Code generated from LuauAST/types/nodes.ts by a one-off script; edit by hand from now on.

package luau

// Identifier is a Luau indexable expression node.
type Identifier struct {
	NodeBase
	Name string
}

func (*Identifier) Kind() SyntaxKind     { return KindIdentifier }
func (n *Identifier) shallowClone() Node { c := *n; return &c }
func (n *Identifier) adoptChildren()     {}

// TemporaryIdentifier is a Luau indexable expression node.
type TemporaryIdentifier struct {
	NodeBase
	Name string
	Id   int
}

func (*TemporaryIdentifier) Kind() SyntaxKind     { return KindTemporaryIdentifier }
func (n *TemporaryIdentifier) shallowClone() Node { c := *n; return &c }
func (n *TemporaryIdentifier) adoptChildren()     {}

// ComputedIndexExpression is a Luau indexable expression node.
type ComputedIndexExpression struct {
	NodeBase
	Expression Node
	Index      Node
}

func (*ComputedIndexExpression) Kind() SyntaxKind     { return KindComputedIndexExpression }
func (n *ComputedIndexExpression) shallowClone() Node { c := *n; return &c }
func (n *ComputedIndexExpression) adoptChildren() {
	n.Expression = adopt(n, n.Expression)
	n.Index = adopt(n, n.Index)
}

// PropertyAccessExpression is a Luau indexable expression node.
type PropertyAccessExpression struct {
	NodeBase
	Expression Node
	Name       string
}

func (*PropertyAccessExpression) Kind() SyntaxKind     { return KindPropertyAccessExpression }
func (n *PropertyAccessExpression) shallowClone() Node { c := *n; return &c }
func (n *PropertyAccessExpression) adoptChildren() {
	n.Expression = adopt(n, n.Expression)
}

// CallExpression is a Luau indexable expression node.
type CallExpression struct {
	NodeBase
	Expression Node
	Args       *List
}

func (*CallExpression) Kind() SyntaxKind     { return KindCallExpression }
func (n *CallExpression) shallowClone() Node { c := *n; return &c }
func (n *CallExpression) adoptChildren() {
	n.Expression = adopt(n, n.Expression)
	adoptList(n, n.Args)
}

// MethodCallExpression is a Luau indexable expression node.
type MethodCallExpression struct {
	NodeBase
	Name       string
	Expression Node
	Args       *List
}

func (*MethodCallExpression) Kind() SyntaxKind     { return KindMethodCallExpression }
func (n *MethodCallExpression) shallowClone() Node { c := *n; return &c }
func (n *MethodCallExpression) adoptChildren() {
	n.Expression = adopt(n, n.Expression)
	adoptList(n, n.Args)
}

// ParenthesizedExpression is a Luau indexable expression node.
type ParenthesizedExpression struct {
	NodeBase
	Expression Node
}

func (*ParenthesizedExpression) Kind() SyntaxKind     { return KindParenthesizedExpression }
func (n *ParenthesizedExpression) shallowClone() Node { c := *n; return &c }
func (n *ParenthesizedExpression) adoptChildren() {
	n.Expression = adopt(n, n.Expression)
}

// None is a Luau expression node.
type None struct {
	NodeBase
}

func (*None) Kind() SyntaxKind     { return KindNone }
func (n *None) shallowClone() Node { c := *n; return &c }
func (n *None) adoptChildren()     {}

// NilLiteral is a Luau expression node.
type NilLiteral struct {
	NodeBase
}

func (*NilLiteral) Kind() SyntaxKind     { return KindNilLiteral }
func (n *NilLiteral) shallowClone() Node { c := *n; return &c }
func (n *NilLiteral) adoptChildren()     {}

// FalseLiteral is a Luau expression node.
type FalseLiteral struct {
	NodeBase
}

func (*FalseLiteral) Kind() SyntaxKind     { return KindFalseLiteral }
func (n *FalseLiteral) shallowClone() Node { c := *n; return &c }
func (n *FalseLiteral) adoptChildren()     {}

// TrueLiteral is a Luau expression node.
type TrueLiteral struct {
	NodeBase
}

func (*TrueLiteral) Kind() SyntaxKind     { return KindTrueLiteral }
func (n *TrueLiteral) shallowClone() Node { c := *n; return &c }
func (n *TrueLiteral) adoptChildren()     {}

// NumberLiteral is a Luau expression node.
type NumberLiteral struct {
	NodeBase
	Value string
}

func (*NumberLiteral) Kind() SyntaxKind     { return KindNumberLiteral }
func (n *NumberLiteral) shallowClone() Node { c := *n; return &c }
func (n *NumberLiteral) adoptChildren()     {}

// StringLiteral is a Luau expression node.
type StringLiteral struct {
	NodeBase
	Value string
	Quote string // empty (pick automatically), a double or a single quote; the value must already be escaped for it
}

func (*StringLiteral) Kind() SyntaxKind     { return KindStringLiteral }
func (n *StringLiteral) shallowClone() Node { c := *n; return &c }
func (n *StringLiteral) adoptChildren()     {}

// VarArgsLiteral is a Luau expression node.
type VarArgsLiteral struct {
	NodeBase
}

func (*VarArgsLiteral) Kind() SyntaxKind     { return KindVarArgsLiteral }
func (n *VarArgsLiteral) shallowClone() Node { c := *n; return &c }
func (n *VarArgsLiteral) adoptChildren()     {}

// FunctionExpression is a Luau expression node.
type FunctionExpression struct {
	NodeBase
	Parameters   *List
	HasDotDotDot bool
	Statements   *List
}

func (*FunctionExpression) Kind() SyntaxKind     { return KindFunctionExpression }
func (n *FunctionExpression) shallowClone() Node { c := *n; return &c }
func (n *FunctionExpression) adoptChildren() {
	adoptList(n, n.Parameters)
	adoptList(n, n.Statements)
}

// BinaryExpression is a Luau expression node.
type BinaryExpression struct {
	NodeBase
	Left     Node
	Operator BinaryOperator
	Right    Node
}

func (*BinaryExpression) Kind() SyntaxKind     { return KindBinaryExpression }
func (n *BinaryExpression) shallowClone() Node { c := *n; return &c }
func (n *BinaryExpression) adoptChildren() {
	n.Left = adopt(n, n.Left)
	n.Right = adopt(n, n.Right)
}

// UnaryExpression is a Luau expression node.
type UnaryExpression struct {
	NodeBase
	Operator   UnaryOperator
	Expression Node
}

func (*UnaryExpression) Kind() SyntaxKind     { return KindUnaryExpression }
func (n *UnaryExpression) shallowClone() Node { c := *n; return &c }
func (n *UnaryExpression) adoptChildren() {
	n.Expression = adopt(n, n.Expression)
}

// IfExpression is a Luau expression node.
type IfExpression struct {
	NodeBase
	Condition   Node
	Expression  Node
	Alternative Node
}

func (*IfExpression) Kind() SyntaxKind     { return KindIfExpression }
func (n *IfExpression) shallowClone() Node { c := *n; return &c }
func (n *IfExpression) adoptChildren() {
	n.Condition = adopt(n, n.Condition)
	n.Expression = adopt(n, n.Expression)
	n.Alternative = adopt(n, n.Alternative)
}

// InterpolatedString is a Luau expression node.
type InterpolatedString struct {
	NodeBase
	Parts *List // *InterpolatedStringPart or Expression
}

func (*InterpolatedString) Kind() SyntaxKind     { return KindInterpolatedString }
func (n *InterpolatedString) shallowClone() Node { c := *n; return &c }
func (n *InterpolatedString) adoptChildren() {
	adoptList(n, n.Parts)
}

// Array is a Luau expression node.
type Array struct {
	NodeBase
	Members *List
}

func (*Array) Kind() SyntaxKind     { return KindArray }
func (n *Array) shallowClone() Node { c := *n; return &c }
func (n *Array) adoptChildren() {
	adoptList(n, n.Members)
}

// Map is a Luau expression node.
type Map struct {
	NodeBase
	Fields *List // *MapField
}

func (*Map) Kind() SyntaxKind     { return KindMap }
func (n *Map) shallowClone() Node { c := *n; return &c }
func (n *Map) adoptChildren() {
	adoptList(n, n.Fields)
}

// Set is a Luau expression node.
type Set struct {
	NodeBase
	Members *List
}

func (*Set) Kind() SyntaxKind     { return KindSet }
func (n *Set) shallowClone() Node { c := *n; return &c }
func (n *Set) adoptChildren() {
	adoptList(n, n.Members)
}

// MixedTable is a Luau expression node.
type MixedTable struct {
	NodeBase
	Fields *List // *MapField or Expression
}

func (*MixedTable) Kind() SyntaxKind     { return KindMixedTable }
func (n *MixedTable) shallowClone() Node { c := *n; return &c }
func (n *MixedTable) adoptChildren() {
	adoptList(n, n.Fields)
}

// Assignment is a Luau statement node.
type Assignment struct {
	NodeBase
	Left     ListOrNode // WritableExpression or *List of WritableExpression
	Operator AssignmentOperator
	Right    ListOrNode // Expression or *List of Expression
}

func (*Assignment) Kind() SyntaxKind     { return KindAssignment }
func (n *Assignment) shallowClone() Node { c := *n; return &c }
func (n *Assignment) adoptChildren() {
	n.Left = adoptListOrNode(n, n.Left)
	n.Right = adoptListOrNode(n, n.Right)
}

// BreakStatement is a Luau statement node.
type BreakStatement struct {
	NodeBase
}

func (*BreakStatement) Kind() SyntaxKind     { return KindBreakStatement }
func (n *BreakStatement) shallowClone() Node { c := *n; return &c }
func (n *BreakStatement) adoptChildren()     {}

// CallStatement is a Luau statement node.
type CallStatement struct {
	NodeBase
	Expression Node
}

func (*CallStatement) Kind() SyntaxKind     { return KindCallStatement }
func (n *CallStatement) shallowClone() Node { c := *n; return &c }
func (n *CallStatement) adoptChildren() {
	n.Expression = adopt(n, n.Expression)
}

// ContinueStatement is a Luau statement node.
type ContinueStatement struct {
	NodeBase
}

func (*ContinueStatement) Kind() SyntaxKind     { return KindContinueStatement }
func (n *ContinueStatement) shallowClone() Node { c := *n; return &c }
func (n *ContinueStatement) adoptChildren()     {}

// DoStatement is a Luau statement node.
type DoStatement struct {
	NodeBase
	Statements *List
}

func (*DoStatement) Kind() SyntaxKind     { return KindDoStatement }
func (n *DoStatement) shallowClone() Node { c := *n; return &c }
func (n *DoStatement) adoptChildren() {
	adoptList(n, n.Statements)
}

// WhileStatement is a Luau statement node.
type WhileStatement struct {
	NodeBase
	Condition  Node
	Statements *List
}

func (*WhileStatement) Kind() SyntaxKind     { return KindWhileStatement }
func (n *WhileStatement) shallowClone() Node { c := *n; return &c }
func (n *WhileStatement) adoptChildren() {
	n.Condition = adopt(n, n.Condition)
	adoptList(n, n.Statements)
}

// RepeatStatement is a Luau statement node.
type RepeatStatement struct {
	NodeBase
	Condition  Node
	Statements *List
}

func (*RepeatStatement) Kind() SyntaxKind     { return KindRepeatStatement }
func (n *RepeatStatement) shallowClone() Node { c := *n; return &c }
func (n *RepeatStatement) adoptChildren() {
	n.Condition = adopt(n, n.Condition)
	adoptList(n, n.Statements)
}

// IfStatement is a Luau statement node.
type IfStatement struct {
	NodeBase
	Condition  Node
	Statements *List
	ElseBody   ListOrNode // *IfStatement or *List of Statement
}

func (*IfStatement) Kind() SyntaxKind     { return KindIfStatement }
func (n *IfStatement) shallowClone() Node { c := *n; return &c }
func (n *IfStatement) adoptChildren() {
	n.Condition = adopt(n, n.Condition)
	adoptList(n, n.Statements)
	n.ElseBody = adoptListOrNode(n, n.ElseBody)
}

// NumericForStatement is a Luau statement node.
type NumericForStatement struct {
	NodeBase
	Id         Node
	Start      Node
	End        Node
	Step       Node // optional
	Statements *List
}

func (*NumericForStatement) Kind() SyntaxKind     { return KindNumericForStatement }
func (n *NumericForStatement) shallowClone() Node { c := *n; return &c }
func (n *NumericForStatement) adoptChildren() {
	n.Id = adopt(n, n.Id)
	n.Start = adopt(n, n.Start)
	n.End = adopt(n, n.End)
	n.Step = adopt(n, n.Step)
	adoptList(n, n.Statements)
}

// ForStatement is a Luau statement node.
type ForStatement struct {
	NodeBase
	Ids        *List
	Expression Node
	Statements *List
}

func (*ForStatement) Kind() SyntaxKind     { return KindForStatement }
func (n *ForStatement) shallowClone() Node { c := *n; return &c }
func (n *ForStatement) adoptChildren() {
	adoptList(n, n.Ids)
	n.Expression = adopt(n, n.Expression)
	adoptList(n, n.Statements)
}

// FunctionDeclaration is a Luau statement node.
type FunctionDeclaration struct {
	NodeBase
	Localize     bool
	Name         Node // AnyIdentifier or *PropertyAccessExpression
	Parameters   *List
	HasDotDotDot bool
	Statements   *List
}

func (*FunctionDeclaration) Kind() SyntaxKind     { return KindFunctionDeclaration }
func (n *FunctionDeclaration) shallowClone() Node { c := *n; return &c }
func (n *FunctionDeclaration) adoptChildren() {
	n.Name = adopt(n, n.Name)
	adoptList(n, n.Parameters)
	adoptList(n, n.Statements)
}

// MethodDeclaration is a Luau statement node.
type MethodDeclaration struct {
	NodeBase
	Expression   Node
	Name         string
	Parameters   *List
	HasDotDotDot bool
	Statements   *List
}

func (*MethodDeclaration) Kind() SyntaxKind     { return KindMethodDeclaration }
func (n *MethodDeclaration) shallowClone() Node { c := *n; return &c }
func (n *MethodDeclaration) adoptChildren() {
	n.Expression = adopt(n, n.Expression)
	adoptList(n, n.Parameters)
	adoptList(n, n.Statements)
}

// VariableDeclaration is a Luau statement node.
type VariableDeclaration struct {
	NodeBase
	Left  ListOrNode // AnyIdentifier or *List of AnyIdentifier
	Right ListOrNode // Expression, *List of Expression, or nil
}

func (*VariableDeclaration) Kind() SyntaxKind     { return KindVariableDeclaration }
func (n *VariableDeclaration) shallowClone() Node { c := *n; return &c }
func (n *VariableDeclaration) adoptChildren() {
	n.Left = adoptListOrNode(n, n.Left)
	n.Right = adoptListOrNode(n, n.Right)
}

// ReturnStatement is a Luau statement node.
type ReturnStatement struct {
	NodeBase
	Expression ListOrNode // Expression or *List of Expression
}

func (*ReturnStatement) Kind() SyntaxKind     { return KindReturnStatement }
func (n *ReturnStatement) shallowClone() Node { c := *n; return &c }
func (n *ReturnStatement) adoptChildren() {
	n.Expression = adoptListOrNode(n, n.Expression)
}

// Comment is a Luau statement node.
type Comment struct {
	NodeBase
	Text string
}

func (*Comment) Kind() SyntaxKind     { return KindComment }
func (n *Comment) shallowClone() Node { c := *n; return &c }
func (n *Comment) adoptChildren()     {}

// MapField is a Luau field node.
type MapField struct {
	NodeBase
	Index Node
	Value Node
}

func (*MapField) Kind() SyntaxKind     { return KindMapField }
func (n *MapField) shallowClone() Node { c := *n; return &c }
func (n *MapField) adoptChildren() {
	n.Index = adopt(n, n.Index)
	n.Value = adopt(n, n.Value)
}

// InterpolatedStringPart is a Luau field node.
type InterpolatedStringPart struct {
	NodeBase
	Text string
}

func (*InterpolatedStringPart) Kind() SyntaxKind     { return KindInterpolatedStringPart }
func (n *InterpolatedStringPart) shallowClone() Node { c := *n; return &c }
func (n *InterpolatedStringPart) adoptChildren()     {}
