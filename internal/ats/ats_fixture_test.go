package ats

import (
	"testing"

	"github.com/cwygoda/gocql2/api"
)

// atsFixture is a dialect-neutral dataset for exercising the CQL2 Abstract Test
// Suite. Keep this file free of cql2sql.SQL snippets and database-specific types so the
// same records and expected result sets can be loaded into PostGIS, SQLite,
// in-memory evaluators, or future dialect adapters.
var atsFixture = atsFixtureDataSource{
	Name:        "ne_110m_populated_places_simple",
	Description: "Curated real-world populated-place records inspired by Natural Earth queryables.",
	Queryables: []atsFixtureQueryable{
		{Name: "name", Type: api.PropertyTypeString, Description: "Local place name, preserving accents where commonly used."},
		{Name: "name_ascii", Type: api.PropertyTypeString, Description: "ASCII/transliterated place name."},
		{Name: "country", Type: api.PropertyTypeString, Description: "Country name."},
		{Name: "capital", Type: api.PropertyTypeBoolean, Description: "Whether the place is a national capital."},
		{Name: "population", Type: api.PropertyTypeInteger, Description: "Approximate municipal/metropolitan population used as stable ATS data."},
		{Name: "elevation_m", Type: api.PropertyTypeNumber, Description: "Approximate elevation in metres above sea level; nullable for IS NULL tests."},
		{Name: "founded_date", Type: api.PropertyTypeDate, Description: "Commonly cited founding date, or first day of cited founding year when only a year is stable."},
		{Name: "last_updated", Type: api.PropertyTypeTimestamp, Description: "Stable fixture timestamp for temporal instant tests."},
		{Name: "record_start", Type: api.PropertyTypeTimestamp, Description: "Start timestamp of the fixture source-validity interval."},
		{Name: "record_end", Type: api.PropertyTypeTimestamp, Description: "End timestamp of the fixture source-validity interval."},
		{Name: "source_start", Type: api.PropertyTypeDate, Description: "Start date of the fixture source-validity interval."},
		{Name: "source_end", Type: api.PropertyTypeDate, Description: "End date of the fixture source-validity interval."},
		{Name: "geom", Type: api.PropertyTypeGeometry, Description: "Representative WGS84 point geometry."},
		{Name: "tags", Type: api.PropertyTypeArray, Description: "Real-world thematic tags such as capital, port, unesco, university, finance."},
	},
	Rows: []atsFixtureRow{
		{
			ID: "lodz",
			Values: map[string]any{
				"name": "Łódź", "name_ascii": "Lodz", "country": "Poland", "capital": false,
				"population": 670642, "elevation_m": 206.0, "founded_date": atsDate("1423-07-29"), "last_updated": atsTimestamp("2024-01-15T09:00:00Z"),
				"source_start": atsDate("2024-01-01"), "source_end": atsDate("2024-12-31"),
				"record_start": atsTimestamp("2024-01-01T00:00:00Z"), "record_end": atsTimestamp("2024-12-31T23:59:59Z"), "geom": atsPoint{Lon: 19.4550, Lat: 51.7592},
				"tags": []string{"industrial", "university", "film"},
			},
		},
		{
			ID: "krakow",
			Values: map[string]any{
				"name": "Kraków", "name_ascii": "Krakow", "country": "Poland", "capital": false,
				"population": 804237, "elevation_m": 219.0, "founded_date": atsDate("1257-06-05"), "last_updated": atsTimestamp("2024-01-20T09:00:00Z"),
				"source_start": atsDate("2024-01-01"), "source_end": atsDate("2024-12-31"),
				"record_start": atsTimestamp("2024-01-01T00:00:00Z"), "record_end": atsTimestamp("2024-12-31T23:59:59Z"), "geom": atsPoint{Lon: 19.9445, Lat: 50.0647},
				"tags": []string{"unesco", "university", "tourism"},
			},
		},
		{
			ID: "sao_paulo",
			Values: map[string]any{
				"name": "São Paulo", "name_ascii": "Sao Paulo", "country": "Brazil", "capital": false,
				"population": 12396372, "elevation_m": 760.0, "founded_date": atsDate("1554-01-25"), "last_updated": atsTimestamp("2024-02-01T12:00:00Z"),
				"source_start": atsDate("2024-01-01"), "source_end": atsDate("2024-12-31"),
				"record_start": atsTimestamp("2024-01-01T00:00:00Z"), "record_end": atsTimestamp("2024-12-31T23:59:59Z"), "geom": atsPoint{Lon: -46.6333, Lat: -23.5505},
				"tags": []string{"megacity", "finance", "university"},
			},
		},
		{
			ID: "munich",
			Values: map[string]any{
				"name": "München", "name_ascii": "Munich", "country": "Germany", "capital": false,
				"population": 1488202, "elevation_m": 519.0, "founded_date": atsDate("1158-06-14"), "last_updated": atsTimestamp("2024-02-10T08:30:00Z"),
				"source_start": atsDate("2024-01-01"), "source_end": atsDate("2024-12-31"),
				"record_start": atsTimestamp("2024-01-01T00:00:00Z"), "record_end": atsTimestamp("2024-12-31T23:59:59Z"), "geom": atsPoint{Lon: 11.5820, Lat: 48.1351},
				"tags": []string{"technology", "university", "rail"},
			},
		},
		{
			ID: "zurich",
			Values: map[string]any{
				"name": "Zürich", "name_ascii": "Zurich", "country": "Switzerland", "capital": false,
				"population": 421878, "elevation_m": 408.0, "founded_date": atsDate("1218-01-01"), "last_updated": atsTimestamp("2024-02-15T10:15:00Z"),
				"source_start": atsDate("2024-01-01"), "source_end": atsDate("2024-12-31"),
				"record_start": atsTimestamp("2024-01-01T00:00:00Z"), "record_end": atsTimestamp("2024-12-31T23:59:59Z"), "geom": atsPoint{Lon: 8.5417, Lat: 47.3769},
				"tags": []string{"finance", "university", "lake"},
			},
		},
		{
			ID: "montreal",
			Values: map[string]any{
				"name": "Montréal", "name_ascii": "Montreal", "country": "Canada", "capital": false,
				"population": 1762949, "elevation_m": 233.0, "founded_date": atsDate("1642-05-17"), "last_updated": atsTimestamp("2024-03-01T14:00:00Z"),
				"source_start": atsDate("2024-01-01"), "source_end": atsDate("2024-12-31"),
				"record_start": atsTimestamp("2024-01-01T00:00:00Z"), "record_end": atsTimestamp("2024-12-31T23:59:59Z"), "geom": atsPoint{Lon: -73.5673, Lat: 45.5017},
				"tags": []string{"port", "university", "island"},
			},
		},
		{
			ID: "bogota",
			Values: map[string]any{
				"name": "Bogotá", "name_ascii": "Bogota", "country": "Colombia", "capital": true,
				"population": 7968095, "elevation_m": 2640.0, "founded_date": atsDate("1538-08-06"), "last_updated": atsTimestamp("2024-03-10T11:45:00Z"),
				"source_start": atsDate("2024-01-01"), "source_end": atsDate("2024-12-31"),
				"record_start": atsTimestamp("2024-01-01T00:00:00Z"), "record_end": atsTimestamp("2024-12-31T23:59:59Z"), "geom": atsPoint{Lon: -74.0721, Lat: 4.7110},
				"tags": []string{"capital", "andes", "university"},
			},
		},
		{
			ID: "istanbul",
			Values: map[string]any{
				"name": "İstanbul", "name_ascii": "Istanbul", "country": "Turkey", "capital": false,
				"population": 15655924, "elevation_m": 39.0, "founded_date": atsDate("1453-05-29"), "last_updated": atsTimestamp("2024-03-20T16:20:00Z"),
				"source_start": atsDate("2024-01-01"), "source_end": atsDate("2024-12-31"),
				"record_start": atsTimestamp("2024-01-01T00:00:00Z"), "record_end": atsTimestamp("2024-12-31T23:59:59Z"), "geom": atsPoint{Lon: 28.9784, Lat: 41.0082},
				"tags": []string{"port", "historic", "transcontinental"},
			},
		},
		{
			ID: "luxembourg",
			Values: map[string]any{
				"name": "Luxembourg", "name_ascii": "Luxembourg", "country": "Luxembourg", "capital": true,
				"population": 132780, "elevation_m": 304.0, "founded_date": atsDate("963-01-01"), "last_updated": atsTimestamp("2024-04-01T07:00:00Z"),
				"source_start": atsDate("2024-01-01"), "source_end": atsDate("2024-12-31"),
				"record_start": atsTimestamp("2024-01-01T00:00:00Z"), "record_end": atsTimestamp("2024-12-31T23:59:59Z"), "geom": atsPoint{Lon: 6.1319, Lat: 49.6116},
				"tags": []string{"capital", "finance", "fortress", "unesco"},
			},
		},
		{
			ID: "alesund",
			Values: map[string]any{
				"name": "Ålesund", "name_ascii": "Alesund", "country": "Norway", "capital": false,
				"population": 67087, "elevation_m": nil, "founded_date": atsDate("1848-01-01"), "last_updated": atsTimestamp("2024-04-12T13:10:00Z"),
				"source_start": atsDate("2024-01-01"), "source_end": atsDate("2024-12-31"),
				"record_start": atsTimestamp("2024-01-01T00:00:00Z"), "record_end": atsTimestamp("2024-12-31T23:59:59Z"), "geom": atsPoint{Lon: 6.1495, Lat: 62.4722},
				"tags": []string{"port", "art_nouveau", "coastal"},
			},
		},
		{
			ID: "a_coruna",
			Values: map[string]any{
				"name": "A Coruña", "name_ascii": "A Coruna", "country": "Spain", "capital": false,
				"population": 245468, "elevation_m": 21.0, "founded_date": atsDate("1208-01-01"), "last_updated": atsTimestamp("2024-04-20T15:45:00Z"),
				"source_start": atsDate("2024-01-01"), "source_end": atsDate("2024-12-31"),
				"record_start": atsTimestamp("2024-01-01T00:00:00Z"), "record_end": atsTimestamp("2024-12-31T23:59:59Z"), "geom": atsPoint{Lon: -8.4115, Lat: 43.3623},
				"tags": []string{"port", "coastal", "lighthouse"},
			},
		},
	},
	Predicates: []atsFixturePredicate{
		{Conformance: api.ConformanceBasicCQL2, Filter: "country = 'Poland'", WantIDs: []string{"krakow", "lodz"}},
		{Conformance: api.ConformanceBasicCQL2, Filter: "capital = true", WantIDs: []string{"bogota", "luxembourg"}},
		{Conformance: api.ConformanceBasicCQL2, Filter: "population >= 10000000", WantIDs: []string{"istanbul", "sao_paulo"}},
		{Conformance: api.ConformanceBasicCQL2, Filter: "elevation_m IS NULL", WantIDs: []string{"alesund"}},
		{Conformance: api.ConformanceAdvancedComparisonOperators, Filter: "name_ascii LIKE 'M%'", WantIDs: []string{"montreal", "munich"}},
		{Conformance: api.ConformanceAdvancedComparisonOperators, Filter: "population BETWEEN 1000000 AND 2000000", WantIDs: []string{"montreal", "munich"}},
		{Conformance: api.ConformanceAdvancedComparisonOperators, Filter: "country IN ('Poland', 'Norway')", WantIDs: []string{"alesund", "krakow", "lodz"}},
		{Conformance: api.ConformanceCaseInsensitiveComparison, Filter: "CASEI(name_ascii) = casei('munich')", WantIDs: []string{"munich"}},
		{Conformance: api.ConformanceAccentInsensitiveComparison, Filter: "ACCENTI(name) = accenti('Lodz')", WantIDs: []string{"lodz"}},
		{Conformance: api.ConformanceAccentInsensitiveComparison, Filter: "ACCENTI(name) LIKE accenti('A%')", WantIDs: []string{"a_coruna", "alesund"}},
		{Conformance: api.ConformanceSpatialFunctions, Filter: "S_INTERSECTS(geom, BBOX(5,47,12,52))", WantIDs: []string{"luxembourg", "munich", "zurich"}},
		{Conformance: api.ConformanceTemporalFunctions, Filter: "T_AFTER(last_updated, TIMESTAMP('2024-03-01T00:00:00Z'))", WantIDs: []string{"a_coruna", "alesund", "bogota", "istanbul", "luxembourg", "montreal"}},
		{Conformance: api.ConformanceArrayFunctions, Filter: "A_OVERLAPS(tags, ('port','capital'))", WantIDs: []string{"a_coruna", "alesund", "bogota", "istanbul", "luxembourg", "montreal"}},
		{Conformance: api.ConformancePropertyProperty, Filter: "population > elevation_m", WantIDs: []string{"a_coruna", "bogota", "istanbul", "krakow", "lodz", "luxembourg", "montreal", "munich", "sao_paulo", "zurich"}},
		{Conformance: api.ConformanceArithmetic, Filter: "population / 1000000 >= 10", WantIDs: []string{"istanbul", "sao_paulo"}},
	},
}

