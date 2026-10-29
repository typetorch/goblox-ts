package luau

// SyntaxKind is the kind of a Luau AST node. Port of LuauAST/impl/enums.ts; the order of the
// constants matters (the First*/Last* ranges classify kinds).
type SyntaxKind int

const (
	// indexable expressions
	KindIdentifier SyntaxKind = iota
	KindTemporaryIdentifier
	KindComputedIndexExpression
	KindPropertyAccessExpression
	KindCallExpression
	KindMethodCallExpression
	KindParenthesizedExpression

	// expressions
	KindNone
	KindNilLiteral
	KindFalseLiteral
	KindTrueLiteral
	KindNumberLiteral
	KindStringLiteral
	KindVarArgsLiteral
	KindFunctionExpression
	KindBinaryExpression
	KindUnaryExpression
	KindIfExpression
	KindInterpolatedString
	KindArray
	KindMap
	KindSet
	KindMixedTable

	// statements
	KindAssignment
	KindBreakStatement
	KindCallStatement
	KindContinueStatement
	KindDoStatement
	KindWhileStatement
	KindRepeatStatement
	KindIfStatement
	KindNumericForStatement
	KindForStatement
	KindFunctionDeclaration
	KindMethodDeclaration
	KindVariableDeclaration
	KindReturnStatement
	KindComment

	// fields
	KindMapField
	KindInterpolatedStringPart

	// used to detect what category a given kind falls into
	KindFirstIndexableExpression = KindIdentifier
	KindLastIndexableExpression  = KindParenthesizedExpression
	KindFirstExpression          = KindIdentifier
	KindLastExpression           = KindMixedTable
	KindFirstStatement           = KindAssignment
	KindLastStatement            = KindComment
	KindFirstField               = KindMapField
	KindLastField                = KindInterpolatedStringPart
)

var kindNames = [...]string{
	"Identifier",
	"TemporaryIdentifier",
	"ComputedIndexExpression",
	"PropertyAccessExpression",
	"CallExpression",
	"MethodCallExpression",
	"ParenthesizedExpression",
	"None",
	"NilLiteral",
	"FalseLiteral",
	"TrueLiteral",
	"NumberLiteral",
	"StringLiteral",
	"VarArgsLiteral",
	"FunctionExpression",
	"BinaryExpression",
	"UnaryExpression",
	"IfExpression",
	"InterpolatedString",
	"Array",
	"Map",
	"Set",
	"MixedTable",
	"Assignment",
	"BreakStatement",
	"CallStatement",
	"ContinueStatement",
	"DoStatement",
	"WhileStatement",
	"RepeatStatement",
	"IfStatement",
	"NumericForStatement",
	"ForStatement",
	"FunctionDeclaration",
	"MethodDeclaration",
	"VariableDeclaration",
	"ReturnStatement",
	"Comment",
	"MapField",
	"InterpolatedStringPart",
}

// GetKindName is LuauAST/util/getKindName.ts.
func GetKindName(kind SyntaxKind) string {
	if kind >= 0 && int(kind) < len(kindNames) {
		return kindNames[kind]
	}
	return "Unknown"
}

func (k SyntaxKind) String() string { return GetKindName(k) }
