package sql_test

import (
	"reflect"
	"strings"
	"testing"

	gocql2 "github.com/cwygoda/gocql2"

	cql2sql "github.com/cwygoda/gocql2/sql"

	"github.com/cwygoda/gocql2/api"
)

func TestToSQLScalarsAndPropertyAliases(t *testing.T) {
	props := []cql2sql.Property{
		{Name: "name", Type: api.PropertyTypeString, Expr: cql2sql.Column("f", "name")},
		{Name: "height", Type: api.PropertyTypeNumber, Expr: cql2sql.RawSQL("(properties->>'height')::numeric")},
		{Name: "active", Type: api.PropertyTypeBoolean, Expr: cql2sql.Column("active")},
	}
	defs := cql2sql.PropertyDefinitions(props...)
	expr, err := gocql2.NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators, api.ConformanceCaseInsensitiveComparison, api.ConformanceArithmetic).WithAllowedProperties(defs...).ParseText("CASEI(name) LIKE casei('foo%') AND height + 2 > 10 AND active IS NOT NULL")
	if err != nil {
		t.Fatal(err)
	}

	sql, err := cql2sql.ToSQL(expr, cql2sql.PostGISDialect(), cql2sql.WithSQLProperties(props...))
	if err != nil {
		t.Fatal(err)
	}

	wantText := `(((lower("f"."name") LIKE lower($1))) AND ((((properties->>'height')::numeric + CAST($2 AS numeric)) > CAST($3 AS numeric))) AND (("active" IS NOT NULL)))`
	if sql.Text != wantText {
		t.Fatalf("cql2sql.SQL text:\n got: %s\nwant: %s", sql.Text, wantText)
	}
	if !reflect.DeepEqual(sql.Args, []any{"foo%", "2", "10"}) {
		t.Fatalf("cql2sql.SQL args = %#v", sql.Args)
	}
}

func TestToSQLFailsClosedForUnmappedProperties(t *testing.T) {
	expr, err := gocql2.NewParser().ParseText("name = 'x'")
	if err != nil {
		t.Fatal(err)
	}
	_, err = cql2sql.ToSQL(expr, cql2sql.PostGISDialect())
	if err == nil || !strings.Contains(err.Error(), "no SQL mapping") {
		t.Fatalf("expected missing mapping error, got %v", err)
	}
}

func TestToSQLDefaultColumnMapping(t *testing.T) {
	expr, err := gocql2.NewParser().ParseText("name = 'x'")
	if err != nil {
		t.Fatal(err)
	}
	sql, err := cql2sql.ToSQL(expr, cql2sql.PostGISDialect(), cql2sql.WithDefaultColumnMapping())
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("name" = $1)` || !reflect.DeepEqual(sql.Args, []any{"x"}) {
		t.Fatalf("unexpected cql2sql.SQL: %#v", sql)
	}
}

func TestToSQLDefaultColumnMappingDoesNotHideResolverErrors(t *testing.T) {
	expr, err := gocql2.NewParser().ParseText("name = 'x'")
	if err != nil {
		t.Fatal(err)
	}
	_, err = cql2sql.ToSQL(
		expr,
		cql2sql.PostGISDialect(),
		cql2sql.WithSQLProperties(cql2sql.Property{Name: "name", Type: api.PropertyTypeString}),
		cql2sql.WithDefaultColumnMapping(),
	)
	if err == nil || !strings.Contains(err.Error(), "has no expression") {
		t.Fatalf("expected explicit resolver error, got %v", err)
	}
}

func TestToSQLDefaultColumnMappingFallsBackOnlyForUnmappedProperties(t *testing.T) {
	expr, err := gocql2.NewParser().ParseText("name = 'x'")
	if err != nil {
		t.Fatal(err)
	}
	sql, err := cql2sql.ToSQL(
		expr,
		cql2sql.PostGISDialect(),
		cql2sql.WithSQLProperties(cql2sql.Property{Name: "other", Type: api.PropertyTypeString, Expr: cql2sql.Column("other")}),
		cql2sql.WithDefaultColumnMapping(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("name" = $1)` || !reflect.DeepEqual(sql.Args, []any{"x"}) {
		t.Fatalf("unexpected cql2sql.SQL: %#v", sql)
	}
}

