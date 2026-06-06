package parser

import (
	"testing"

	"github.com/cwygoda/gocql2/api"
)

func TestParseTextSpatialPredicates(t *testing.T) {
	cases := []struct {
		input string
		op    api.SpatialPredicateOp
	}{
		{input: `S_INTERSECTS(geom,BBOX(-180,-90,180,90))`, op: api.SpatialOpIntersects},
		{input: `S_INTERSECTS(geom,BBOX(-180,-90,0,180,90,100))`, op: api.SpatialOpIntersects},
		{input: `S_DISJOINT(geom,POINT(7.02 49.92))`, op: api.SpatialOpDisjoint},
		{input: `S_DISJOINT(geom,POINT Z (7.02 49.92 1))`, op: api.SpatialOpDisjoint},
		{input: `S_EQUALS(geom,LINESTRING(7 50 1,10 51 2))`, op: api.SpatialOpEquals},
		{input: `S_TOUCHES(geom,POLYGON((-180 -90,180 -90,180 90,-180 90,-180 -90)))`, op: api.SpatialOpTouches},
		{input: `S_CROSSES(geom,MULTIPOINT(7 50,10 51))`, op: api.SpatialOpCrosses},
		{input: `S_CROSSES(geom,MULTIPOINT((7 50),(10 51)))`, op: api.SpatialOpCrosses},
		{input: `S_WITHIN(geom,MULTILINESTRING((-180 -45,0 -45),(0 45,180 45)))`, op: api.SpatialOpWithin},
		{input: `S_CONTAINS(geom,MULTIPOLYGON(((-180 -90,-90 -90,-90 90,-180 90,-180 -90))))`, op: api.SpatialOpContains},
		{input: `S_OVERLAPS(geom,GEOMETRYCOLLECTION(POINT(7 50),POLYGON((0 0,10 0,10 10,0 10,0 0))))`, op: api.SpatialOpOverlaps},
	}
	parser := NewParser().WithConformance(api.ConformanceSpatialFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "geom", Type: api.PropertyTypeGeometry})
	for _, tc := range cases {
		expr, err := parser.ParseText(tc.input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", tc.input, err)
		}
		spatial, ok := expr.(*api.SpatialPredicateExpression)
		if !ok || spatial.Op != tc.op {
			t.Fatalf("%q parsed as %#v, want spatial predicate %s", tc.input, expr, tc.op)
		}
	}
}

func TestParseTextGeometryPreservesZCoordinates(t *testing.T) {
	parser := NewParser().WithConformance(api.ConformanceSpatialFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "geom", Type: api.PropertyTypeGeometry})
	expr, err := parser.ParseText(`S_EQUALS(geom,LINESTRING(7 50 1,10 51 2))`)
	if err != nil {
		t.Fatalf("ParseText() error = %v", err)
	}
	spatial, ok := expr.(*api.SpatialPredicateExpression)
	if !ok {
		t.Fatalf("ParseText() expression = %#v, want spatial predicate", expr)
	}
	geom, ok := spatial.Right.(*api.GeometryLiteral)
	if !ok {
		t.Fatalf("right operand = %#v, want geometry literal", spatial.Right)
	}
	coords, ok := geom.Coordinates.([]api.Coordinate)
	if !ok {
		t.Fatalf("coordinates = %#v, want coordinate slice", geom.Coordinates)
	}
	if got, want := coords[0], (api.Coordinate{X: 7, Y: 50, Z: 1, HasZ: true}); got != want {
		t.Fatalf("first coordinate = %#v, want %#v", got, want)
	}
	if got, want := coords[1], (api.Coordinate{X: 10, Y: 51, Z: 2, HasZ: true}); got != want {
		t.Fatalf("second coordinate = %#v, want %#v", got, want)
	}
}