type atsFixtureDataSource struct {
	Name        string
	Description string
	Queryables  []atsFixtureQueryable
	Rows        []atsFixtureRow
	Predicates  []atsFixturePredicate
}

type atsFixtureQueryable struct {
	Name        string
	Type        api.PropertyType
	Description string
}

//nolint:govet // Test fixture rows keep stable IDs before payload values for readability.
type atsFixtureRow struct {
	ID     string
	Values map[string]any
}

type atsFixturePredicate struct {
	Conformance string
	Filter      string
	WantIDs     []string
}

type atsDate string

type atsTimestamp string

type atsPoint struct {
	Lon float64
	Lat float64
}

func atsFixtureQueryablesOfTypes(types ...api.PropertyType) []atsFixtureQueryable {
	wanted := make(map[api.PropertyType]struct{}, len(types))
	for _, typ := range types {
		wanted[typ] = struct{}{}
	}

	queryables := []atsFixtureQueryable{}
	for _, queryable := range atsFixture.Queryables {
		if _, ok := wanted[queryable.Type]; ok {
			queryables = append(queryables, queryable)
		}
	}
	return queryables
}

func atsFixtureQueryablesByName() map[string]atsFixtureQueryable {
	queryables := make(map[string]atsFixtureQueryable, len(atsFixture.Queryables))
	for _, queryable := range atsFixture.Queryables {
		queryables[queryable.Name] = queryable
	}
	return queryables
}