func TestToSQLSpatialPostGIS(t *testing.T) {
	props := []cql2sql.Property{{Name: "geom", Type: api.PropertyTypeGeometry, Expr: cql2sql.RawSQL("ST_Transform(raw_geom, 4326)")}}
	expr, err := gocql2.NewParser().WithConformance(api.ConformanceSpatialFunctions).WithAllowedProperties(cql2sql.PropertyDefinitions(props...)...).ParseText("S_INTERSECTS(geom, BBOX(-180,-90,180,90))")
	if err != nil {
		t.Fatal(err)
	}
	sql, err := cql2sql.ToSQL(expr, cql2sql.PostGISDialect(), cql2sql.WithSQLProperties(props...))
	if err != nil {
		t.Fatal(err)
	}
	want := "ST_Intersects(ST_Transform(raw_geom, 4326), ST_MakeEnvelope($1, $2, $3, $4, 4326))"
	if sql.Text != want {
		t.Fatalf("cql2sql.SQL text:\n got: %s\nwant: %s", sql.Text, want)
	}
	if !reflect.DeepEqual(sql.Args, []any{-180.0, -90.0, 180.0, 90.0}) {
		t.Fatalf("cql2sql.SQL args = %#v", sql.Args)
	}
}

func TestToSQLSpatialPostGISRejects3DBBox(t *testing.T) {
	props := []cql2sql.Property{{Name: "geom", Type: api.PropertyTypeGeometry, Expr: cql2sql.Column("geom")}}
	expr, err := gocql2.NewParser().WithConformance(api.ConformanceSpatialFunctions).WithAllowedProperties(cql2sql.PropertyDefinitions(props...)...).ParseText("S_INTERSECTS(geom, BBOX(-180,-90,0,180,90,100))")
	if err != nil {
		t.Fatal(err)
	}
	_, err = cql2sql.ToSQL(expr, cql2sql.PostGISDialect(), cql2sql.WithSQLProperties(props...))
	if err == nil || !strings.Contains(err.Error(), "3D BBOX") {
		t.Fatalf("expected 3D BBOX unsupported error, got %v", err)
	}
}

