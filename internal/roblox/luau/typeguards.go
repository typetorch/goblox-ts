package luau

// Port of LuauAST/impl/typeGuards.ts.

func makeGuard(kinds ...SyntaxKind) func(node Node) bool {
	var set [KindLastField + 1]bool
	for _, k := range kinds {
		set[k] = true
	}
	return func(node Node) bool { return node != nil && set[node.Kind()] }
}

// indexable expressions
var (
	IsAnyIdentifier            = makeGuard(KindIdentifier, KindTemporaryIdentifier)
	IsIdentifier               = makeGuard(KindIdentifier)
	IsTemporaryIdentifier      = makeGuard(KindTemporaryIdentifier)
	IsComputedIndexExpression  = makeGuard(KindComputedIndexExpression)
	IsPropertyAccessExpression = makeGuard(KindPropertyAccessExpression)
	IsCallExpression           = makeGuard(KindCallExpression)
	IsMethodCallExpression     = makeGuard(KindMethodCallExpression)
	IsParenthesizedExpression  = makeGuard(KindParenthesizedExpression)
)

func IsIndexableExpression(node Node) bool {
	return node.Kind() >= KindFirstIndexableExpression && node.Kind() <= KindLastIndexableExpression
}

// expressions
var (
	IsNone               = makeGuard(KindNone)
	IsNilLiteral         = makeGuard(KindNilLiteral)
	IsFalseLiteral       = makeGuard(KindFalseLiteral)
	IsTrueLiteral        = makeGuard(KindTrueLiteral)
	IsNumberLiteral      = makeGuard(KindNumberLiteral)
	IsStringLiteral      = makeGuard(KindStringLiteral)
	IsVarArgsLiteral     = makeGuard(KindVarArgsLiteral)
	IsFunctionExpression = makeGuard(KindFunctionExpression)
	IsBinaryExpression   = makeGuard(KindBinaryExpression)
	IsUnaryExpression    = makeGuard(KindUnaryExpression)
	IsIfExpression       = makeGuard(KindIfExpression)
	IsInterpolatedString = makeGuard(KindInterpolatedString)
	IsArray              = makeGuard(KindArray)
	IsMap                = makeGuard(KindMap)
	IsSet                = makeGuard(KindSet)
	IsMixedTable         = makeGuard(KindMixedTable)
)

func IsExpression(node Node) bool {
	return node.Kind() >= KindFirstExpression && node.Kind() <= KindLastExpression
}

// statements
var (
	IsAssignment          = makeGuard(KindAssignment)
	IsBreakStatement      = makeGuard(KindBreakStatement)
	IsCallStatement       = makeGuard(KindCallStatement)
	IsContinueStatement   = makeGuard(KindContinueStatement)
	IsDoStatement         = makeGuard(KindDoStatement)
	IsWhileStatement      = makeGuard(KindWhileStatement)
	IsRepeatStatement     = makeGuard(KindRepeatStatement)
	IsIfStatement         = makeGuard(KindIfStatement)
	IsNumericForStatement = makeGuard(KindNumericForStatement)
	IsForStatement        = makeGuard(KindForStatement)
	IsFunctionDeclaration = makeGuard(KindFunctionDeclaration)
	IsMethodDeclaration   = makeGuard(KindMethodDeclaration)
	IsVariableDeclaration = makeGuard(KindVariableDeclaration)
	IsReturnStatement     = makeGuard(KindReturnStatement)
	IsComment             = makeGuard(KindComment)
)

func IsStatement(node Node) bool {
	return node.Kind() >= KindFirstStatement && node.Kind() <= KindLastStatement
}

// fields
var (
	IsMapField               = makeGuard(KindMapField)
	IsInterpolatedStringPart = makeGuard(KindInterpolatedStringPart)
)

func IsField(node Node) bool {
	return node.Kind() >= KindFirstField && node.Kind() <= KindLastField
}

// IsNode reports whether value is a Node (and not nil or a *List).
func IsNode(value ListOrNode) bool {
	n, ok := value.(Node)
	return ok && n != nil
}

var (
	IsSimple = makeGuard(
		KindIdentifier,
		KindTemporaryIdentifier,
		KindNilLiteral,
		KindTrueLiteral,
		KindFalseLiteral,
		KindNumberLiteral,
		KindStringLiteral,
	)

	IsSimplePrimitive = makeGuard(
		KindNilLiteral,
		KindTrueLiteral,
		KindFalseLiteral,
		KindNumberLiteral,
		KindStringLiteral,
	)

	IsTable = makeGuard(KindArray, KindSet, KindMap, KindMixedTable)

	IsFinalStatement = makeGuard(KindBreakStatement, KindReturnStatement, KindContinueStatement)

	IsCall = makeGuard(KindCallExpression, KindMethodCallExpression)

	IsWritableExpression = makeGuard(
		KindIdentifier,
		KindTemporaryIdentifier,
		KindPropertyAccessExpression,
		KindComputedIndexExpression,
	)

	IsFunctionLike = makeGuard(KindFunctionDeclaration, KindFunctionExpression, KindMethodDeclaration)

	HasStatements = makeGuard(
		KindForStatement,
		KindNumericForStatement,
		KindFunctionExpression,
		KindDoStatement,
		KindFunctionDeclaration,
		KindIfStatement,
		KindMethodDeclaration,
		KindRepeatStatement,
		KindWhileStatement,
	)

	IsExpressionWithPrecedence = makeGuard(KindIfExpression, KindUnaryExpression, KindBinaryExpression)
)

// GetStatements returns the statements of a node for which HasStatements is true.
func GetStatements(node Node) *List {
	switch n := node.(type) {
	case *ForStatement:
		return n.Statements
	case *NumericForStatement:
		return n.Statements
	case *FunctionExpression:
		return n.Statements
	case *DoStatement:
		return n.Statements
	case *FunctionDeclaration:
		return n.Statements
	case *IfStatement:
		return n.Statements
	case *MethodDeclaration:
		return n.Statements
	case *RepeatStatement:
		return n.Statements
	case *WhileStatement:
		return n.Statements
	}
	return nil
}

// GetParameters returns the parameters and hasDotDotDot of a node for which IsFunctionLike is true.
func GetParameters(node Node) (*List, bool) {
	switch n := node.(type) {
	case *FunctionExpression:
		return n.Parameters, n.HasDotDotDot
	case *FunctionDeclaration:
		return n.Parameters, n.HasDotDotDot
	case *MethodDeclaration:
		return n.Parameters, n.HasDotDotDot
	}
	return nil, false
}