func TestATSFixtureDataSourceMetadata(t *testing.T) {
	if atsFixture.Name != "ne_110m_populated_places_simple" {
		t.Fatalf("ATS fixture data source = %q", atsFixture.Name)
	}
	if len(atsFixture.Rows) == 0 {
		t.Fatal("ATS fixture has no rows")
	}

	queryables := atsFixtureQueryablesByName()
	for _, typ := range []api.PropertyType{
		api.PropertyTypeString,
		api.PropertyTypeBoolean,
		api.PropertyTypeNumber,
		api.PropertyTypeInteger,
		api.PropertyTypeDate,
		api.PropertyTypeTimestamp,
		api.PropertyTypeGeometry,
		api.PropertyTypeArray,
	} {
		if got := atsFixtureQueryablesOfTypes(typ); len(got) == 0 {
			t.Fatalf("ATS fixture has no queryable of type %q", typ)
		}
	}

	ids := map[string]struct{}{}
	for _, row := range atsFixture.Rows {
		if row.ID == "" {
			t.Fatal("ATS fixture row has empty ID")
		}
		if _, exists := ids[row.ID]; exists {
			t.Fatalf("duplicate ATS fixture row ID %q", row.ID)
		}
		ids[row.ID] = struct{}{}

		for name := range row.Values {
			if _, ok := queryables[name]; !ok {
				t.Fatalf("ATS fixture row %q has value for unknown queryable %q", row.ID, name)
			}
		}
		for name := range queryables {
			if _, ok := row.Values[name]; !ok {
				t.Fatalf("ATS fixture row %q missing queryable %q", row.ID, name)
			}
		}
	}

	for _, predicate := range atsFixture.Predicates {
		if predicate.Filter == "" {
			t.Fatal("ATS fixture predicate has empty filter")
		}
		for _, id := range predicate.WantIDs {
			if _, ok := ids[id]; !ok {
				t.Fatalf("ATS fixture predicate %q expects unknown row ID %q", predicate.Filter, id)
			}
		}
	}
}
