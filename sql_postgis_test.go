package gocql2

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostGISDialectWithTestcontainers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PostGIS Testcontainers integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ctr, err := postgres.Run(
		ctx,
		"postgis/postgis:16-3.4",
		postgres.WithDatabase("cql2"),
		postgres.WithUsername("cql2"),
		postgres.WithPassword("cql2"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("PostGIS container unavailable: %v", err)
	}
	testcontainers.CleanupContainer(t, ctr)

	conn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("pgx", conn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("close database: %v", closeErr)
		}
	}()

	if err := setupPostGISFixture(ctx, db); err != nil {
		t.Fatal(err)
	}

	props := []SQLProperty{
		{Name: "name", Type: PropertyTypeString, Expr: Column("name")},
		{Name: "height", Type: PropertyTypeNumber, Expr: RawSQL("(properties->>'height')::numeric")},
		{Name: "active", Type: PropertyTypeBoolean, Expr: Column("active")},
		{Name: "event_time", Type: PropertyTypeTimestamp, Expr: Column("event_time")},
		{Name: "tags", Type: PropertyTypeArray, Expr: Column("tags")},
		{Name: "geom", Type: PropertyTypeGeometry, Expr: Column("geom")},
	}
	parseOpts := []ParseOption{
		WithConformance(
			ConformanceAdvancedComparisonOperators,
			ConformanceCaseInsensitiveComparison,
			ConformanceAccentInsensitiveComparison,
			ConformanceArithmetic,
			ConformanceTemporalFunctions,
			ConformanceArrayFunctions,
			ConformanceSpatialFunctions,
		),
		WithAllowedProperties(SQLPropertyDefinitions(props...)...),
	}
	sqlOpts := []SQLOption{WithSQLProperties(props...)}

	tests := []struct {
		name string
		cql  string
		want []string
	}{
		{name: "json alias and arithmetic", cql: "height + 1 > 11", want: []string{"b", "c"}},
		{name: "case insensitive", cql: "CASEI(name) = casei('CAFE')", want: []string{"b"}},
		{name: "accent insensitive", cql: "ACCENTI(CASEI(name)) = accenti(casei('cafe'))", want: []string{"a", "b"}},
		{name: "array overlap", cql: "A_OVERLAPS(tags, ('blue'))", want: []string{"a", "c"}},
		{name: "temporal after", cql: "T_AFTER(event_time, TIMESTAMP('2022-01-01T00:00:00Z'))", want: []string{"b", "c"}},
		{name: "spatial bbox", cql: "S_INTERSECTS(geom, BBOX(-1,-1,1,1))", want: []string{"a"}},
		{name: "geometry literal", cql: "S_INTERSECTS(geom, POINT(10 10))", want: []string{"b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := postGISQueryIDs(ctx, db, tt.cql, parseOpts, sqlOpts)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ids, tt.want) {
				t.Fatalf("ids for %q = %#v, want %#v", tt.cql, ids, tt.want)
			}
		})
	}
}

func setupPostGISFixture(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS postgis`,
		`CREATE EXTENSION IF NOT EXISTS unaccent`,
		`CREATE TABLE features (
			id text PRIMARY KEY,
			name text,
			active boolean,
			event_time timestamptz,
			tags text[],
			geom geometry(Geometry, 4326),
			properties jsonb
		)`,
		`INSERT INTO features (id, name, active, event_time, tags, geom, properties) VALUES
			('a', 'Café', true,  '2021-06-01T00:00:00Z', ARRAY['red','blue'],   ST_SetSRID(ST_Point(0, 0), 4326),  '{"height": 10}'),
			('b', 'CAFE', false, '2022-06-01T00:00:00Z', ARRAY['green'],      ST_SetSRID(ST_Point(10, 10), 4326), '{"height": 12}'),
			('c', 'Tea',  true,  '2023-06-01T00:00:00Z', ARRAY['blue'],       ST_SetSRID(ST_Point(50, 50), 4326), '{"height": 20}')`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func postGISQueryIDs(ctx context.Context, db *sql.DB, filter string, parseOpts []ParseOption, sqlOpts []SQLOption) ([]string, error) {
	expr, err := ParseText(filter, parseOpts...)
	if err != nil {
		return nil, err
	}
	where, err := ToSQL(expr, PostGISDialect(), sqlOpts...)
	if err != nil {
		return nil, err
	}
	// Generated SQL is a trusted package output; all user CQL2 literals are bind args.
	rows, err := db.QueryContext(ctx, "SELECT id FROM features WHERE "+where.Text+" ORDER BY id", where.Args...) //nolint:gosec
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, errors.Join(err, rows.Close())
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, errors.Join(err, rows.Close())
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return ids, nil
}
