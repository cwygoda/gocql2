package gocql2

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// SQL is a parameterized SQL fragment generated from a CQL2 expression.
type SQL struct {
	Text string
	Args []any
}

// SQLFragment is a SQL fragment without independently-owned arguments. Dialect
// renderers receive a context that owns argument numbering.
type SQLFragment struct {
	Text string
}

// SQLExpr is a trusted application-authored SQL expression used to map a CQL2
// property/queryable name to SQL.
type SQLExpr interface{ sqlExpr() }

type sqlColumnExpr struct {
	Parts []string
}

type sqlRawExpr struct {
	SQL string
}

func (sqlColumnExpr) sqlExpr() {}
func (sqlRawExpr) sqlExpr()    {}

// Column maps a CQL2 property to a dialect-quoted SQL identifier. Multiple
// parts are rendered as a qualified identifier, e.g. Column("t", "name").
func Column(parts ...string) SQLExpr { return sqlColumnExpr{Parts: append([]string(nil), parts...)} }

// RawSQL maps a CQL2 property to trusted application-authored SQL. Never build
// RawSQL from untrusted CQL2 input.
func RawSQL(sql string) SQLExpr { return sqlRawExpr{SQL: sql} }

// SQLProperty maps a CQL2 queryable/property name to a SQL expression.
//
//nolint:govet // Public field order follows API readability: CQL2 name/type, then SQL expression.
type SQLProperty struct {
	Name string
	Type PropertyType
	Expr SQLExpr
}

// ErrNoSQLMapping reports that a resolver has no SQL mapping for a CQL2 property.
var ErrNoSQLMapping = errors.New("no SQL mapping for CQL2 property")

// PropertyResolver resolves CQL2 property references during SQL generation.
type PropertyResolver interface {
	ResolveSQLProperty(ref *PropertyRef) (SQLExpr, error)
}

// SQLDialect renders SQL constructs that vary by database.
type SQLDialect interface {
	Placeholder(index int) string
	QuoteIdentifier(name string) (string, error)
	RenderNumberLiteral(ctx SQLRenderContext, value string) (SQLFragment, error)
	RenderTemporalInstant(ctx SQLRenderContext, value *TemporalInstant) (SQLFragment, error)
	RenderFunction(ctx SQLRenderContext, call *FunctionCall) (SQLFragment, bool, error)
	RenderSpatialPredicate(ctx SQLRenderContext, expr *SpatialPredicateExpression) (SQLFragment, bool, error)
	RenderTemporalPredicate(ctx SQLRenderContext, expr *TemporalPredicateExpression) (SQLFragment, bool, error)
	RenderArrayPredicate(ctx SQLRenderContext, expr *ArrayPredicateExpression) (SQLFragment, bool, error)
	RenderGeometryLiteral(ctx SQLRenderContext, lit *GeometryLiteral) (SQLFragment, bool, error)
}

// SQLRenderContext is passed to dialect hooks for recursive rendering and bind
// argument allocation.
type SQLRenderContext interface {
	Render(node Node) (SQLFragment, error)
	RenderScalar(expr ScalarExpression) (SQLFragment, error)
	RenderTemporalOperand(node Node) (SQLTemporalOperand, error)
	RenderTemporalStart(node Node) (SQLFragment, error)
	RenderTemporalEnd(node Node) (SQLFragment, error)
	AddArg(value any) SQLFragment
}

// SQLTemporalOperand represents an instant as Start == End and an interval as
// separate start/end SQL expressions.
type SQLTemporalOperand struct {
	Start SQLFragment
	End   SQLFragment
}

// BaseDialect provides conservative defaults for custom dialects.
type BaseDialect struct{}

// Placeholder renders a positional placeholder. BaseDialect uses question marks.
func (BaseDialect) Placeholder(int) string { return "?" }

