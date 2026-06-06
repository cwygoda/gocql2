package sql

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cwygoda/gocql2/api"
)

// SQL is a parameterized SQL fragment generated from a CQL2 expression.
type SQL struct {
	Text string
	Args []any
}

// Fragment is a SQL fragment without independently-owned arguments. Dialect
// renderers receive a context that owns argument numbering.
type Fragment struct {
	Text string
}

// Expr is a trusted application-authored SQL expression used to map a CQL2
// property/queryable name to SQL.
type Expr interface{ sqlExpr() }

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
func Column(parts ...string) Expr { return sqlColumnExpr{Parts: append([]string(nil), parts...)} }

// RawSQL maps a CQL2 property to trusted application-authored SQL. Never build
// RawSQL from untrusted CQL2 input.
func RawSQL(sql string) Expr { return sqlRawExpr{SQL: sql} }

// Property maps a CQL2 queryable/property name to a SQL expression.
//
//nolint:govet // Public field order follows API readability: CQL2 name/type, then SQL expression.
type Property struct {
	Name string
	Type api.PropertyType
	Expr Expr
}

// ErrNoSQLMapping reports that a resolver has no SQL mapping for a CQL2 property.
var ErrNoSQLMapping = errors.New("no SQL mapping for CQL2 property")

// PropertyResolver resolves CQL2 property references during SQL generation.
type PropertyResolver interface {
	ResolveProperty(ref *api.PropertyRef) (Expr, error)
}

// Dialect renders SQL constructs that vary by database.
type Dialect interface {
	Placeholder(index int) string
	QuoteIdentifier(name string) (string, error)
	RenderNumberLiteral(ctx RenderContext, value string) (Fragment, error)
	RenderTemporalInstant(ctx RenderContext, value *api.TemporalInstant) (Fragment, error)
	RenderFunction(ctx RenderContext, call *api.FunctionCall) (Fragment, bool, error)
	RenderSpatialPredicate(ctx RenderContext, expr *api.SpatialPredicateExpression) (Fragment, bool, error)
	RenderTemporalPredicate(ctx RenderContext, expr *api.TemporalPredicateExpression) (Fragment, bool, error)
	RenderArrayPredicate(ctx RenderContext, expr *api.ArrayPredicateExpression) (Fragment, bool, error)
	RenderGeometryLiteral(ctx RenderContext, lit *api.GeometryLiteral) (Fragment, bool, error)
}

// RenderContext is passed to dialect hooks for recursive rendering and bind
// argument allocation.
type RenderContext interface {
	Render(node api.Node) (Fragment, error)
	RenderScalar(expr api.ScalarExpression) (Fragment, error)
	RenderTemporalOperand(node api.Node) (TemporalOperand, error)
	RenderTemporalStart(node api.Node) (Fragment, error)
	RenderTemporalEnd(node api.Node) (Fragment, error)
	AddArg(value any) Fragment
}

