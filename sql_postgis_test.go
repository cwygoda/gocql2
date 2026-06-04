package gocql2

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
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

	props := atsFixturePostGISSQLProperties()
	parseOpts := []ParseOption{
		WithConformance(
			ConformanceAdvancedComparisonOperators,
			ConformanceCaseInsensitiveComparison,
			ConformanceAccentInsensitiveComparison,
			ConformanceArithmetic,
			ConformanceTemporalFunctions,
			ConformanceArrayFunctions,
			ConformanceSpatialFunctions,
			ConformancePropertyProperty,
		),
		WithAllowedProperties(SQLPropertyDefinitions(props...)...),
	}
	sqlOpts := []SQLOption{WithSQLProperties(props...)}

	tests := []struct {
		name string
		cql  string
		want []string
	}{
		{name: "arithmetic over integer queryable", cql: "population / 1000000 >= 10", want: []string{"istanbul", "sao_paulo"}},
		{name: "case insensitive", cql: "CASEI(name_ascii) = casei('munich')", want: []string{"munich"}},
		{name: "accent insensitive", cql: "ACCENTI(name) = accenti('Lodz')", want: []string{"lodz"}},
		{name: "array overlap", cql: "A_OVERLAPS(tags, ('port','capital'))", want: []string{"a_coruna", "alesund", "bogota", "istanbul", "luxembourg", "montreal"}},
		{name: "temporal after", cql: "T_AFTER(last_updated, TIMESTAMP('2024-03-01T00:00:00Z'))", want: []string{"a_coruna", "alesund", "bogota", "istanbul", "luxembourg", "montreal"}},
		{name: "spatial bbox", cql: "S_INTERSECTS(geom, BBOX(5,47,12,52))", want: []string{"luxembourg", "munich", "zurich"}},
		{name: "geometry literal", cql: "S_INTERSECTS(geom, POINT(6.1319 49.6116))", want: []string{"luxembourg"}},
	}
	for _, predicate := range atsFixture.Predicates {
		tests = append(tests, struct {
			name string
			cql  string
			want []string
		}{name: "ATS fixture predicate " + predicate.Filter, cql: predicate.Filter, want: predicate.WantIDs})
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
		`DROP TABLE IF EXISTS features`,
		`CREATE TABLE features (
			id text PRIMARY KEY,
			name text,
			name_ascii text,
			country text,
			capital boolean,
			population integer,
			elevation_m numeric,
			founded_date date,
			last_updated timestamptz,
			record_start timestamptz,
			record_end timestamptz,
			source_start date,
			source_end date,
			geom geometry(Point, 4326),
			tags text[]
		)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	const insertSQL = `INSERT INTO features (
		id, name, name_ascii, country, capital, population, elevation_m,
		founded_date, last_updated, record_start, record_end, source_start, source_end, geom, tags
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7,
		$8::date, $9::timestamptz, $10::timestamptz, $11::timestamptz, $12::date, $13::date,
		ST_SetSRID(ST_Point($14, $15), 4326), $16::text[]
	)`
	for _, row := range atsFixture.Rows {
		point, ok := row.Values["geom"].(atsPoint)
		if !ok {
			return fmt.Errorf("ATS fixture row %q has geom %T, want atsPoint", row.ID, row.Values["geom"])
		}
		tags, ok := row.Values["tags"].([]string)
		if !ok {
			return fmt.Errorf("ATS fixture row %q has tags %T, want []string", row.ID, row.Values["tags"])
		}
		_, err := db.ExecContext(
			ctx,
			insertSQL,
			row.ID,
			row.Values["name"],
			row.Values["name_ascii"],
			row.Values["country"],
			row.Values["capital"],
			row.Values["population"],
			row.Values["elevation_m"],
			row.Values["founded_date"],
			row.Values["last_updated"],
			row.Values["record_start"],
			row.Values["record_end"],
			row.Values["source_start"],
			row.Values["source_end"],
			point.Lon,
			point.Lat,
			postGISTextArrayLiteral(tags),
		)
		if err != nil {
			return fmt.Errorf("insert ATS fixture row %q: %w", row.ID, err)
		}
	}
	return nil
}

func atsFixturePostGISSQLProperties() []SQLProperty {
	props := make([]SQLProperty, 0, len(atsFixture.Queryables))
	for _, queryable := range atsFixture.Queryables {
		props = append(props, SQLProperty{Name: queryable.Name, Type: queryable.Type, Expr: Column(queryable.Name)})
	}
	return props
}

func postGISTextArrayLiteral(items []string) string {
	quoted := make([]string, len(items))
	replacer := strings.NewReplacer(`\\`, `\\\\`, `"`, `\\"`)
	for i, item := range items {
		quoted[i] = `"` + replacer.Replace(item) + `"`
	}
	return `{` + strings.Join(quoted, `,`) + `}`
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
