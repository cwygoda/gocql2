package gocql2

import (
	"reflect"
	"strings"
	"testing"
)

func TestToSQLScalarsAndPropertyAliases(t *testing.T) {
	props := []SQLProperty{
		{Name: "name", Type: PropertyTypeString, Expr: Column("f", "name")},
		{Name: "height", Type: PropertyTypeNumber, Expr: RawSQL("(properties->>'height')::numeric")},
		{Name: "active", Type: PropertyTypeBoolean, Expr: Column("active")},
	}
	defs := SQLPropertyDefinitions(props...)
	expr, err := ParseText(
		"CASEI(name) LIKE casei('foo%') AND height + 2 > 10 AND active IS NOT NULL",
		WithConformance(ConformanceAdvancedComparisonOperators, ConformanceCaseInsensitiveComparison, ConformanceArithmetic),
		WithAllowedProperties(defs...),
	)
	if err != nil {
		t.Fatal(err)
	}

	sql, err := ToSQL(expr, PostGISDialect(), WithSQLProperties(props...))
	if err != nil {
		t.Fatal(err)
	}

	wantText := `(((lower("f"."name") LIKE lower($1))) AND ((((properties->>'height')::numeric + CAST($2 AS numeric)) > CAST($3 AS numeric))) AND (("active" IS NOT NULL)))`
	if sql.Text != wantText {
		t.Fatalf("SQL text:\n got: %s\nwant: %s", sql.Text, wantText)
	}
	if !reflect.DeepEqual(sql.Args, []any{"foo%", "2", "10"}) {
		t.Fatalf("SQL args = %#v", sql.Args)
	}
}

func TestToSQLFailsClosedForUnmappedProperties(t *testing.T) {
	expr, err := ParseText("name = 'x'")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ToSQL(expr, PostGISDialect())
	if err == nil || !strings.Contains(err.Error(), "no SQL mapping") {
		t.Fatalf("expected missing mapping error, got %v", err)
	}
}