func TestParseTextSpatialValidation(t *testing.T) {
	cases := map[string]string{
		`S_INTERSECTS(geom,POINT(90 180))`:                                       "latitude must be between -90 and 90",
		`S_INTERSECTS(geom,BBOX(10,-10,-10,10))`:                                 "minimum longitude",
		`S_INTERSECTS(geom,LINESTRING(7 50))`:                                    "at least 2 coordinates",
		`S_INTERSECTS(geom,POLYGON((0 0,10 0,10 10,0 10)))`:                      "linear ring must be closed",
		`S_INTERSECTS(name,POINT(7 50))`:                                         "cannot be used as a spatial operand",
		`S_INTERSECTS(geom,'POINT(7 50)')`:                                       "expected spatial operand",
		`S_INTERSECTS(geom,GEOMETRYCOLLECTION())`:                                "geometry collection must not be empty",
		`S_INTERSECTS(geom,POINT(1 - 2))`:                                        "expected number",
		`S_INTERSECTS(geom,MULTIPOINT(7 50,90 180))`:                             "latitude must be between -90 and 90",
		`S_INTERSECTS(geom,MULTIPOINT((7 50),(90 180)))`:                         "latitude must be between -90 and 90",
		`S_INTERSECTS(geom,GEOMETRYCOLLECTION(BBOX(-180,-90,180,90)))`:           "geometry collection cannot contain BBOX",
		`S_INTERSECTS(geom,GEOMETRYCOLLECTION(GEOMETRYCOLLECTION(POINT(7 50))))`: "geometry collection cannot contain GeometryCollection",
		`S_INTERSECTS(geom,POLYGON((0 0,10 0,10 10,0 10,0 0)),POINT(0 0))`:       "closing parenthesis",
		`S_INTERSECTS(geom,BBOX(-180,-90,0,180,90))`:                             "exactly four or six numbers",
		`S_INTERSECTS(geom,BBOX(-180,-90,100,180,90,0))`:                         "minimum elevation",
	}
	parser := NewParser().WithConformance(api.ConformanceSpatialFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "geom", Type: api.PropertyTypeGeometry},
		api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString})

	for input, want := range cases {
		_, err := parser.ParseText(input)
		assertParseErrorContains(t, err, want)
	}
}

func TestValidateGeometryLiteralLineStringRequiresTwoCoordinates(t *testing.T) {
	geom := &api.GeometryLiteral{
		Type:        api.GeometryTypeLineString,
		Coordinates: []api.Coordinate{{X: 7, Y: 50}},
		Src:         api.Span{Start: api.NoLocation(), End: api.NoLocation()},
	}
	err := validateGeometryLiteral(geom, api.LanguageText)
	assertParseErrorContains(t, err, "linestring requires at least two coordinates")
}

func TestParseJSONSpatialPredicates(t *testing.T) {
	cases := []struct {
		input string
		op    api.SpatialPredicateOp
	}{
		{input: `{"op":"s_intersects","args":[{"property":"geom"},{"bbox":[-180,-90,180,90]}]}`, op: api.SpatialOpIntersects},
		{input: `{"op":"s_intersects","args":[{"property":"geom"},{"bbox":[-180,-90,0,180,90,100]}]}`, op: api.SpatialOpIntersects},
		{input: `{"op":"s_disjoint","args":[{"property":"geom"},{"type":"Point","coordinates":[7.02,49.92]}]}`, op: api.SpatialOpDisjoint},
		{input: `{"op":"s_disjoint","args":[{"property":"geom"},{"type":"Point","coordinates":[7.02,49.92,1]}]}`, op: api.SpatialOpDisjoint},
		{input: `{"op":"s_equals","args":[{"property":"geom"},{"type":"LineString","coordinates":[[7,50,1],[10,51,2]]}]}`, op: api.SpatialOpEquals},
		{input: `{"op":"s_touches","args":[{"property":"geom"},{"type":"Polygon","coordinates":[[[-180,-90],[180,-90],[180,90],[-180,90],[-180,-90]]]}]}`, op: api.SpatialOpTouches},
		{input: `{"op":"s_crosses","args":[{"property":"geom"},{"type":"MultiPoint","coordinates":[[7,50],[10,51]]}]}`, op: api.SpatialOpCrosses},
		{input: `{"op":"s_within","args":[{"property":"geom"},{"type":"MultiLineString","coordinates":[[[-180,-45],[0,-45]],[[0,45],[180,45]]]}]}`, op: api.SpatialOpWithin},
		{input: `{"op":"s_contains","args":[{"property":"geom"},{"type":"MultiPolygon","coordinates":[[[[-180,-90],[-90,-90],[-90,90],[-180,90],[-180,-90]]]]}]}`, op: api.SpatialOpContains},
		{input: `{"op":"s_overlaps","args":[{"property":"geom"},{"type":"GeometryCollection","geometries":[{"type":"Point","coordinates":[7,50]},{"type":"Polygon","coordinates":[[[0,0],[10,0],[10,10],[0,10],[0,0]]]}]}]}`, op: api.SpatialOpOverlaps},
	}
	parser := NewParser().WithConformance(api.ConformanceSpatialFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "geom", Type: api.PropertyTypeGeometry})
	for _, tc := range cases {
		expr, err := parser.ParseJSON([]byte(tc.input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", tc.input, err)
		}
		spatial, ok := expr.(*api.SpatialPredicateExpression)
		if !ok || spatial.Op != tc.op {
			t.Fatalf("%s parsed as %#v, want spatial predicate %s", tc.input, expr, tc.op)
		}
	}
}