// QuoteIdentifier quotes one SQL identifier part using ANSI double quotes.
func (BaseDialect) QuoteIdentifier(name string) (string, error) {
	if name == "" || strings.ContainsRune(name, '\x00') {
		return "", fmt.Errorf("invalid SQL identifier %q", name)
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`, nil
}

// RenderNumberLiteral renders a numeric literal from its exact CQL2 text.
func (BaseDialect) RenderNumberLiteral(ctx SQLRenderContext, value string) (SQLFragment, error) {
	return SQLFragment{Text: "CAST(" + ctx.AddArg(value).Text + " AS NUMERIC)"}, nil
}

// RenderTemporalInstant renders a temporal instant literal.
func (BaseDialect) RenderTemporalInstant(ctx SQLRenderContext, value *TemporalInstant) (SQLFragment, error) {
	arg := ctx.AddArg(value.Value)
	if value.Kind == TemporalInstantDate {
		return SQLFragment{Text: "CAST(" + arg.Text + " AS DATE)"}, nil
	}
	return SQLFragment{Text: "CAST(" + arg.Text + " AS TIMESTAMP)"}, nil
}

// RenderFunction declines all function rendering.
func (BaseDialect) RenderFunction(SQLRenderContext, *FunctionCall) (SQLFragment, bool, error) {
	return SQLFragment{}, false, nil
}

// RenderSpatialPredicate declines all spatial predicate rendering.
func (BaseDialect) RenderSpatialPredicate(SQLRenderContext, *SpatialPredicateExpression) (SQLFragment, bool, error) {
	return SQLFragment{}, false, nil
}

// RenderTemporalPredicate declines all temporal predicate rendering.
func (BaseDialect) RenderTemporalPredicate(SQLRenderContext, *TemporalPredicateExpression) (SQLFragment, bool, error) {
	return SQLFragment{}, false, nil
}

// RenderArrayPredicate declines all array predicate rendering.
func (BaseDialect) RenderArrayPredicate(SQLRenderContext, *ArrayPredicateExpression) (SQLFragment, bool, error) {
	return SQLFragment{}, false, nil
}

// RenderGeometryLiteral declines all geometry literal rendering.
func (BaseDialect) RenderGeometryLiteral(SQLRenderContext, *GeometryLiteral) (SQLFragment, bool, error) {
	return SQLFragment{}, false, nil
}

// SQLOption configures SQL generation.
type SQLOption func(*sqlConfig)

type sqlConfig struct {
	resolver             PropertyResolver
	defaultColumnMapping bool
}

// WithSQLProperties configures explicit fail-closed CQL2 property-to-SQL mappings.
func WithSQLProperties(props ...SQLProperty) SQLOption {
	return func(cfg *sqlConfig) {
		cfg.resolver = sqlPropertyMap(props)
	}
}

// WithPropertyResolver configures a custom property resolver.
func WithPropertyResolver(resolver PropertyResolver) SQLOption {
	return func(cfg *sqlConfig) { cfg.resolver = resolver }
}

// WithDefaultColumnMapping allows unmapped CQL2 properties to map to quoted SQL
// columns of the same name. Without this or an explicit resolver, SQL generation
// fails closed for properties.
func WithDefaultColumnMapping() SQLOption {
	return func(cfg *sqlConfig) { cfg.defaultColumnMapping = true }
}

type sqlPropertyMap []SQLProperty

func (m sqlPropertyMap) ResolveSQLProperty(ref *PropertyRef) (SQLExpr, error) {
	for _, prop := range m {
		if prop.Name == ref.Name {
			if prop.Expr == nil {
				return nil, fmt.Errorf("SQL property %q has no expression", prop.Name)
			}
			return prop.Expr, nil
		}
	}
	return nil, fmt.Errorf("%w %q", ErrNoSQLMapping, ref.Name)
}

// SQLPropertyDefinitions returns parse-time property definitions matching SQL
// property mappings, avoiding duplicate schema declarations for common callers.
func SQLPropertyDefinitions(props ...SQLProperty) []PropertyDefinition {
	defs := make([]PropertyDefinition, 0, len(props))
	for _, prop := range props {
		if prop.Name != "" {
			defs = append(defs, PropertyDefinition{Name: prop.Name, Type: prop.Type})
		}
	}
	return defs
}

// ToSQL compiles a parsed CQL2 expression to a parameterized SQL fragment.
func ToSQL(expr Expression, dialect SQLDialect, opts ...SQLOption) (SQL, error) {
	if expr == nil {
		return SQL{}, fmt.Errorf("cannot render nil CQL2 expression")
	}
	if dialect == nil {
		dialect = BaseDialect{}
	}
	cfg := sqlConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	compiler := &sqlCompiler{dialect: dialect, cfg: cfg}
	frag, err := compiler.renderExpression(expr)
	if err != nil {
		return SQL{}, err
	}
	return SQL{Text: frag.Text, Args: append([]any(nil), compiler.args...)}, nil
}

type sqlCompiler struct {
	dialect SQLDialect
	cfg     sqlConfig
	args    []any
}

func (c *sqlCompiler) Render(node Node) (SQLFragment, error) {
	switch value := node.(type) {
	case Expression:
		return c.renderExpression(value)
	case ScalarExpression:
		return c.renderScalar(value)
	case *GeometryLiteral:
		return c.renderGeometryLiteral(value)
	case *TemporalInterval:
		operand, err := c.RenderTemporalOperand(value)
		if err != nil {
			return SQLFragment{}, err
		}
		return SQLFragment{Text: fmt.Sprintf("(%s, %s)", operand.Start.Text, operand.End.Text)}, nil
	case *TemporalUnbounded:
		return SQLFragment{}, fmt.Errorf("temporal unbounded endpoint cannot be rendered as a standalone SQL expression")
	default:
		return SQLFragment{}, fmt.Errorf("unsupported CQL2 node %T for SQL generation", node)
	}
}

func (c *sqlCompiler) RenderScalar(expr ScalarExpression) (SQLFragment, error) {
	return c.renderScalar(expr)
}

func (c *sqlCompiler) AddArg(value any) SQLFragment {
	c.args = append(c.args, value)
	return SQLFragment{Text: c.dialect.Placeholder(len(c.args))}
}

func (c *sqlCompiler) renderExpression(expr Expression) (SQLFragment, error) {
	switch value := expr.(type) {
	case *LogicalExpression:
		return c.renderLogical(value)
	case *ComparisonExpression:
		return c.renderComparison(value)
	case *LikeExpression:
		return c.renderLike(value)
	case *BetweenExpression:
		return c.renderBetween(value)
	case *InExpression:
		return c.renderIn(value)
	case *IsNullExpression:
		return c.renderIsNull(value)
	case *SpatialPredicateExpression:
		frag, handled, err := c.dialect.RenderSpatialPredicate(c, value)
		if err != nil || handled {
			return frag, err
		}
		return SQLFragment{}, fmt.Errorf("dialect does not support spatial predicate %q", value.Op)
	case *TemporalPredicateExpression:
		frag, handled, err := c.dialect.RenderTemporalPredicate(c, value)
		if err != nil || handled {
			return frag, err
		}
		return SQLFragment{}, fmt.Errorf("dialect does not support temporal predicate %q", value.Op)
	case *ArrayPredicateExpression:
		frag, handled, err := c.dialect.RenderArrayPredicate(c, value)
		if err != nil || handled {
			return frag, err
		}
		return SQLFragment{}, fmt.Errorf("dialect does not support array predicate %q", value.Op)
	case *FunctionCall:
		frag, handled, err := c.dialect.RenderFunction(c, value)
		if err != nil || handled {
			return frag, err
		}
		return SQLFragment{}, fmt.Errorf("dialect does not support function %q", value.Name)
	case *Literal:
		if value.Kind == LiteralBool {
			if b, ok := value.Value.(bool); ok && b {
				return SQLFragment{Text: "TRUE"}, nil
			}
			return SQLFragment{Text: "FALSE"}, nil
		}
		return c.renderScalar(value)
	default:
		return SQLFragment{}, fmt.Errorf("unsupported CQL2 expression %T for SQL generation", expr)
	}
}

func (c *sqlCompiler) renderScalar(expr ScalarExpression) (SQLFragment, error) {
	switch value := expr.(type) {
	case *Literal:
		return c.renderLiteral(value)
	case *PropertyRef:
		return c.renderProperty(value)
	case *FunctionCall:
		frag, handled, err := c.dialect.RenderFunction(c, value)
		if err != nil || handled {
			return frag, err
		}
		return SQLFragment{}, fmt.Errorf("dialect does not support function %q", value.Name)
	case *ArithmeticExpression:
		return c.renderArithmetic(value)
	case *TemporalInstant:
		return c.renderTemporalInstant(value)
	case *ArrayLiteral:
		return c.renderArrayLiteral(value)
	default:
		return SQLFragment{}, fmt.Errorf("unsupported CQL2 scalar %T for SQL generation", expr)
	}
}

func (c *sqlCompiler) renderLiteral(lit *Literal) (SQLFragment, error) {
	switch lit.Kind {
	case LiteralNumber:
		return c.dialect.RenderNumberLiteral(c, sqlNumberText(lit.Value))
	case LiteralNull:
		return SQLFragment{Text: "NULL"}, nil
	default:
		return c.AddArg(lit.Value), nil
	}
}

func sqlNumberText(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

func (c *sqlCompiler) renderProperty(ref *PropertyRef) (SQLFragment, error) {
	var expr SQLExpr
	if c.cfg.resolver != nil {
		resolved, err := c.cfg.resolver.ResolveSQLProperty(ref)
		if err == nil {
			expr = resolved
		} else if !c.cfg.defaultColumnMapping || !errors.Is(err, ErrNoSQLMapping) {
			return SQLFragment{}, err
		}
	}
	if expr == nil {
		if !c.cfg.defaultColumnMapping {
			return SQLFragment{}, fmt.Errorf("%w %q", ErrNoSQLMapping, ref.Name)
		}
		expr = Column(ref.Name)
	}
	return c.renderSQLExpr(expr)
}

func (c *sqlCompiler) renderSQLExpr(expr SQLExpr) (SQLFragment, error) {
	switch value := expr.(type) {
	case sqlColumnExpr:
		if len(value.Parts) == 0 {
			return SQLFragment{}, fmt.Errorf("SQL column mapping has no identifier parts")
		}
		parts := make([]string, len(value.Parts))
		for i, part := range value.Parts {
			quoted, err := c.dialect.QuoteIdentifier(part)
			if err != nil {
				return SQLFragment{}, err
			}
			parts[i] = quoted
		}
		return SQLFragment{Text: strings.Join(parts, ".")}, nil
	case sqlRawExpr:
		if strings.TrimSpace(value.SQL) == "" {
			return SQLFragment{}, fmt.Errorf("raw SQL mapping must not be empty")
		}
		return SQLFragment{Text: value.SQL}, nil
	default:
		return SQLFragment{}, fmt.Errorf("unsupported SQL expression mapping %T", expr)
	}
}

func (c *sqlCompiler) renderLogical(expr *LogicalExpression) (SQLFragment, error) {
	if expr.Op == LogicalNot {
		if len(expr.Args) != 1 {
			return SQLFragment{}, fmt.Errorf("NOT expects exactly one argument")
		}
		arg, err := c.renderExpression(expr.Args[0])
		if err != nil {
			return SQLFragment{}, err
		}
		return SQLFragment{Text: "(NOT (" + arg.Text + "))"}, nil
	}
	if len(expr.Args) == 0 {
		return SQLFragment{}, fmt.Errorf("logical expression %q has no arguments", expr.Op)
	}
	parts := make([]string, len(expr.Args))
	for i, argExpr := range expr.Args {
		arg, err := c.renderExpression(argExpr)
		if err != nil {
			return SQLFragment{}, err
		}
		parts[i] = "(" + arg.Text + ")"
	}
	op := " AND "
	if expr.Op == LogicalOr {
		op = " OR "
	}
	return SQLFragment{Text: "(" + strings.Join(parts, op) + ")"}, nil
}

func (c *sqlCompiler) renderComparison(expr *ComparisonExpression) (SQLFragment, error) {
	left, err := c.renderScalar(expr.Left)
	if err != nil {
		return SQLFragment{}, err
	}
	right, err := c.renderScalar(expr.Right)
	if err != nil {
		return SQLFragment{}, err
	}
	return SQLFragment{Text: fmt.Sprintf("(%s %s %s)", left.Text, expr.Op, right.Text)}, nil
}

func (c *sqlCompiler) renderLike(expr *LikeExpression) (SQLFragment, error) {
	left, err := c.renderScalar(expr.Expr)
	if err != nil {
		return SQLFragment{}, err
	}
	right, err := c.renderScalar(expr.Pattern)
	if err != nil {
		return SQLFragment{}, err
	}
	op := "LIKE"
	if expr.Not {
		op = "NOT LIKE"
	}
	return SQLFragment{Text: fmt.Sprintf("(%s %s %s)", left.Text, op, right.Text)}, nil
}

func (c *sqlCompiler) renderBetween(expr *BetweenExpression) (SQLFragment, error) {
	value, err := c.renderScalar(expr.Expr)
	if err != nil {
		return SQLFragment{}, err
	}
	lower, err := c.renderScalar(expr.Lower)
	if err != nil {
		return SQLFragment{}, err
	}
	upper, err := c.renderScalar(expr.Upper)
	if err != nil {
		return SQLFragment{}, err
	}
	not := ""
	if expr.Not {
		not = " NOT"
	}
	return SQLFragment{Text: fmt.Sprintf("(%s%s BETWEEN %s AND %s)", value.Text, not, lower.Text, upper.Text)}, nil
}

func (c *sqlCompiler) renderIn(expr *InExpression) (SQLFragment, error) {
	value, err := c.renderScalar(expr.Expr)
	if err != nil {
		return SQLFragment{}, err
	}
	if len(expr.Values) == 0 {
		return SQLFragment{}, fmt.Errorf("IN expression has no values")
	}
	parts := make([]string, len(expr.Values))
	for i, itemExpr := range expr.Values {
		item, err := c.renderScalar(itemExpr)
		if err != nil {
			return SQLFragment{}, err
		}
		parts[i] = item.Text
	}
	not := ""
	if expr.Not {
		not = " NOT"
	}
	return SQLFragment{Text: fmt.Sprintf("(%s%s IN (%s))", value.Text, not, strings.Join(parts, ", "))}, nil
}

func (c *sqlCompiler) renderIsNull(expr *IsNullExpression) (SQLFragment, error) {
	value, err := c.Render(expr.Expr)
	if err != nil {
		return SQLFragment{}, err
	}
	not := ""
	if expr.Not {
		not = " NOT"
	}
	return SQLFragment{Text: fmt.Sprintf("(%s IS%s NULL)", value.Text, not)}, nil
}

func (c *sqlCompiler) renderArithmetic(expr *ArithmeticExpression) (SQLFragment, error) {
	left, err := c.renderScalar(expr.Left)
	if err != nil {
		return SQLFragment{}, err
	}
	right, err := c.renderScalar(expr.Right)
	if err != nil {
		return SQLFragment{}, err
	}
	switch expr.Op {
	case ArithmeticPow:
		return SQLFragment{Text: fmt.Sprintf("power(%s, %s)", left.Text, right.Text)}, nil
	case ArithmeticIntDiv:
		return SQLFragment{Text: fmt.Sprintf("trunc((%s) / (%s))", left.Text, right.Text)}, nil
	default:
		return SQLFragment{Text: fmt.Sprintf("(%s %s %s)", left.Text, expr.Op, right.Text)}, nil
	}
}

func (c *sqlCompiler) renderTemporalInstant(value *TemporalInstant) (SQLFragment, error) {
	return c.dialect.RenderTemporalInstant(c, value)
}

func (c *sqlCompiler) RenderTemporalOperand(node Node) (SQLTemporalOperand, error) {
	start, err := c.RenderTemporalStart(node)
	if err != nil {
		return SQLTemporalOperand{}, err
	}
	end, err := c.RenderTemporalEnd(node)
	if err != nil {
		return SQLTemporalOperand{}, err
	}
	return SQLTemporalOperand{Start: start, End: end}, nil
}

func (c *sqlCompiler) RenderTemporalStart(node Node) (SQLFragment, error) {
	if interval, ok := node.(*TemporalInterval); ok {
		return c.renderTemporalEndpoint(interval.Start)
	}
	return c.renderTemporalInstantOperand(node)
}

func (c *sqlCompiler) RenderTemporalEnd(node Node) (SQLFragment, error) {
	if interval, ok := node.(*TemporalInterval); ok {
		return c.renderTemporalEndpoint(interval.End)
	}
	return c.renderTemporalInstantOperand(node)
}

func (c *sqlCompiler) renderTemporalInstantOperand(node Node) (SQLFragment, error) {
	switch value := node.(type) {
	case *TemporalInstant:
		return c.renderTemporalInstant(value)
	case *PropertyRef:
		return c.renderProperty(value)
	case *FunctionCall:
		return c.renderScalar(value)
	default:
		if scalar, ok := node.(ScalarExpression); ok {
			return c.renderScalar(scalar)
		}
		return SQLFragment{}, fmt.Errorf("unsupported temporal operand %T", node)
	}
}

func (c *sqlCompiler) renderTemporalEndpoint(node Node) (SQLFragment, error) {
	if _, ok := node.(*TemporalUnbounded); ok {
		return SQLFragment{}, fmt.Errorf("unbounded temporal intervals are not supported by SQL generation yet")
	}
	return c.Render(node)
}

func (c *sqlCompiler) renderArrayLiteral(lit *ArrayLiteral) (SQLFragment, error) {
	parts := make([]string, len(lit.Values))
	for i, value := range lit.Values {
		frag, err := c.Render(value)
		if err != nil {
			return SQLFragment{}, err
		}
		parts[i] = frag.Text
	}
	return SQLFragment{Text: "ARRAY[" + strings.Join(parts, ", ") + "]"}, nil
}

func (c *sqlCompiler) renderGeometryLiteral(lit *GeometryLiteral) (SQLFragment, error) {
	frag, handled, err := c.dialect.RenderGeometryLiteral(c, lit)
	if err != nil || handled {
		return frag, err
	}
	return SQLFragment{}, fmt.Errorf("dialect does not support geometry literal %q", lit.Type)
}

// PostGISOption configures the built-in PostgreSQL/PostGIS dialect.
type PostGISOption func(*postGISDialect)

// WithSRID configures the SRID used for CQL2 geometry literals. The default is 4326.
func WithSRID(srid int) PostGISOption { return func(d *postGISDialect) { d.srid = srid } }

// PostGISDialect returns a PostgreSQL/PostGIS SQL dialect.
func PostGISDialect(opts ...PostGISOption) SQLDialect {
	d := &postGISDialect{srid: 4326}
	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}
	return d
}

type postGISDialect struct {
	BaseDialect
	srid int
}

func (d *postGISDialect) Placeholder(index int) string { return "$" + strconv.Itoa(index) }

func (d *postGISDialect) RenderNumberLiteral(ctx SQLRenderContext, value string) (SQLFragment, error) {
	return SQLFragment{Text: "CAST(" + ctx.AddArg(value).Text + " AS numeric)"}, nil
}

func (d *postGISDialect) RenderTemporalInstant(ctx SQLRenderContext, value *TemporalInstant) (SQLFragment, error) {
	arg := ctx.AddArg(value.Value)
	if value.Kind == TemporalInstantDate {
		return SQLFragment{Text: "CAST(" + arg.Text + " AS date)"}, nil
	}
	return SQLFragment{Text: "CAST(" + arg.Text + " AS timestamptz)"}, nil
}

func (d *postGISDialect) RenderFunction(ctx SQLRenderContext, call *FunctionCall) (SQLFragment, bool, error) {
	name := normalizeFunctionName(call.Name)
	if name != FunctionNameCaseI && name != FunctionNameAccenti {
		return SQLFragment{}, false, nil
	}
	if len(call.Args) != 1 {
		return SQLFragment{}, true, fmt.Errorf("function %q expects exactly one argument", name)
	}
	arg, err := ctx.Render(call.Args[0])
	if err != nil {
		return SQLFragment{}, true, err
	}
	fn := "lower"
	if name == FunctionNameAccenti {
		fn = "unaccent"
	}
	return SQLFragment{Text: fmt.Sprintf("%s(%s)", fn, arg.Text)}, true, nil
}

func (d *postGISDialect) RenderSpatialPredicate(ctx SQLRenderContext, expr *SpatialPredicateExpression) (SQLFragment, bool, error) {
	fn := map[SpatialPredicateOp]string{
		SpatialOpContains:   "ST_Contains",
		SpatialOpCrosses:    "ST_Crosses",
		SpatialOpDisjoint:   "ST_Disjoint",
		SpatialOpEquals:     "ST_Equals",
		SpatialOpIntersects: "ST_Intersects",
		SpatialOpOverlaps:   "ST_Overlaps",
		SpatialOpTouches:    "ST_Touches",
		SpatialOpWithin:     "ST_Within",
	}[expr.Op]
	if fn == "" {
		return SQLFragment{}, false, nil
	}
	left, err := ctx.Render(expr.Left)
	if err != nil {
		return SQLFragment{}, true, err
	}
	right, err := ctx.Render(expr.Right)
	if err != nil {
		return SQLFragment{}, true, err
	}
	return SQLFragment{Text: fmt.Sprintf("%s(%s, %s)", fn, left.Text, right.Text)}, true, nil
}

func (d *postGISDialect) RenderArrayPredicate(ctx SQLRenderContext, expr *ArrayPredicateExpression) (SQLFragment, bool, error) {
	op := map[ArrayPredicateOp]string{
		ArrayOpContains:    "@>",
		ArrayOpContainedBy: "<@",
		ArrayOpEquals:      "=",
		ArrayOpOverlaps:    "&&",
	}[expr.Op]
	if op == "" {
		return SQLFragment{}, false, nil
	}
	left, err := ctx.Render(expr.Left)
	if err != nil {
		return SQLFragment{}, true, err
	}
	right, err := ctx.Render(expr.Right)
	if err != nil {
		return SQLFragment{}, true, err
	}
	return SQLFragment{Text: fmt.Sprintf("(%s %s %s)", left.Text, op, right.Text)}, true, nil
}

func (d *postGISDialect) RenderTemporalPredicate(ctx SQLRenderContext, expr *TemporalPredicateExpression) (SQLFragment, bool, error) {
	cmp := func(a SQLFragment, op string, b SQLFragment) string {
		return fmt.Sprintf("(%s %s %s)", a.Text, op, b.Text)
	}
	and := func(parts ...string) SQLFragment { return SQLFragment{Text: "(" + strings.Join(parts, " AND ") + ")"} }
	or := func(parts ...string) SQLFragment { return SQLFragment{Text: "(" + strings.Join(parts, " OR ") + ")"} }
	start := func(node Node) (SQLFragment, error) { return ctx.RenderTemporalStart(node) }
	end := func(node Node) (SQLFragment, error) { return ctx.RenderTemporalEnd(node) }
	both := func(node Node) (SQLTemporalOperand, error) {
		startFrag, err := start(node)
		if err != nil {
			return SQLTemporalOperand{}, err
		}
		endFrag, err := end(node)
		if err != nil {
			return SQLTemporalOperand{}, err
		}
		return SQLTemporalOperand{Start: startFrag, End: endFrag}, nil
	}
	switch expr.Op {
	case TemporalOpAfter:
		left, err := start(expr.Left)
		if err != nil {
			return SQLFragment{}, true, err
		}
		right, err := end(expr.Right)
		if err != nil {
			return SQLFragment{}, true, err
		}
		return SQLFragment{Text: cmp(left, ">", right)}, true, nil
	case TemporalOpBefore:
		left, err := end(expr.Left)
		if err != nil {
			return SQLFragment{}, true, err
		}
		right, err := start(expr.Right)
		if err != nil {
			return SQLFragment{}, true, err
		}
		return SQLFragment{Text: cmp(left, "<", right)}, true, nil
	case TemporalOpMeets:
		left, err := end(expr.Left)
		if err != nil {
			return SQLFragment{}, true, err
		}
		right, err := start(expr.Right)
		if err != nil {
			return SQLFragment{}, true, err
		}
		return SQLFragment{Text: cmp(left, "=", right)}, true, nil
	case TemporalOpMetBy:
		left, err := start(expr.Left)
		if err != nil {
			return SQLFragment{}, true, err
		}
		right, err := end(expr.Right)
		if err != nil {
			return SQLFragment{}, true, err
		}
		return SQLFragment{Text: cmp(left, "=", right)}, true, nil
	}
	left, err := both(expr.Left)
	if err != nil {
		return SQLFragment{}, true, err
	}
	right, err := both(expr.Right)
	if err != nil {
		return SQLFragment{}, true, err
	}
	disjoint := or(cmp(left.End, "<", right.Start), cmp(left.Start, ">", right.End)).Text
	switch expr.Op {
	case TemporalOpDisjoint:
		return SQLFragment{Text: disjoint}, true, nil
	case TemporalOpEquals:
		return and(cmp(left.Start, "=", right.Start), cmp(left.End, "=", right.End)), true, nil
	case TemporalOpIntersects:
		return SQLFragment{Text: "(NOT " + disjoint + ")"}, true, nil
	case TemporalOpContains:
		return and(cmp(left.Start, "<=", right.Start), cmp(left.End, ">=", right.End)), true, nil
	case TemporalOpDuring:
		return and(cmp(left.Start, ">=", right.Start), cmp(left.End, "<=", right.End)), true, nil
	case TemporalOpFinishedBy:
		return and(cmp(left.End, "=", right.End), cmp(left.Start, "<", right.Start)), true, nil
	case TemporalOpFinishes:
		return and(cmp(left.End, "=", right.End), cmp(left.Start, ">", right.Start)), true, nil
	case TemporalOpOverlappedBy:
		return and(cmp(right.Start, "<", left.Start), cmp(right.End, ">", left.Start), cmp(right.End, "<", left.End)), true, nil
	case TemporalOpOverlaps:
		return and(cmp(left.Start, "<", right.Start), cmp(left.End, ">", right.Start), cmp(left.End, "<", right.End)), true, nil
	case TemporalOpStartedBy:
		return and(cmp(left.Start, "=", right.Start), cmp(left.End, ">", right.End)), true, nil
	case TemporalOpStarts:
		return and(cmp(left.Start, "=", right.Start), cmp(left.End, "<", right.End)), true, nil
	default:
		return SQLFragment{}, false, nil
	}
}

func (d *postGISDialect) RenderGeometryLiteral(ctx SQLRenderContext, lit *GeometryLiteral) (SQLFragment, bool, error) {
	if lit.Type == GeometryTypeBBox {
		if len(lit.BBox) == 4 {
			parts := make([]string, 4)
			for i, value := range lit.BBox {
				parts[i] = ctx.AddArg(value).Text
			}
			return SQLFragment{Text: fmt.Sprintf("ST_MakeEnvelope(%s, %s, %s, %s, %d)", parts[0], parts[1], parts[2], parts[3], d.srid)}, true, nil
		}
		if len(lit.BBox) == 6 {
			return SQLFragment{}, true, fmt.Errorf("PostGIS SQL generation does not support 3D BBOX literals yet")
		}
		return SQLFragment{}, true, fmt.Errorf("invalid BBOX coordinate count %d", len(lit.BBox))
	}
	geojson, err := geometryLiteralGeoJSON(lit)
	if err != nil {
		return SQLFragment{}, true, err
	}
	arg := ctx.AddArg(geojson)
	return SQLFragment{Text: fmt.Sprintf("ST_SetSRID(ST_GeomFromGeoJSON(%s), %d)", arg.Text, d.srid)}, true, nil
}

func geometryLiteralGeoJSON(lit *GeometryLiteral) (string, error) {
	obj, err := geometryLiteralGeoJSONObject(lit)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func geometryLiteralGeoJSONObject(lit *GeometryLiteral) (map[string]any, error) {
	if lit == nil {
		return nil, fmt.Errorf("nil geometry literal")
	}
	if lit.Type == GeometryTypeGeometryCollection {
		geoms := make([]any, len(lit.Geometries))
		for i, geom := range lit.Geometries {
			obj, err := geometryLiteralGeoJSONObject(geom)
			if err != nil {
				return nil, err
			}
			geoms[i] = obj
		}
		return map[string]any{"type": string(lit.Type), "geometries": geoms}, nil
	}
	coords, err := geoJSONCoordinates(lit.Coordinates)
	if err != nil {
		return nil, fmt.Errorf("render %s coordinates: %w", lit.Type, err)
	}
	return map[string]any{"type": string(lit.Type), "coordinates": coords}, nil
}

func geoJSONCoordinates(value any) (any, error) {
	switch v := value.(type) {
	case Coordinate:
		return geoJSONCoordinate(v), nil
	case []Coordinate:
		out := make([]any, len(v))
		for i, coord := range v {
			out[i] = geoJSONCoordinate(coord)
		}
		return out, nil
	case [][]Coordinate:
		out := make([]any, len(v))
		for i, ring := range v {
			coords, err := geoJSONCoordinates(ring)
			if err != nil {
				return nil, err
			}
			out[i] = coords
		}
		return out, nil
	case [][][]Coordinate:
		out := make([]any, len(v))
		for i, polygon := range v {
			coords, err := geoJSONCoordinates(polygon)
			if err != nil {
				return nil, err
			}
			out[i] = coords
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported coordinate shape %T", value)
	}
}

func geoJSONCoordinate(coord Coordinate) []float64 {
	if coord.HasZ {
		return []float64{coord.X, coord.Y, coord.Z}
	}
	return []float64{coord.X, coord.Y}
}