// TemporalOperand represents an instant as Start == End and an interval as
// separate start/end SQL expressions.
type TemporalOperand struct {
	Start Fragment
	End   Fragment
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
func (BaseDialect) RenderNumberLiteral(ctx RenderContext, value string) (Fragment, error) {
	return Fragment{Text: "CAST(" + ctx.AddArg(value).Text + " AS NUMERIC)"}, nil
}

// RenderTemporalInstant renders a temporal instant literal.
func (BaseDialect) RenderTemporalInstant(ctx RenderContext, value *api.TemporalInstant) (Fragment, error) {
	arg := ctx.AddArg(value.Value)
	if value.Kind == api.TemporalInstantDate {
		return Fragment{Text: "CAST(" + arg.Text + " AS DATE)"}, nil
	}
	return Fragment{Text: "CAST(" + arg.Text + " AS TIMESTAMP)"}, nil
}

// RenderFunction declines all function rendering.
func (BaseDialect) RenderFunction(RenderContext, *api.FunctionCall) (Fragment, bool, error) {
	return Fragment{}, false, nil
}

// RenderSpatialPredicate declines all spatial predicate rendering.
func (BaseDialect) RenderSpatialPredicate(RenderContext, *api.SpatialPredicateExpression) (Fragment, bool, error) {
	return Fragment{}, false, nil
}

// RenderTemporalPredicate declines all temporal predicate rendering.
func (BaseDialect) RenderTemporalPredicate(RenderContext, *api.TemporalPredicateExpression) (Fragment, bool, error) {
	return Fragment{}, false, nil
}

// RenderArrayPredicate declines all array predicate rendering.
func (BaseDialect) RenderArrayPredicate(RenderContext, *api.ArrayPredicateExpression) (Fragment, bool, error) {
	return Fragment{}, false, nil
}

// RenderGeometryLiteral declines all geometry literal rendering.
func (BaseDialect) RenderGeometryLiteral(RenderContext, *api.GeometryLiteral) (Fragment, bool, error) {
	return Fragment{}, false, nil
}

// Option configures SQL generation.
type Option func(*sqlConfig)

type sqlConfig struct {
	resolver             PropertyResolver
	defaultColumnMapping bool
}

// WithSQLProperties configures explicit fail-closed CQL2 property-to-SQL mappings.
func WithSQLProperties(props ...Property) Option {
	return func(cfg *sqlConfig) {
		cfg.resolver = sqlPropertyMap(props)
	}
}

// WithPropertyResolver configures a custom property resolver.
func WithPropertyResolver(resolver PropertyResolver) Option {
	return func(cfg *sqlConfig) { cfg.resolver = resolver }
}

// WithDefaultColumnMapping allows unmapped CQL2 properties to map to quoted SQL
// columns of the same name. Without this or an explicit resolver, SQL generation
// fails closed for properties.
func WithDefaultColumnMapping() Option {
	return func(cfg *sqlConfig) { cfg.defaultColumnMapping = true }
}

type sqlPropertyMap []Property

func (m sqlPropertyMap) ResolveProperty(ref *api.PropertyRef) (Expr, error) {
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

// PropertyDefinitions returns parse-time property definitions matching SQL
// property mappings, avoiding duplicate schema declarations for common callers.
func PropertyDefinitions(props ...Property) []api.PropertyDefinition {
	defs := make([]api.PropertyDefinition, 0, len(props))
	for _, prop := range props {
		if prop.Name != "" {
			defs = append(defs, api.PropertyDefinition{Name: prop.Name, Type: prop.Type})
		}
	}
	return defs
}

// ToSQL compiles a parsed CQL2 expression to a parameterized SQL fragment.
func ToSQL(expr api.Expression, dialect Dialect, opts ...Option) (SQL, error) {
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
	dialect Dialect
	cfg     sqlConfig
	args    []any
}

func (c *sqlCompiler) Render(node api.Node) (Fragment, error) {
	switch value := node.(type) {
	case api.Expression:
		return c.renderExpression(value)
	case api.ScalarExpression:
		return c.renderScalar(value)
	case *api.GeometryLiteral:
		return c.renderGeometryLiteral(value)
	case *api.TemporalInterval:
		operand, err := c.RenderTemporalOperand(value)
		if err != nil {
			return Fragment{}, err
		}
		return Fragment{Text: fmt.Sprintf("(%s, %s)", operand.Start.Text, operand.End.Text)}, nil
	case *api.TemporalUnbounded:
		return Fragment{}, fmt.Errorf("temporal unbounded endpoint cannot be rendered as a standalone SQL expression")
	default:
		return Fragment{}, fmt.Errorf("unsupported CQL2 node %T for SQL generation", node)
	}
}

func (c *sqlCompiler) RenderScalar(expr api.ScalarExpression) (Fragment, error) {
	return c.renderScalar(expr)
}

func (c *sqlCompiler) AddArg(value any) Fragment {
	c.args = append(c.args, value)
	return Fragment{Text: c.dialect.Placeholder(len(c.args))}
}

func (c *sqlCompiler) renderExpression(expr api.Expression) (Fragment, error) {
	switch value := expr.(type) {
	case *api.LogicalExpression:
		return c.renderLogical(value)
	case *api.ComparisonExpression:
		return c.renderComparison(value)
	case *api.LikeExpression:
		return c.renderLike(value)
	case *api.BetweenExpression:
		return c.renderBetween(value)
	case *api.InExpression:
		return c.renderIn(value)
	case *api.IsNullExpression:
		return c.renderIsNull(value)
	case *api.SpatialPredicateExpression:
		frag, handled, err := c.dialect.RenderSpatialPredicate(c, value)
		if err != nil || handled {
			return frag, err
		}
		return Fragment{}, fmt.Errorf("dialect does not support spatial predicate %q", value.Op)
	case *api.TemporalPredicateExpression:
		frag, handled, err := c.dialect.RenderTemporalPredicate(c, value)
		if err != nil || handled {
			return frag, err
		}
		return Fragment{}, fmt.Errorf("dialect does not support temporal predicate %q", value.Op)
	case *api.ArrayPredicateExpression:
		frag, handled, err := c.dialect.RenderArrayPredicate(c, value)
		if err != nil || handled {
			return frag, err
		}
		return Fragment{}, fmt.Errorf("dialect does not support array predicate %q", value.Op)
	case *api.FunctionCall:
		frag, handled, err := c.dialect.RenderFunction(c, value)
		if err != nil || handled {
			return frag, err
		}
		return Fragment{}, fmt.Errorf("dialect does not support function %q", value.Name)
	case *api.Literal:
		if value.Kind == api.LiteralBool {
			if b, ok := value.Value.(bool); ok && b {
				return Fragment{Text: "TRUE"}, nil
			}
			return Fragment{Text: "FALSE"}, nil
		}
		return c.renderScalar(value)
	default:
		return Fragment{}, fmt.Errorf("unsupported CQL2 expression %T for SQL generation", expr)
	}
}

func (c *sqlCompiler) renderScalar(expr api.ScalarExpression) (Fragment, error) {
	switch value := expr.(type) {
	case *api.Literal:
		return c.renderLiteral(value)
	case *api.PropertyRef:
		return c.renderProperty(value)
	case *api.FunctionCall:
		frag, handled, err := c.dialect.RenderFunction(c, value)
		if err != nil || handled {
			return frag, err
		}
		return Fragment{}, fmt.Errorf("dialect does not support function %q", value.Name)
	case *api.ArithmeticExpression:
		return c.renderArithmetic(value)
	case *api.TemporalInstant:
		return c.renderTemporalInstant(value)
	case *api.ArrayLiteral:
		return c.renderArrayLiteral(value)
	default:
		return Fragment{}, fmt.Errorf("unsupported CQL2 scalar %T for SQL generation", expr)
	}
}

func (c *sqlCompiler) renderLiteral(lit *api.Literal) (Fragment, error) {
	switch lit.Kind {
	case api.LiteralNumber:
		return c.dialect.RenderNumberLiteral(c, sqlNumberText(lit.Value))
	case api.LiteralNull:
		return Fragment{Text: "NULL"}, nil
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

func (c *sqlCompiler) renderProperty(ref *api.PropertyRef) (Fragment, error) {
	var expr Expr
	if c.cfg.resolver != nil {
		resolved, err := c.cfg.resolver.ResolveProperty(ref)
		if err == nil {
			expr = resolved
		} else if !c.cfg.defaultColumnMapping || !errors.Is(err, ErrNoSQLMapping) {
			return Fragment{}, err
		}
	}
	if expr == nil {
		if !c.cfg.defaultColumnMapping {
			return Fragment{}, fmt.Errorf("%w %q", ErrNoSQLMapping, ref.Name)
		}
		expr = Column(ref.Name)
	}
	return c.renderExpr(expr)
}

func (c *sqlCompiler) renderExpr(expr Expr) (Fragment, error) {
	switch value := expr.(type) {
	case sqlColumnExpr:
		if len(value.Parts) == 0 {
			return Fragment{}, fmt.Errorf("SQL column mapping has no identifier parts")
		}
		parts := make([]string, len(value.Parts))
		for i, part := range value.Parts {
			quoted, err := c.dialect.QuoteIdentifier(part)
			if err != nil {
				return Fragment{}, err
			}
			parts[i] = quoted
		}
		return Fragment{Text: strings.Join(parts, ".")}, nil
	case sqlRawExpr:
		if strings.TrimSpace(value.SQL) == "" {
			return Fragment{}, fmt.Errorf("raw SQL mapping must not be empty")
		}
		return Fragment{Text: value.SQL}, nil
	default:
		return Fragment{}, fmt.Errorf("unsupported SQL expression mapping %T", expr)
	}
}

func (c *sqlCompiler) renderLogical(expr *api.LogicalExpression) (Fragment, error) {
	if expr.Op == api.LogicalNot {
		if len(expr.Args) != 1 {
			return Fragment{}, fmt.Errorf("NOT expects exactly one argument")
		}
		arg, err := c.renderExpression(expr.Args[0])
		if err != nil {
			return Fragment{}, err
		}
		return Fragment{Text: "(NOT (COALESCE(" + arg.Text + ", FALSE)))"}, nil
	}
	if len(expr.Args) == 0 {
		return Fragment{}, fmt.Errorf("logical expression %q has no arguments", expr.Op)
	}
	parts := make([]string, len(expr.Args))
	for i, argExpr := range expr.Args {
		arg, err := c.renderExpression(argExpr)
		if err != nil {
			return Fragment{}, err
		}
		parts[i] = "(" + arg.Text + ")"
	}
	op := " AND "
	if expr.Op == api.LogicalOr {
		op = " OR "
	}
	return Fragment{Text: "(" + strings.Join(parts, op) + ")"}, nil
}

func (c *sqlCompiler) renderComparison(expr *api.ComparisonExpression) (Fragment, error) {
	left, err := c.renderScalar(expr.Left)
	if err != nil {
		return Fragment{}, err
	}
	right, err := c.renderScalar(expr.Right)
	if err != nil {
		return Fragment{}, err
	}
	return Fragment{Text: fmt.Sprintf("(%s %s %s)", left.Text, expr.Op, right.Text)}, nil
}

func (c *sqlCompiler) renderLike(expr *api.LikeExpression) (Fragment, error) {
	left, err := c.renderScalar(expr.Expr)
	if err != nil {
		return Fragment{}, err
	}
	right, err := c.renderScalar(expr.Pattern)
	if err != nil {
		return Fragment{}, err
	}
	op := "LIKE"
	if expr.Not {
		op = "NOT LIKE"
	}
	return Fragment{Text: fmt.Sprintf("(%s %s %s)", left.Text, op, right.Text)}, nil
}

func (c *sqlCompiler) renderBetween(expr *api.BetweenExpression) (Fragment, error) {
	value, err := c.renderScalar(expr.Expr)
	if err != nil {
		return Fragment{}, err
	}
	lower, err := c.renderScalar(expr.Lower)
	if err != nil {
		return Fragment{}, err
	}
	upper, err := c.renderScalar(expr.Upper)
	if err != nil {
		return Fragment{}, err
	}
	not := ""
	if expr.Not {
		not = " NOT"
	}
	return Fragment{Text: fmt.Sprintf("(%s%s BETWEEN %s AND %s)", value.Text, not, lower.Text, upper.Text)}, nil
}

func (c *sqlCompiler) renderIn(expr *api.InExpression) (Fragment, error) {
	value, err := c.renderScalar(expr.Expr)
	if err != nil {
		return Fragment{}, err
	}
	if len(expr.Values) == 0 {
		return Fragment{}, fmt.Errorf("IN expression has no values")
	}
	parts := make([]string, len(expr.Values))
	for i, itemExpr := range expr.Values {
		item, err := c.renderScalar(itemExpr)
		if err != nil {
			return Fragment{}, err
		}
		parts[i] = item.Text
	}
	not := ""
	if expr.Not {
		not = " NOT"
	}
	return Fragment{Text: fmt.Sprintf("(%s%s IN (%s))", value.Text, not, strings.Join(parts, ", "))}, nil
}

func (c *sqlCompiler) renderIsNull(expr *api.IsNullExpression) (Fragment, error) {
	value, err := c.Render(expr.Expr)
	if err != nil {
		return Fragment{}, err
	}
	not := ""
	if expr.Not {
		not = " NOT"
	}
	return Fragment{Text: fmt.Sprintf("(%s IS%s NULL)", value.Text, not)}, nil
}

func (c *sqlCompiler) renderArithmetic(expr *api.ArithmeticExpression) (Fragment, error) {
	left, err := c.renderScalar(expr.Left)
	if err != nil {
		return Fragment{}, err
	}
	right, err := c.renderScalar(expr.Right)
	if err != nil {
		return Fragment{}, err
	}
	switch expr.Op {
	case api.ArithmeticPow:
		return Fragment{Text: fmt.Sprintf("power(%s, %s)", left.Text, right.Text)}, nil
	case api.ArithmeticIntDiv:
		return Fragment{Text: fmt.Sprintf("trunc((%s) / (%s))", left.Text, right.Text)}, nil
	default:
		return Fragment{Text: fmt.Sprintf("(%s %s %s)", left.Text, expr.Op, right.Text)}, nil
	}
}

func (c *sqlCompiler) renderTemporalInstant(value *api.TemporalInstant) (Fragment, error) {
	return c.dialect.RenderTemporalInstant(c, value)
}

func (c *sqlCompiler) RenderTemporalOperand(node api.Node) (TemporalOperand, error) {
	start, err := c.RenderTemporalStart(node)
	if err != nil {
		return TemporalOperand{}, err
	}
	end, err := c.RenderTemporalEnd(node)
	if err != nil {
		return TemporalOperand{}, err
	}
	return TemporalOperand{Start: start, End: end}, nil
}

func (c *sqlCompiler) RenderTemporalStart(node api.Node) (Fragment, error) {
	if interval, ok := node.(*api.TemporalInterval); ok {
		return c.renderTemporalEndpoint(interval.Start)
	}
	return c.renderTemporalInstantOperand(node)
}

func (c *sqlCompiler) RenderTemporalEnd(node api.Node) (Fragment, error) {
	if interval, ok := node.(*api.TemporalInterval); ok {
		return c.renderTemporalEndpoint(interval.End)
	}
	return c.renderTemporalInstantOperand(node)
}

func (c *sqlCompiler) renderTemporalInstantOperand(node api.Node) (Fragment, error) {
	switch value := node.(type) {
	case *api.TemporalInstant:
		return c.renderTemporalInstant(value)
	case *api.PropertyRef:
		return c.renderProperty(value)
	case *api.FunctionCall:
		return c.renderScalar(value)
	default:
		if scalar, ok := node.(api.ScalarExpression); ok {
			return c.renderScalar(scalar)
		}
		return Fragment{}, fmt.Errorf("unsupported temporal operand %T", node)
	}
}

func (c *sqlCompiler) renderTemporalEndpoint(node api.Node) (Fragment, error) {
	if _, ok := node.(*api.TemporalUnbounded); ok {
		return Fragment{}, fmt.Errorf("unbounded temporal intervals are not supported by SQL generation yet")
	}
	return c.Render(node)
}

func (c *sqlCompiler) renderArrayLiteral(lit *api.ArrayLiteral) (Fragment, error) {
	parts := make([]string, len(lit.Values))
	for i, value := range lit.Values {
		frag, err := c.Render(value)
		if err != nil {
			return Fragment{}, err
		}
		parts[i] = frag.Text
	}
	return Fragment{Text: "ARRAY[" + strings.Join(parts, ", ") + "]"}, nil
}

func (c *sqlCompiler) renderGeometryLiteral(lit *api.GeometryLiteral) (Fragment, error) {
	frag, handled, err := c.dialect.RenderGeometryLiteral(c, lit)
	if err != nil || handled {
		return frag, err
	}
	return Fragment{}, fmt.Errorf("dialect does not support geometry literal %q", lit.Type)
}

// PostGISOption configures the built-in PostgreSQL/PostGIS dialect.
type PostGISOption func(*postGISDialect)

// WithSRID configures the SRID used for CQL2 geometry literals. The default is 4326.
func WithSRID(srid int) PostGISOption { return func(d *postGISDialect) { d.srid = srid } }

// PostGISDialect returns a PostgreSQL/PostGIS SQL dialect.
func PostGISDialect(opts ...PostGISOption) Dialect {
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

func (d *postGISDialect) RenderNumberLiteral(ctx RenderContext, value string) (Fragment, error) {
	return Fragment{Text: "CAST(" + ctx.AddArg(value).Text + " AS numeric)"}, nil
}

func (d *postGISDialect) RenderTemporalInstant(ctx RenderContext, value *api.TemporalInstant) (Fragment, error) {
	arg := ctx.AddArg(value.Value)
	if value.Kind == api.TemporalInstantDate {
		return Fragment{Text: "CAST(" + arg.Text + " AS date)"}, nil
	}
	return Fragment{Text: "CAST(" + arg.Text + " AS timestamptz)"}, nil
}

func (d *postGISDialect) RenderFunction(ctx RenderContext, call *api.FunctionCall) (Fragment, bool, error) {
	name := strings.ToLower(call.Name)
	if name != api.FunctionNameCaseI && name != api.FunctionNameAccenti {
		return Fragment{}, false, nil
	}
	if len(call.Args) != 1 {
		return Fragment{}, true, fmt.Errorf("function %q expects exactly one argument", name)
	}
	arg, err := ctx.Render(call.Args[0])
	if err != nil {
		return Fragment{}, true, err
	}
	fn := "lower"
	if name == api.FunctionNameAccenti {
		fn = "unaccent"
	}
	return Fragment{Text: fmt.Sprintf("%s(%s)", fn, arg.Text)}, true, nil
}

func (d *postGISDialect) RenderSpatialPredicate(ctx RenderContext, expr *api.SpatialPredicateExpression) (Fragment, bool, error) {
	fn := map[api.SpatialPredicateOp]string{
		api.SpatialOpContains:   "ST_Contains",
		api.SpatialOpCrosses:    "ST_Crosses",
		api.SpatialOpDisjoint:   "ST_Disjoint",
		api.SpatialOpEquals:     "ST_Equals",
		api.SpatialOpIntersects: "ST_Intersects",
		api.SpatialOpOverlaps:   "ST_Overlaps",
		api.SpatialOpTouches:    "ST_Touches",
		api.SpatialOpWithin:     "ST_Within",
	}[expr.Op]
	if fn == "" {
		return Fragment{}, false, nil
	}
	left, err := ctx.Render(expr.Left)
	if err != nil {
		return Fragment{}, true, err
	}
	right, err := ctx.Render(expr.Right)
	if err != nil {
		return Fragment{}, true, err
	}
	return Fragment{Text: fmt.Sprintf("%s(%s, %s)", fn, left.Text, right.Text)}, true, nil
}

func (d *postGISDialect) RenderArrayPredicate(ctx RenderContext, expr *api.ArrayPredicateExpression) (Fragment, bool, error) {
	op := map[api.ArrayPredicateOp]string{
		api.ArrayOpContains:    "@>",
		api.ArrayOpContainedBy: "<@",
		api.ArrayOpEquals:      "=",
		api.ArrayOpOverlaps:    "&&",
	}[expr.Op]
	if op == "" {
		return Fragment{}, false, nil
	}
	left, err := ctx.Render(expr.Left)
	if err != nil {
		return Fragment{}, true, err
	}
	right, err := ctx.Render(expr.Right)
	if err != nil {
		return Fragment{}, true, err
	}
	return Fragment{Text: fmt.Sprintf("(%s %s %s)", left.Text, op, right.Text)}, true, nil
}

func (d *postGISDialect) RenderTemporalPredicate(ctx RenderContext, expr *api.TemporalPredicateExpression) (Fragment, bool, error) {
	cmp := func(a Fragment, op string, b Fragment) string {
		return fmt.Sprintf("(%s %s %s)", a.Text, op, b.Text)
	}
	and := func(parts ...string) Fragment { return Fragment{Text: "(" + strings.Join(parts, " AND ") + ")"} }
	or := func(parts ...string) Fragment { return Fragment{Text: "(" + strings.Join(parts, " OR ") + ")"} }
	start := func(node api.Node) (Fragment, error) { return ctx.RenderTemporalStart(node) }
	end := func(node api.Node) (Fragment, error) { return ctx.RenderTemporalEnd(node) }
	both := func(node api.Node) (TemporalOperand, error) {
		startFrag, err := start(node)
		if err != nil {
			return TemporalOperand{}, err
		}
		endFrag, err := end(node)
		if err != nil {
			return TemporalOperand{}, err
		}
		return TemporalOperand{Start: startFrag, End: endFrag}, nil
	}
	switch expr.Op {
	case api.TemporalOpAfter:
		left, err := start(expr.Left)
		if err != nil {
			return Fragment{}, true, err
		}
		right, err := end(expr.Right)
		if err != nil {
			return Fragment{}, true, err
		}
		return Fragment{Text: cmp(left, ">", right)}, true, nil
	case api.TemporalOpBefore:
		left, err := end(expr.Left)
		if err != nil {
			return Fragment{}, true, err
		}
		right, err := start(expr.Right)
		if err != nil {
			return Fragment{}, true, err
		}
		return Fragment{Text: cmp(left, "<", right)}, true, nil
	case api.TemporalOpMeets:
		left, err := end(expr.Left)
		if err != nil {
			return Fragment{}, true, err
		}
		right, err := start(expr.Right)
		if err != nil {
			return Fragment{}, true, err
		}
		return Fragment{Text: cmp(left, "=", right)}, true, nil
	case api.TemporalOpMetBy:
		left, err := start(expr.Left)
		if err != nil {
			return Fragment{}, true, err
		}
		right, err := end(expr.Right)
		if err != nil {
			return Fragment{}, true, err
		}
		return Fragment{Text: cmp(left, "=", right)}, true, nil
	}
	left, err := both(expr.Left)
	if err != nil {
		return Fragment{}, true, err
	}
	right, err := both(expr.Right)
	if err != nil {
		return Fragment{}, true, err
	}
	disjoint := or(cmp(left.End, "<", right.Start), cmp(left.Start, ">", right.End)).Text
	switch expr.Op {
	case api.TemporalOpDisjoint:
		return Fragment{Text: disjoint}, true, nil
	case api.TemporalOpEquals:
		return and(cmp(left.Start, "=", right.Start), cmp(left.End, "=", right.End)), true, nil
	case api.TemporalOpIntersects:
		return Fragment{Text: "(NOT " + disjoint + ")"}, true, nil
	case api.TemporalOpContains:
		return and(cmp(left.Start, "<=", right.Start), cmp(left.End, ">=", right.End)), true, nil
	case api.TemporalOpDuring:
		return and(cmp(left.Start, ">=", right.Start), cmp(left.End, "<=", right.End)), true, nil
	case api.TemporalOpFinishedBy:
		return and(cmp(left.End, "=", right.End), cmp(left.Start, "<", right.Start)), true, nil
	case api.TemporalOpFinishes:
		return and(cmp(left.End, "=", right.End), cmp(left.Start, ">", right.Start)), true, nil
	case api.TemporalOpOverlappedBy:
		return and(cmp(right.Start, "<", left.Start), cmp(right.End, ">", left.Start), cmp(right.End, "<", left.End)), true, nil
	case api.TemporalOpOverlaps:
		return and(cmp(left.Start, "<", right.Start), cmp(left.End, ">", right.Start), cmp(left.End, "<", right.End)), true, nil
	case api.TemporalOpStartedBy:
		return and(cmp(left.Start, "=", right.Start), cmp(left.End, ">", right.End)), true, nil
	case api.TemporalOpStarts:
		return and(cmp(left.Start, "=", right.Start), cmp(left.End, "<", right.End)), true, nil
	default:
		return Fragment{}, false, nil
	}
}

func (d *postGISDialect) RenderGeometryLiteral(ctx RenderContext, lit *api.GeometryLiteral) (Fragment, bool, error) {
	if lit.Type == api.GeometryTypeBBox {
		if len(lit.BBox) == 4 {
			parts := make([]string, 4)
			for i, value := range lit.BBox {
				parts[i] = ctx.AddArg(value).Text
			}
			return Fragment{Text: fmt.Sprintf("ST_MakeEnvelope(%s, %s, %s, %s, %d)", parts[0], parts[1], parts[2], parts[3], d.srid)}, true, nil
		}
		if len(lit.BBox) == 6 {
			return Fragment{}, true, fmt.Errorf("PostGIS SQL generation does not support 3D BBOX literals yet")
		}
		return Fragment{}, true, fmt.Errorf("invalid BBOX coordinate count %d", len(lit.BBox))
	}
	geojson, err := geometryLiteralGeoJSON(lit)
	if err != nil {
		return Fragment{}, true, err
	}
	arg := ctx.AddArg(geojson)
	return Fragment{Text: fmt.Sprintf("ST_SetSRID(ST_GeomFromGeoJSON(%s), %d)", arg.Text, d.srid)}, true, nil
}

func geometryLiteralGeoJSON(lit *api.GeometryLiteral) (string, error) {
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

func geometryLiteralGeoJSONObject(lit *api.GeometryLiteral) (map[string]any, error) {
	if lit == nil {
		return nil, fmt.Errorf("nil geometry literal")
	}
	if lit.Type == api.GeometryTypeGeometryCollection {
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
	case api.Coordinate:
		return geoJSONCoordinate(v), nil
	case []api.Coordinate:
		out := make([]any, len(v))
		for i, coord := range v {
			out[i] = geoJSONCoordinate(coord)
		}
		return out, nil
	case [][]api.Coordinate:
		out := make([]any, len(v))
		for i, ring := range v {
			coords, err := geoJSONCoordinates(ring)
			if err != nil {
				return nil, err
			}
			out[i] = coords
		}
		return out, nil
	case [][][]api.Coordinate:
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

func geoJSONCoordinate(coord api.Coordinate) []float64 {
	if coord.HasZ {
		return []float64{coord.X, coord.Y, coord.Z}
	}
	return []float64{coord.X, coord.Y}
}
