// Package gocql2 parses OGC CQL2 filters.
//
//nolint:revive // AST names are self-explanatory; avoid repetitive doc noise.
package gocql2

// Language identifies a concrete CQL2 encoding.
type Language string

// Supported CQL2 language identifiers.
const (
	LanguageText Language = "cql2-text"
	LanguageJSON Language = "cql2-json"
)

// Node is implemented by every parsed AST node.
type Node interface {
	Span() Span
}

// Expression is a CQL2 boolean/filter expression.
type Expression interface {
	Node
	expressionNode()
}

// ScalarExpression is a scalar-valued CQL2 expression.
type ScalarExpression interface {
	Node
	scalarNode()
}

// Span records where an AST node came from. Text parses populate byte/character
// offsets and line/column values; JSON semantic parses populate JSON paths.
type Span struct {
	Start Location
	End   Location
}

// LogicalOp is a boolean operation.
type LogicalOp string

// Supported logical operators.
const (
	LogicalAnd LogicalOp = "and"
	LogicalOr  LogicalOp = "or"
	LogicalNot LogicalOp = "not"
)

// LogicalExpression represents AND, OR, and NOT.
type LogicalExpression struct {
	Op   LogicalOp
	Args []Expression
	Src  Span
}

func (*LogicalExpression) expressionNode() {}

func (e *LogicalExpression) Span() Span { return e.Src }

// ComparisonOp is a binary comparison operation.
type ComparisonOp string

// Supported comparison operators.
const (
	OpEqual              ComparisonOp = "="
	OpNotEqual           ComparisonOp = "<>"
	OpLessThan           ComparisonOp = "<"
	OpGreaterThan        ComparisonOp = ">"
	OpLessThanOrEqual    ComparisonOp = "<="
	OpGreaterThanOrEqual ComparisonOp = ">="
)

// ComparisonExpression compares two scalar expressions.
type ComparisonExpression struct {
	Op    ComparisonOp
	Left  ScalarExpression
	Right ScalarExpression
	Src   Span
}

func (*ComparisonExpression) expressionNode() {}

func (e *ComparisonExpression) Span() Span { return e.Src }

// ArithmeticOp is a numeric arithmetic operation.
type ArithmeticOp string

// Supported arithmetic operators.
const (
	ArithmeticAdd    ArithmeticOp = "+"
	ArithmeticSub    ArithmeticOp = "-"
	ArithmeticMul    ArithmeticOp = "*"
	ArithmeticDiv    ArithmeticOp = "/"
	ArithmeticPow    ArithmeticOp = "^"
	ArithmeticMod    ArithmeticOp = "%"
	ArithmeticIntDiv ArithmeticOp = "div"
)

// ArithmeticExpression combines two numeric scalar expressions.
type ArithmeticExpression struct {
	Op    ArithmeticOp
	Left  ScalarExpression
	Right ScalarExpression
	Src   Span
}

func (*ArithmeticExpression) scalarNode() {}

func (e *ArithmeticExpression) Span() Span { return e.Src }

// LikeExpression represents a LIKE predicate. Pattern is a string literal,
// optionally wrapped by CQL2 text functions such as casei or accenti.
type LikeExpression struct {
	Expr    ScalarExpression
	Pattern ScalarExpression
	Src     Span
	Not     bool
}

func (*LikeExpression) expressionNode() {}

func (e *LikeExpression) Span() Span { return e.Src }

// BetweenExpression represents a BETWEEN predicate.
type BetweenExpression struct {
	Expr  ScalarExpression
	Lower ScalarExpression
	Upper ScalarExpression
	Src   Span
	Not   bool
}

func (*BetweenExpression) expressionNode() {}

func (e *BetweenExpression) Span() Span { return e.Src }

// InExpression represents an IN-list predicate.
//
//nolint:govet // Public field order is grouped by AST meaning rather than fieldalignment.
type InExpression struct {
	Expr   ScalarExpression
	Src    Span
	Values []ScalarExpression
	Not    bool
}

func (*InExpression) expressionNode() {}

func (e *InExpression) Span() Span { return e.Src }

// IsNullExpression represents an IS NULL predicate.
type IsNullExpression struct {
	Expr Node
	Src  Span
	Not  bool
}

func (*IsNullExpression) expressionNode() {}

func (e *IsNullExpression) Span() Span { return e.Src }

// PropertyRef references a feature property.
type PropertyRef struct {
	Name string
	Type PropertyType
	Src  Span
}

func (*PropertyRef) scalarNode() {}

func (p *PropertyRef) Span() Span { return p.Src }

// LiteralKind identifies a literal value kind.
type LiteralKind string

// Supported literal kinds.
const (
	LiteralString LiteralKind = "string"
	LiteralNumber LiteralKind = "number"
	LiteralBool   LiteralKind = "bool"
	LiteralNull   LiteralKind = "null"
)

// Literal is a parsed scalar literal. Numeric literals use a canonical decimal
// string so text and JSON inputs produce aligned values without float rounding.
type Literal struct {
	Kind  LiteralKind
	Value any
	Src   Span
}

func (*Literal) expressionNode() {}
func (*Literal) scalarNode()     {}

func (l *Literal) Span() Span { return l.Src }

// FunctionCall represents a CQL2 function reference. It may appear as either a
// scalar expression or a boolean expression, depending on the function.
type FunctionCall struct {
	Name        string
	Args        []Node
	ReturnTypes []FunctionType
	Src         Span
}

func (*FunctionCall) expressionNode() {}
func (*FunctionCall) scalarNode()     {}

func (f *FunctionCall) Span() Span { return f.Src }

// ArrayLiteral represents a JSON/text array/list value used in function args or
// IN-list internals.
type ArrayLiteral struct {
	Values []Node
	Src    Span
}

func (*ArrayLiteral) scalarNode() {}

func (a *ArrayLiteral) Span() Span { return a.Src }