func TestToSQLTemporalLiteralsAreDialectRendered(t *testing.T) {
	expr, err := gocql2.NewParser().WithAllowedProperties(api.PropertyDefinition{Name: "event_time", Type: api.PropertyTypeTimestamp}).ParseText("event_time = TIMESTAMP('2022-04-14T14:48:46Z')")
	if err != nil {
		t.Fatal(err)
	}
	sql, err := cql2sql.ToSQL(
		expr,
		testTemporalDialect{},
		cql2sql.WithSQLProperties(cql2sql.Property{Name: "event_time", Type: api.PropertyTypeTimestamp, Expr: cql2sql.Column("event_time")}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("event_time" = temporal(?))` || !reflect.DeepEqual(sql.Args, []any{"2022-04-14T14:48:46Z"}) {
		t.Fatalf("unexpected cql2sql.SQL: %#v", sql)
	}
}

type testTemporalDialect struct{ cql2sql.BaseDialect }

func (testTemporalDialect) RenderTemporalInstant(ctx cql2sql.RenderContext, value *api.TemporalInstant) (cql2sql.Fragment, error) {
	return cql2sql.Fragment{Text: "temporal(" + ctx.AddArg(value.Value).Text + ")"}, nil
}

func TestToSQLTemporalAndArrays(t *testing.T) {
	props := []cql2sql.Property{
		{Name: "event_time", Type: api.PropertyTypeTimestamp, Expr: cql2sql.Column("event_time")},
		{Name: "tags", Type: api.PropertyTypeArray, Expr: cql2sql.Column("tags")},
	}
	defs := cql2sql.PropertyDefinitions(props...)
	tests := []struct {
		name string
		cql  string
		want string
		args []any
	}{
		{
			name: "temporal interval",
			cql:  "T_AFTER(event_time, INTERVAL('2021-01-01T00:00:00Z', '2021-12-31T23:59:59Z'))",
			want: `("event_time" > CAST($1 AS timestamptz))`,
			args: []any{"2021-12-31T23:59:59Z"},
		},
		{
			name: "array contains",
			cql:  `A_CONTAINS(tags, ('foo', 'bar'))`,
			want: `("tags" @> ARRAY[$1, $2])`,
			args: []any{"foo", "bar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := gocql2.NewParser().WithConformance(api.ConformanceTemporalFunctions, api.ConformanceArrayFunctions).WithAllowedProperties(defs...).ParseText(tt.cql)
			if err != nil {
				t.Fatal(err)
			}
			sql, err := cql2sql.ToSQL(expr, cql2sql.PostGISDialect(), cql2sql.WithSQLProperties(props...))
			if err != nil {
				t.Fatal(err)
			}
			if sql.Text != tt.want || !reflect.DeepEqual(sql.Args, tt.args) {
				t.Fatalf("cql2sql.SQL = %#v, want text %q args %#v", sql, tt.want, tt.args)
			}
		})
	}
}

func TestToSQLPredicateVariants(t *testing.T) {
	props := []cql2sql.Property{
		{Name: "name", Type: api.PropertyTypeString, Expr: cql2sql.Column("name")},
		{Name: "height", Type: api.PropertyTypeNumber, Expr: cql2sql.Column("height")},
	}
	defs := cql2sql.PropertyDefinitions(props...)
	tests := []struct {
		name string
		cql  string
		conf []string
		want string
		args []any
	}{
		{name: "between", cql: "height BETWEEN 1 AND 2", conf: []string{api.ConformanceAdvancedComparisonOperators}, want: `("height" BETWEEN CAST($1 AS numeric) AND CAST($2 AS numeric))`, args: []any{"1", "2"}},
		{name: "not between", cql: "height NOT BETWEEN 1 AND 2", conf: []string{api.ConformanceAdvancedComparisonOperators}, want: `("height" NOT BETWEEN CAST($1 AS numeric) AND CAST($2 AS numeric))`, args: []any{"1", "2"}},
		{name: "in", cql: "name IN ('x','y')", conf: []string{api.ConformanceAdvancedComparisonOperators}, want: `("name" IN ($1, $2))`, args: []any{"x", "y"}},
		{name: "not in", cql: "name NOT IN ('x','y')", conf: []string{api.ConformanceAdvancedComparisonOperators}, want: `("name" NOT IN ($1, $2))`, args: []any{"x", "y"}},
		{name: "not", cql: "NOT (name = 'x')", want: `(NOT (COALESCE(("name" = $1), FALSE)))`, args: []any{"x"}},
		{name: "or", cql: "name = 'x' OR height = 1", want: `((("name" = $1)) OR (("height" = CAST($2 AS numeric))))`, args: []any{"x", "1"}},
		{name: "integer division", cql: "height div 2 = 3", conf: []string{api.ConformanceArithmetic}, want: `(trunc(("height") / (CAST($1 AS numeric))) = CAST($2 AS numeric))`, args: []any{"2", "3"}},
		{name: "power", cql: "height ^ 2 = 4", conf: []string{api.ConformanceArithmetic}, want: `(power("height", CAST($1 AS numeric)) = CAST($2 AS numeric))`, args: []any{"2", "4"}},
		{name: "boolean literal", cql: "TRUE", want: `TRUE`},
		{name: "false literal", cql: "FALSE", want: `FALSE`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := gocql2.NewParser().WithAllowedProperties(defs...)
			if len(tt.conf) > 0 {
				parser.WithConformance(tt.conf...)
			}
			expr, err := parser.ParseText(tt.cql)
			if err != nil {
				t.Fatal(err)
			}
			sql, err := cql2sql.ToSQL(expr, cql2sql.PostGISDialect(), cql2sql.WithSQLProperties(props...))
			if err != nil {
				t.Fatal(err)
			}
			if sql.Text != tt.want || !reflect.DeepEqual(sql.Args, tt.args) {
				t.Fatalf("cql2sql.SQL = %#v, want text %q args %#v", sql, tt.want, tt.args)
			}
		})
	}
}

type testResolver func(*api.PropertyRef) (cql2sql.Expr, error)

func (r testResolver) ResolveProperty(ref *api.PropertyRef) (cql2sql.Expr, error) {
	return r(ref)
}

func TestToSQLBaseDialectAndMappingEdges(t *testing.T) {
	expr, err := gocql2.NewParser().WithAllowedProperties(api.PropertyDefinition{Name: "height", Type: api.PropertyTypeNumber}).ParseText("height = 1")
	if err != nil {
		t.Fatal(err)
	}
	sql, err := cql2sql.ToSQL(expr, cql2sql.BaseDialect{}, cql2sql.WithPropertyResolver(testResolver(func(ref *api.PropertyRef) (cql2sql.Expr, error) {
		if ref.Name == "height" {
			return cql2sql.Column("metrics", ref.Name), nil
		}
		return nil, cql2sql.ErrNoSQLMapping
	})))
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("metrics"."height" = CAST(? AS NUMERIC))` || !reflect.DeepEqual(sql.Args, []any{"1"}) {
		t.Fatalf("unexpected cql2sql.SQL: %#v", sql)
	}

	dateExpr, err := gocql2.NewParser().WithAllowedProperties(api.PropertyDefinition{Name: "event_date", Type: api.PropertyTypeDate}).ParseText("event_date = DATE('2022-01-01')")
	if err != nil {
		t.Fatal(err)
	}
	sql, err = cql2sql.ToSQL(dateExpr, nil, nil, cql2sql.WithDefaultColumnMapping())
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("event_date" = CAST(? AS DATE))` || !reflect.DeepEqual(sql.Args, []any{"2022-01-01"}) {
		t.Fatalf("unexpected cql2sql.SQL: %#v", sql)
	}

	timestampExpr, err := gocql2.NewParser().WithAllowedProperties(api.PropertyDefinition{Name: "event_time", Type: api.PropertyTypeTimestamp}).ParseText("event_time = TIMESTAMP('2022-01-01T00:00:00Z')")
	if err != nil {
		t.Fatal(err)
	}
	sql, err = cql2sql.ToSQL(timestampExpr, cql2sql.BaseDialect{}, cql2sql.WithDefaultColumnMapping())
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("event_time" = CAST(? AS TIMESTAMP))` || !reflect.DeepEqual(sql.Args, []any{"2022-01-01T00:00:00Z"}) {
		t.Fatalf("unexpected cql2sql.SQL: %#v", sql)
	}
}

func TestToSQLErrorsAndUnsupportedDialectHooks(t *testing.T) {
	if _, err := cql2sql.ToSQL(nil, nil); err == nil || !strings.Contains(err.Error(), "nil CQL2 expression") {
		t.Fatalf("expected nil expression error, got %v", err)
	}

	errorCases := []struct { //nolint:govet // Test table favors named fields over memory layout.
		parser func() *gocql2.Parser
		sql    []cql2sql.Option
		name   string
		cql    string
		match  string
	}{
		{name: "empty raw cql2sql.SQL", cql: "name = 'x'", parser: func() *gocql2.Parser {
			return gocql2.NewParser().WithAllowedProperties(api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString})
		}, sql: []cql2sql.Option{cql2sql.WithSQLProperties(cql2sql.Property{Name: "name", Type: api.PropertyTypeString, Expr: cql2sql.RawSQL(" ")})}, match: "raw SQL mapping"},
		{name: "empty column", cql: "name = 'x'", parser: func() *gocql2.Parser {
			return gocql2.NewParser().WithAllowedProperties(api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString})
		}, sql: []cql2sql.Option{cql2sql.WithSQLProperties(cql2sql.Property{Name: "name", Type: api.PropertyTypeString, Expr: cql2sql.Column()})}, match: "no identifier parts"},
		{name: "invalid identifier", cql: "name = 'x'", parser: func() *gocql2.Parser {
			return gocql2.NewParser().WithAllowedProperties(api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString})
		}, sql: []cql2sql.Option{cql2sql.WithSQLProperties(cql2sql.Property{Name: "name", Type: api.PropertyTypeString, Expr: cql2sql.Column("bad\x00name")})}, match: "invalid SQL identifier"},
		{name: "expression function", cql: "bool_fn()", parser: func() *gocql2.Parser {
			return gocql2.NewParser().WithAllowedFunctions(api.FunctionDefinition{Name: "bool_fn", Returns: []api.FunctionType{api.FunctionTypeBoolean}})
		}, match: `does not support function "bool_fn"`},
		{name: "scalar function", cql: "name = string_fn()", parser: func() *gocql2.Parser {
			return gocql2.NewParser().WithAllowedProperties(api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString}).WithAllowedFunctions(api.FunctionDefinition{Name: "string_fn", Returns: []api.FunctionType{api.FunctionTypeString}})
		}, sql: []cql2sql.Option{cql2sql.WithDefaultColumnMapping()}, match: `does not support function "string_fn"`},
		{name: "spatial predicate", cql: "S_INTERSECTS(geom, POINT(0 0))", parser: func() *gocql2.Parser {
			return gocql2.NewParser().WithConformance(api.ConformanceSpatialFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "geom", Type: api.PropertyTypeGeometry})
		}, sql: []cql2sql.Option{cql2sql.WithDefaultColumnMapping()}, match: "does not support spatial predicate"},
		{name: "temporal predicate", cql: "T_AFTER(event_time, TIMESTAMP('2022-01-01T00:00:00Z'))", parser: func() *gocql2.Parser {
			return gocql2.NewParser().WithConformance(api.ConformanceTemporalFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "event_time", Type: api.PropertyTypeTimestamp})
		}, sql: []cql2sql.Option{cql2sql.WithDefaultColumnMapping()}, match: "does not support temporal predicate"},
		{name: "array predicate", cql: "A_CONTAINS(tags, ('x'))", parser: func() *gocql2.Parser {
			return gocql2.NewParser().WithConformance(api.ConformanceArrayFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "tags", Type: api.PropertyTypeArray})
		}, sql: []cql2sql.Option{cql2sql.WithDefaultColumnMapping()}, match: "does not support array predicate"},
		{name: "geometry literal", cql: "POINT(0 0) IS NULL", match: "does not support geometry literal"},
	}
	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			parser := gocql2.NewParser()
			if tt.parser != nil {
				parser = tt.parser()
			}
			expr, err := parser.ParseText(tt.cql)
			if err != nil {
				t.Fatal(err)
			}
			_, err = cql2sql.ToSQL(expr, cql2sql.BaseDialect{}, tt.sql...)
			if err == nil || !strings.Contains(err.Error(), tt.match) {
				t.Fatalf("expected error containing %q, got %v", tt.match, err)
			}
		})
	}
}