func TestToSQLDefaultColumnMapping(t *testing.T) {
	expr, err := ParseText("name = 'x'")
	if err != nil {
		t.Fatal(err)
	}
	sql, err := ToSQL(expr, PostGISDialect(), WithDefaultColumnMapping())
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("name" = $1)` || !reflect.DeepEqual(sql.Args, []any{"x"}) {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
}

func TestToSQLDefaultColumnMappingDoesNotHideResolverErrors(t *testing.T) {
	expr, err := ParseText("name = 'x'")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ToSQL(
		expr,
		PostGISDialect(),
		WithSQLProperties(SQLProperty{Name: "name", Type: PropertyTypeString}),
		WithDefaultColumnMapping(),
	)
	if err == nil || !strings.Contains(err.Error(), "has no expression") {
		t.Fatalf("expected explicit resolver error, got %v", err)
	}
}

func TestToSQLDefaultColumnMappingFallsBackOnlyForUnmappedProperties(t *testing.T) {
	expr, err := ParseText("name = 'x'")
	if err != nil {
		t.Fatal(err)
	}
	sql, err := ToSQL(
		expr,
		PostGISDialect(),
		WithSQLProperties(SQLProperty{Name: "other", Type: PropertyTypeString, Expr: Column("other")}),
		WithDefaultColumnMapping(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("name" = $1)` || !reflect.DeepEqual(sql.Args, []any{"x"}) {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
}

func TestToSQLSpatialPostGIS(t *testing.T) {
	props := []SQLProperty{{Name: "geom", Type: PropertyTypeGeometry, Expr: RawSQL("ST_Transform(raw_geom, 4326)")}}
	expr, err := ParseText(
		"S_INTERSECTS(geom, BBOX(-180,-90,180,90))",
		WithConformance(ConformanceSpatialFunctions),
		WithAllowedProperties(SQLPropertyDefinitions(props...)...),
	)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := ToSQL(expr, PostGISDialect(), WithSQLProperties(props...))
	if err != nil {
		t.Fatal(err)
	}
	want := "ST_Intersects(ST_Transform(raw_geom, 4326), ST_MakeEnvelope($1, $2, $3, $4, 4326))"
	if sql.Text != want {
		t.Fatalf("SQL text:\n got: %s\nwant: %s", sql.Text, want)
	}
	if !reflect.DeepEqual(sql.Args, []any{-180.0, -90.0, 180.0, 90.0}) {
		t.Fatalf("SQL args = %#v", sql.Args)
	}
}

func TestToSQLSpatialPostGISRejects3DBBox(t *testing.T) {
	props := []SQLProperty{{Name: "geom", Type: PropertyTypeGeometry, Expr: Column("geom")}}
	expr, err := ParseText(
		"S_INTERSECTS(geom, BBOX(-180,-90,0,180,90,100))",
		WithConformance(ConformanceSpatialFunctions),
		WithAllowedProperties(SQLPropertyDefinitions(props...)...),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ToSQL(expr, PostGISDialect(), WithSQLProperties(props...))
	if err == nil || !strings.Contains(err.Error(), "3D BBOX") {
		t.Fatalf("expected 3D BBOX unsupported error, got %v", err)
	}
}

func TestToSQLTemporalLiteralsAreDialectRendered(t *testing.T) {
	expr, err := ParseText(
		"event_time = TIMESTAMP('2022-04-14T14:48:46Z')",
		WithAllowedProperties(PropertyDefinition{Name: "event_time", Type: PropertyTypeTimestamp}),
	)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := ToSQL(
		expr,
		testTemporalDialect{},
		WithSQLProperties(SQLProperty{Name: "event_time", Type: PropertyTypeTimestamp, Expr: Column("event_time")}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("event_time" = temporal(?))` || !reflect.DeepEqual(sql.Args, []any{"2022-04-14T14:48:46Z"}) {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
}

type testTemporalDialect struct{ BaseDialect }

func (testTemporalDialect) RenderTemporalInstant(ctx SQLRenderContext, value *TemporalInstant) (SQLFragment, error) {
	return SQLFragment{Text: "temporal(" + ctx.AddArg(value.Value).Text + ")"}, nil
}

func TestToSQLTemporalAndArrays(t *testing.T) {
	props := []SQLProperty{
		{Name: "event_time", Type: PropertyTypeTimestamp, Expr: Column("event_time")},
		{Name: "tags", Type: PropertyTypeArray, Expr: Column("tags")},
	}
	defs := SQLPropertyDefinitions(props...)
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
			expr, err := ParseText(
				tt.cql,
				WithConformance(ConformanceTemporalFunctions, ConformanceArrayFunctions),
				WithAllowedProperties(defs...),
			)
			if err != nil {
				t.Fatal(err)
			}
			sql, err := ToSQL(expr, PostGISDialect(), WithSQLProperties(props...))
			if err != nil {
				t.Fatal(err)
			}
			if sql.Text != tt.want || !reflect.DeepEqual(sql.Args, tt.args) {
				t.Fatalf("SQL = %#v, want text %q args %#v", sql, tt.want, tt.args)
			}
		})
	}
}

func TestToSQLPredicateVariants(t *testing.T) {
	props := []SQLProperty{
		{Name: "name", Type: PropertyTypeString, Expr: Column("name")},
		{Name: "height", Type: PropertyTypeNumber, Expr: Column("height")},
	}
	defs := SQLPropertyDefinitions(props...)
	tests := []struct {
		name string
		cql  string
		conf []string
		want string
		args []any
	}{
		{name: "between", cql: "height BETWEEN 1 AND 2", conf: []string{ConformanceAdvancedComparisonOperators}, want: `("height" BETWEEN CAST($1 AS numeric) AND CAST($2 AS numeric))`, args: []any{"1", "2"}},
		{name: "not between", cql: "height NOT BETWEEN 1 AND 2", conf: []string{ConformanceAdvancedComparisonOperators}, want: `("height" NOT BETWEEN CAST($1 AS numeric) AND CAST($2 AS numeric))`, args: []any{"1", "2"}},
		{name: "in", cql: "name IN ('x','y')", conf: []string{ConformanceAdvancedComparisonOperators}, want: `("name" IN ($1, $2))`, args: []any{"x", "y"}},
		{name: "not in", cql: "name NOT IN ('x','y')", conf: []string{ConformanceAdvancedComparisonOperators}, want: `("name" NOT IN ($1, $2))`, args: []any{"x", "y"}},
		{name: "not", cql: "NOT (name = 'x')", want: `(NOT (COALESCE(("name" = $1), FALSE)))`, args: []any{"x"}},
		{name: "or", cql: "name = 'x' OR height = 1", want: `((("name" = $1)) OR (("height" = CAST($2 AS numeric))))`, args: []any{"x", "1"}},
		{name: "integer division", cql: "height div 2 = 3", conf: []string{ConformanceArithmetic}, want: `(trunc(("height") / (CAST($1 AS numeric))) = CAST($2 AS numeric))`, args: []any{"2", "3"}},
		{name: "power", cql: "height ^ 2 = 4", conf: []string{ConformanceArithmetic}, want: `(power("height", CAST($1 AS numeric)) = CAST($2 AS numeric))`, args: []any{"2", "4"}},
		{name: "boolean literal", cql: "TRUE", want: `TRUE`},
		{name: "false literal", cql: "FALSE", want: `FALSE`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []ParseOption{WithAllowedProperties(defs...)}
			if len(tt.conf) > 0 {
				opts = append(opts, WithConformance(tt.conf...))
			}
			expr, err := ParseText(tt.cql, opts...)
			if err != nil {
				t.Fatal(err)
			}
			sql, err := ToSQL(expr, PostGISDialect(), WithSQLProperties(props...))
			if err != nil {
				t.Fatal(err)
			}
			if sql.Text != tt.want || !reflect.DeepEqual(sql.Args, tt.args) {
				t.Fatalf("SQL = %#v, want text %q args %#v", sql, tt.want, tt.args)
			}
		})
	}
}

type testResolver func(*PropertyRef) (SQLExpr, error)

func (r testResolver) ResolveSQLProperty(ref *PropertyRef) (SQLExpr, error) { return r(ref) }

func TestToSQLBaseDialectAndMappingEdges(t *testing.T) {
	expr, err := ParseText("height = 1", WithAllowedProperties(PropertyDefinition{Name: "height", Type: PropertyTypeNumber}))
	if err != nil {
		t.Fatal(err)
	}
	sql, err := ToSQL(expr, BaseDialect{}, WithPropertyResolver(testResolver(func(ref *PropertyRef) (SQLExpr, error) {
		if ref.Name == "height" {
			return Column("metrics", ref.Name), nil
		}
		return nil, ErrNoSQLMapping
	})))
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("metrics"."height" = CAST(? AS NUMERIC))` || !reflect.DeepEqual(sql.Args, []any{"1"}) {
		t.Fatalf("unexpected SQL: %#v", sql)
	}

	dateExpr, err := ParseText("event_date = DATE('2022-01-01')", WithAllowedProperties(PropertyDefinition{Name: "event_date", Type: PropertyTypeDate}))
	if err != nil {
		t.Fatal(err)
	}
	sql, err = ToSQL(dateExpr, nil, nil, WithDefaultColumnMapping())
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("event_date" = CAST(? AS DATE))` || !reflect.DeepEqual(sql.Args, []any{"2022-01-01"}) {
		t.Fatalf("unexpected SQL: %#v", sql)
	}

	timestampExpr, err := ParseText("event_time = TIMESTAMP('2022-01-01T00:00:00Z')", WithAllowedProperties(PropertyDefinition{Name: "event_time", Type: PropertyTypeTimestamp}))
	if err != nil {
		t.Fatal(err)
	}
	sql, err = ToSQL(timestampExpr, BaseDialect{}, WithDefaultColumnMapping())
	if err != nil {
		t.Fatal(err)
	}
	if sql.Text != `("event_time" = CAST(? AS TIMESTAMP))` || !reflect.DeepEqual(sql.Args, []any{"2022-01-01T00:00:00Z"}) {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
}

func TestToSQLErrorsAndUnsupportedDialectHooks(t *testing.T) {
	if _, err := ToSQL(nil, nil); err == nil || !strings.Contains(err.Error(), "nil CQL2 expression") {
		t.Fatalf("expected nil expression error, got %v", err)
	}

	errorCases := []struct { //nolint:govet // Test table favors named fields over memory layout.
		opts  []ParseOption
		sql   []SQLOption
		name  string
		cql   string
		match string
	}{
		{name: "empty raw SQL", cql: "name = 'x'", opts: []ParseOption{WithAllowedProperties(PropertyDefinition{Name: "name", Type: PropertyTypeString})}, sql: []SQLOption{WithSQLProperties(SQLProperty{Name: "name", Type: PropertyTypeString, Expr: RawSQL(" ")})}, match: "raw SQL mapping"},
		{name: "empty column", cql: "name = 'x'", opts: []ParseOption{WithAllowedProperties(PropertyDefinition{Name: "name", Type: PropertyTypeString})}, sql: []SQLOption{WithSQLProperties(SQLProperty{Name: "name", Type: PropertyTypeString, Expr: Column()})}, match: "no identifier parts"},
		{name: "invalid identifier", cql: "name = 'x'", opts: []ParseOption{WithAllowedProperties(PropertyDefinition{Name: "name", Type: PropertyTypeString})}, sql: []SQLOption{WithSQLProperties(SQLProperty{Name: "name", Type: PropertyTypeString, Expr: Column("bad\x00name")})}, match: "invalid SQL identifier"},
		{name: "expression function", cql: "bool_fn()", opts: []ParseOption{WithAllowedFunctions(FunctionDefinition{Name: "bool_fn", Returns: []FunctionType{FunctionTypeBoolean}})}, match: `does not support function "bool_fn"`},
		{name: "scalar function", cql: "name = string_fn()", opts: []ParseOption{WithAllowedProperties(PropertyDefinition{Name: "name", Type: PropertyTypeString}), WithAllowedFunctions(FunctionDefinition{Name: "string_fn", Returns: []FunctionType{FunctionTypeString}})}, sql: []SQLOption{WithDefaultColumnMapping()}, match: `does not support function "string_fn"`},
		{name: "spatial predicate", cql: "S_INTERSECTS(geom, POINT(0 0))", opts: []ParseOption{WithConformance(ConformanceSpatialFunctions), WithAllowedProperties(PropertyDefinition{Name: "geom", Type: PropertyTypeGeometry})}, sql: []SQLOption{WithDefaultColumnMapping()}, match: "does not support spatial predicate"},
		{name: "temporal predicate", cql: "T_AFTER(event_time, TIMESTAMP('2022-01-01T00:00:00Z'))", opts: []ParseOption{WithConformance(ConformanceTemporalFunctions), WithAllowedProperties(PropertyDefinition{Name: "event_time", Type: PropertyTypeTimestamp})}, sql: []SQLOption{WithDefaultColumnMapping()}, match: "does not support temporal predicate"},
		{name: "array predicate", cql: "A_CONTAINS(tags, ('x'))", opts: []ParseOption{WithConformance(ConformanceArrayFunctions), WithAllowedProperties(PropertyDefinition{Name: "tags", Type: PropertyTypeArray})}, sql: []SQLOption{WithDefaultColumnMapping()}, match: "does not support array predicate"},
		{name: "geometry literal", cql: "POINT(0 0) IS NULL", match: "does not support geometry literal"},
	}
	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseText(tt.cql, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ToSQL(expr, BaseDialect{}, tt.sql...)
			if err == nil || !strings.Contains(err.Error(), tt.match) {
				t.Fatalf("expected error containing %q, got %v", tt.match, err)
			}
		})
	}
}

func TestPostGISTemporalPredicateOperations(t *testing.T) {
	props := []SQLProperty{{Name: "event_time", Type: PropertyTypeTimestamp, Expr: Column("event_time")}}
	defs := SQLPropertyDefinitions(props...)
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
			expr, err := ParseText(cql, WithConformance(ConformanceTemporalFunctions), WithAllowedProperties(defs...))
			if err != nil {
				t.Fatal(err)
			}
			sql, err := ToSQL(expr, PostGISDialect(), WithSQLProperties(props...))
			if err != nil {
				t.Fatal(err)
			}
			if sql.Text == "" || len(sql.Args) == 0 {
				t.Fatalf("unexpected empty SQL: %#v", sql)
			}
		})
	}
}

func TestPostGISSpatialArrayAndGeometryVariants(t *testing.T) {
	props := []SQLProperty{
		{Name: "geom", Type: PropertyTypeGeometry, Expr: Column("geom")},
		{Name: "tags", Type: PropertyTypeArray, Expr: Column("tags")},
	}
	defs := SQLPropertyDefinitions(props...)
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
			expr, err := ParseText(cql, WithConformance(ConformanceSpatialFunctions, ConformanceArrayFunctions), WithAllowedProperties(defs...))
			if err != nil {
				t.Fatal(err)
			}
			sql, err := ToSQL(expr, PostGISDialect(WithSRID(3857)), WithSQLProperties(props...))
			if err != nil {
				t.Fatal(err)
			}
			if sql.Text == "" || len(sql.Args) == 0 {
				t.Fatalf("unexpected empty SQL: %#v", sql)
			}
		})
	}
}

func TestSQLGeometryLiteralErrorEdges(t *testing.T) {
	if _, err := geometryLiteralGeoJSON(nil); err == nil || !strings.Contains(err.Error(), "nil geometry literal") {
		t.Fatalf("expected nil geometry literal error, got %v", err)
	}
	_, err := geometryLiteralGeoJSON(&GeometryLiteral{Type: GeometryTypePoint, Coordinates: "bad"})
	if err == nil || !strings.Contains(err.Error(), "unsupported coordinate shape") {
		t.Fatalf("expected coordinate shape error, got %v", err)
	}
	_, err = ToSQL(&IsNullExpression{Expr: &TemporalUnbounded{}}, PostGISDialect())
	if err == nil || !strings.Contains(err.Error(), "standalone SQL expression") {
		t.Fatalf("expected temporal unbounded render error, got %v", err)
	}
}