func TestParseJSONGeometryPreservesZCoordinates(t *testing.T) {
	parser := NewParser().WithConformance(api.ConformanceSpatialFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "geom", Type: api.PropertyTypeGeometry})
	expr, err := parser.ParseJSON([]byte(`{"op":"s_equals","args":[{"property":"geom"},{"type":"LineString","coordinates":[[7,50,1],[10,51,2]]}]}`))
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}
	spatial, ok := expr.(*api.SpatialPredicateExpression)
	if !ok {
		t.Fatalf("ParseJSON() expression = %#v, want spatial predicate", expr)
	}
	geom, ok := spatial.Right.(*api.GeometryLiteral)
	if !ok {
		t.Fatalf("right operand = %#v, want geometry literal", spatial.Right)
	}
	coords, ok := geom.Coordinates.([]api.Coordinate)
	if !ok {
		t.Fatalf("coordinates = %#v, want coordinate slice", geom.Coordinates)
	}
	if got, want := coords[0], (api.Coordinate{X: 7, Y: 50, Z: 1, HasZ: true}); got != want {
		t.Fatalf("first coordinate = %#v, want %#v", got, want)
	}
	if got, want := coords[1], (api.Coordinate{X: 10, Y: 51, Z: 2, HasZ: true}); got != want {
		t.Fatalf("second coordinate = %#v, want %#v", got, want)
	}
}

func TestParseJSONGeometryMatchesSchemaRules(t *testing.T) {
	parser := NewParser().WithConformance(api.ConformanceSpatialFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "geom", Type: api.PropertyTypeGeometry})

	accepted := []string{
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"Point","coordinates":[7,50],"foreign":"ok"}]}`,
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"MultiPoint","coordinates":[]}]}`,
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"Polygon","coordinates":[]}]}`,
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"MultiLineString","coordinates":[]}]}`,
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"MultiPolygon","coordinates":[]}]}`,
	}
	for _, input := range accepted {
		if _, err := parser.ParseJSON([]byte(input)); err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
	}

	rejected := map[string]string{
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"GeometryCollection","geometries":[{"type":"Point","coordinates":[7,50]}]}]}`:                                                                                                                          "at least two geometries",
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"GeometryCollection","geometries":[{"bbox":[-180,-90,180,90]},{"type":"Point","coordinates":[7,50]}]}]}`:                                                                                               "geometry collection cannot contain BBOX",
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"GeometryCollection","geometries":[{"type":"GeometryCollection","geometries":[{"type":"Point","coordinates":[7,50]},{"type":"Point","coordinates":[8,51]}]},{"type":"Point","coordinates":[7,50]}]}]}`: "geometry collection cannot contain GeometryCollection",
	}
	for input, want := range rejected {
		_, err := parser.ParseJSON([]byte(input))
		assertParseErrorContains(t, err, want)
	}
}

func TestParseJSONSpatialOpNamesAreCaseSensitive(t *testing.T) {
	cases := []string{
		`{"op":"S_INTERSECTS","args":[{"property":"geom"},{"type":"Point","coordinates":[7,50]}]}`,
		`{"op":"S_Intersects","args":[{"property":"geom"},{"type":"Point","coordinates":[7,50]}]}`,
	}
	parser := NewParser().WithAllowedProperties(api.PropertyDefinition{Name: "geom", Type: api.PropertyTypeGeometry}).WithConformance(api.ConformanceSpatialFunctions)

	for _, input := range cases {
		_, err := parser.ParseJSON([]byte(input))
		assertParseErrorContains(t, err, "unsupported reserved operation")
	}
}

func TestParseJSONGeoJSONValidation(t *testing.T) {
	cases := map[string]string{
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"Point","coordinates":[90,180]}]}`:                             "latitude must be between -90 and 90",
		`{"op":"s_intersects","args":[{"property":"geom"},{"bbox":[10,-10,-10,10]}]}`:                                            "minimum longitude",
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"Point","coordinates":[7,50,1,2]}]}`:                           "coordinate must have exactly two or three numbers",
		`{"op":"s_intersects","args":[{"property":"geom"},{"bbox":[-180,-90,0,180,90]}]}`:                                        "exactly four or six numbers",
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"Polygon","coordinates":[[[0,0],[10,0],[10,10],[0,10]]]}]}`:    "linear ring must be closed",
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"Feature","geometry":{"type":"Point","coordinates":[7,50]}}]}`: "unsupported GeoJSON geometry type",
		`{"op":"s_intersects","args":[{"property":"name"},{"type":"Point","coordinates":[7,50]}]}`:                               "cannot be used as a spatial operand",
		`{"op":"s_intersects","args":[{"property":"geom"},"POINT(7 50)"]}`:                                                       "expected spatial operand",
	}
	parser := NewParser().WithConformance(api.ConformanceSpatialFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "geom", Type: api.PropertyTypeGeometry},
		api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString})

	for input, want := range cases {
		_, err := parser.ParseJSON([]byte(input))
		assertParseErrorContains(t, err, want)
	}
}