func TestPostGISTemporalPredicateOperations(t *testing.T) {
	props := []cql2sql.Property{{Name: "event_time", Type: api.PropertyTypeTimestamp, Expr: cql2sql.Column("event_time")}}
	defs := cql2sql.PropertyDefinitions(props...)
	cases := []string{
		"T_BEFORE(event_time, TIMESTAMP('2022-01-01T00:00:00Z'))",
		"T_MEETS(INTERVAL('2021-01-01T00:00:00Z','2021-01-02T00:00:00Z'), INTERVAL('2021-01-02T00:00:00Z','2021-01-03T00:00:00Z'))",
		"T_METBY(INTERVAL('2021-01-02T00:00:00Z','2021-01-03T00:00:00Z'), INTERVAL('2021-01-01T00:00:00Z','2021-01-02T00:00:00Z'))",
		"T_DISJOINT(INTERVAL('2021-01-01T00:00:00Z','2021-01-02T00:00:00Z'), INTERVAL('2021-01-03T00:00:00Z','2021-01-04T00:00:00Z'))",
		"T_EQUALS(INTERVAL('2021-01-01T00:00:00Z','2021-01-02T00:00:00Z'), INTERVAL('2021-01-01T00:00:00Z','2021-01-02T00:00:00Z'))",
		"T_INTERSECTS(INTERVAL('2021-01-01T00:00:00Z','2021-01-03T00:00:00Z'), INTERVAL('2021-01-02T00:00:00Z','2021-01-04T00:00:00Z'))",
		"T_CONTAINS(INTERVAL('2021-01-01T00:00:00Z','2021-01-04T00:00:00Z'), INTERVAL('2021-01-02T00:00:00Z','2021-01-03T00:00:00Z'))",
		"T_DURING(INTERVAL('2021-01-02T00:00:00Z','2021-01-03T00:00:00Z'), INTERVAL('2021-01-01T00:00:00Z','2021-01-04T00:00:00Z'))",
		"T_FINISHEDBY(INTERVAL('2021-01-01T00:00:00Z','2021-01-04T00:00:00Z'), INTERVAL('2021-01-02T00:00:00Z','2021-01-04T00:00:00Z'))",
		"T_FINISHES(INTERVAL('2021-01-02T00:00:00Z','2021-01-04T00:00:00Z'), INTERVAL('2021-01-01T00:00:00Z','2021-01-04T00:00:00Z'))",
		"T_OVERLAPPEDBY(INTERVAL('2021-01-02T00:00:00Z','2021-01-04T00:00:00Z'), INTERVAL('2021-01-01T00:00:00Z','2021-01-03T00:00:00Z'))",
		"T_OVERLAPS(INTERVAL('2021-01-01T00:00:00Z','2021-01-03T00:00:00Z'), INTERVAL('2021-01-02T00:00:00Z','2021-01-04T00:00:00Z'))",
		"T_STARTEDBY(INTERVAL('2021-01-01T00:00:00Z','2021-01-04T00:00:00Z'), INTERVAL('2021-01-01T00:00:00Z','2021-01-03T00:00:00Z'))",
		"T_STARTS(INTERVAL('2021-01-01T00:00:00Z','2021-01-03T00:00:00Z'), INTERVAL('2021-01-01T00:00:00Z','2021-01-04T00:00:00Z'))",
	}
	for _, cql := range cases {
		t.Run(cql, func(t *testing.T) {
			expr, err := gocql2.NewParser().WithConformance(api.ConformanceTemporalFunctions).WithAllowedProperties(defs...).ParseText(cql)
			if err != nil {
				t.Fatal(err)
			}
			sql, err := cql2sql.ToSQL(expr, cql2sql.PostGISDialect(), cql2sql.WithSQLProperties(props...))
			if err != nil {
				t.Fatal(err)
			}
			if sql.Text == "" || len(sql.Args) == 0 {
				t.Fatalf("unexpected empty cql2sql.SQL: %#v", sql)
			}
		})
	}
}

