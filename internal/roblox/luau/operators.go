package luau

// Port of LuauAST/types/operators.ts.

// BinaryOperator is one of "+" "-" "*" "/" "//" "^" "%" ".." "<" "<=" ">" ">=" "==" "~=" "and" "or".
type BinaryOperator = string

// UnaryOperator is one of "-" "not" "#".
type UnaryOperator = string

// AssignmentOperator is one of "=" "+=" "-=" "*=" "/=" "//=" "%=" "^=" "..=".
type AssignmentOperator = string