func TestPostGISSpatialArrayAndGeometryVariants(t *testing.T) {
	props := []cql2sql.Property{
		{Name: "geom", Type: api.PropertyTypeGeometry, Expr: cql2sql.Column("geom")},
		{Name: "tags", Type: api.PropertyTypeArray, Expr: cql2sql.Column("tags")},
	}
	defs := cql2sql.PropertyDefinitions(props...)
	cases := []string{
		"S_CONTAINS(geom, POINT(0 0))",
		"S_CROSSES(geom, LINESTRING(0 0, 1 1))",
		"S_DISJOINT(geom, POLYGON((0 0, 1 0, 1 1, 0 0)))",
		"S_EQUALS(geom, MULTIPOINT((0 0),(1 1)))",
		"S_OVERLAPS(geom, MULTILINESTRING((0 0, 1 1),(2 2, 3 3)))",
		"S_TOUCHES(geom, MULTIPOLYGON(((0 0, 1 0, 1 1, 0 0))))",
		"S_WITHIN(geom, GEOMETRYCOLLECTION(POINT(0 0),LINESTRING(0 0, 1 1)))",
		"A_CONTAINEDBY(tags, ('red', 'blue'))",
		"A_EQUALS(tags, ('red', 'blue'))",
		"A_OVERLAPS(tags, ('red', 'blue'))",
	}
	for _, cql := range cases {
		t.Run(cql, func(t *testing.T) {
			expr, err := gocql2.NewParser().WithConformance(api.ConformanceSpatialFunctions, api.ConformanceArrayFunctions).WithAllowedProperties(defs...).ParseText(cql)
			if err != nil {
				t.Fatal(err)
			}
			sql, err := cql2sql.ToSQL(expr, cql2sql.PostGISDialect(cql2sql.WithSRID(3857)), cql2sql.WithSQLProperties(props...))
			if err != nil {
				t.Fatal(err)
			}
			if sql.Text == "" || len(sql.Args) == 0 {
				t.Fatalf("unexpected empty cql2sql.SQL: %#v", sql)
			}
		})
	}
}

func TestSQLGeometryLiteralErrorEdges(t *testing.T) {
	_, err := cql2sql.ToSQL(
		&api.SpatialPredicateExpression{
			Op:    api.SpatialOpIntersects,
			Left:  &api.PropertyRef{Name: "geom"},
			Right: &api.GeometryLiteral{Type: api.GeometryTypePoint, Coordinates: "bad"},
		},
		cql2sql.PostGISDialect(),
		cql2sql.WithDefaultColumnMapping(),
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported coordinate shape") {
		t.Fatalf("expected coordinate shape error, got %v", err)
	}
	_, err = cql2sql.ToSQL(&api.IsNullExpression{Expr: &api.TemporalUnbounded{}}, cql2sql.PostGISDialect())
	if err == nil || !strings.Contains(err.Error(), "standalone SQL expression") {
		t.Fatalf("expected temporal unbounded render error, got %v", err)
	}
}
