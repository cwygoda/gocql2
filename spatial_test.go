package gocql2

import "testing"

func TestParseTextSpatialPredicates(t *testing.T) {
	cases := []struct {
		input string
		op    SpatialPredicateOp
	}{
		{input: `S_INTERSECTS(geom,BBOX(-180,-90,180,90))`, op: SpatialOpIntersects},
		{input: `S_INTERSECTS(geom,BBOX(-180,-90,0,180,90,100))`, op: SpatialOpIntersects},
		{input: `S_DISJOINT(geom,POINT(7.02 49.92))`, op: SpatialOpDisjoint},
		{input: `S_DISJOINT(geom,POINT Z (7.02 49.92 1))`, op: SpatialOpDisjoint},
		{input: `S_EQUALS(geom,LINESTRING(7 50 1,10 51 2))`, op: SpatialOpEquals},
		{input: `S_TOUCHES(geom,POLYGON((-180 -90,180 -90,180 90,-180 90,-180 -90)))`, op: SpatialOpTouches},
		{input: `S_CROSSES(geom,MULTIPOINT(7 50,10 51))`, op: SpatialOpCrosses},
		{input: `S_WITHIN(geom,MULTILINESTRING((-180 -45,0 -45),(0 45,180 45)))`, op: SpatialOpWithin},
		{input: `S_CONTAINS(geom,MULTIPOLYGON(((-180 -90,-90 -90,-90 90,-180 90,-180 -90))))`, op: SpatialOpContains},
		{input: `S_OVERLAPS(geom,GEOMETRYCOLLECTION(POINT(7 50),POLYGON((0 0,10 0,10 10,0 10,0 0))))`, op: SpatialOpOverlaps},
	}
	parser := NewParser(WithAllowedProperties(PropertyDefinition{Name: "geom", Type: PropertyTypeGeometry}))
	for _, tc := range cases {
		expr, err := parser.ParseText(tc.input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", tc.input, err)
		}
		spatial, ok := expr.(*SpatialPredicateExpression)
		if !ok || spatial.Op != tc.op {
			t.Fatalf("%q parsed as %#v, want spatial predicate %s", tc.input, expr, tc.op)
		}
	}
}

func TestParseTextSpatialValidation(t *testing.T) {
	cases := map[string]string{
		`S_INTERSECTS(geom,POINT(90 180))`:                                 "latitude must be between -90 and 90",
		`S_INTERSECTS(geom,BBOX(10,-10,-10,10))`:                           "minimum longitude",
		`S_INTERSECTS(geom,LINESTRING(7 50))`:                              "at least 2 coordinates",
		`S_INTERSECTS(geom,POLYGON((0 0,10 0,10 10,0 10)))`:                "linear ring must be closed",
		`S_INTERSECTS(name,POINT(7 50))`:                                   "cannot be used as a spatial operand",
		`S_INTERSECTS(geom,'POINT(7 50)')`:                                 "expected spatial operand",
		`S_INTERSECTS(geom,GEOMETRYCOLLECTION())`:                          "geometry collection must not be empty",
		`S_INTERSECTS(geom,POINT(1 - 2))`:                                  "expected number",
		`S_INTERSECTS(geom,MULTIPOINT(7 50,90 180))`:                       "latitude must be between -90 and 90",
		`S_INTERSECTS(geom,POLYGON((0 0,10 0,10 10,0 10,0 0)),POINT(0 0))`: "closing parenthesis",
		`S_INTERSECTS(geom,BBOX(-180,-90,0,180,90))`:                       "exactly four or six numbers",
		`S_INTERSECTS(geom,BBOX(-180,-90,100,180,90,0))`:                   "minimum elevation",
	}
	parser := NewParser(WithAllowedProperties(
		PropertyDefinition{Name: "geom", Type: PropertyTypeGeometry},
		PropertyDefinition{Name: "name", Type: PropertyTypeString},
	))
	for input, want := range cases {
		_, err := parser.ParseText(input)
		assertParseErrorContains(t, err, want)
	}
}

func TestValidateGeometryLiteralLineStringRequiresTwoCoordinates(t *testing.T) {
	geom := &GeometryLiteral{
		Type:        GeometryTypeLineString,
		Coordinates: []Coordinate{{X: 7, Y: 50}},
		Src:         Span{Start: NoLocation(), End: NoLocation()},
	}
	err := validateGeometryLiteral(geom, LanguageText)
	assertParseErrorContains(t, err, "linestring requires at least two coordinates")
}

func TestParseJSONSpatialPredicates(t *testing.T) {
	cases := []struct {
		input string
		op    SpatialPredicateOp
	}{
		{input: `{"op":"s_intersects","args":[{"property":"geom"},{"bbox":[-180,-90,180,90]}]}`, op: SpatialOpIntersects},
		{input: `{"op":"s_intersects","args":[{"property":"geom"},{"bbox":[-180,-90,0,180,90,100]}]}`, op: SpatialOpIntersects},
		{input: `{"op":"s_disjoint","args":[{"property":"geom"},{"type":"Point","coordinates":[7.02,49.92]}]}`, op: SpatialOpDisjoint},
		{input: `{"op":"s_disjoint","args":[{"property":"geom"},{"type":"Point","coordinates":[7.02,49.92,1]}]}`, op: SpatialOpDisjoint},
		{input: `{"op":"s_equals","args":[{"property":"geom"},{"type":"LineString","coordinates":[[7,50,1],[10,51,2]]}]}`, op: SpatialOpEquals},
		{input: `{"op":"s_touches","args":[{"property":"geom"},{"type":"Polygon","coordinates":[[[-180,-90],[180,-90],[180,90],[-180,90],[-180,-90]]]}]}`, op: SpatialOpTouches},
		{input: `{"op":"s_crosses","args":[{"property":"geom"},{"type":"MultiPoint","coordinates":[[7,50],[10,51]]}]}`, op: SpatialOpCrosses},
		{input: `{"op":"s_within","args":[{"property":"geom"},{"type":"MultiLineString","coordinates":[[[-180,-45],[0,-45]],[[0,45],[180,45]]]}]}`, op: SpatialOpWithin},
		{input: `{"op":"s_contains","args":[{"property":"geom"},{"type":"MultiPolygon","coordinates":[[[[-180,-90],[-90,-90],[-90,90],[-180,90],[-180,-90]]]]}]}`, op: SpatialOpContains},
		{input: `{"op":"s_overlaps","args":[{"property":"geom"},{"type":"GeometryCollection","geometries":[{"type":"Point","coordinates":[7,50]},{"type":"Polygon","coordinates":[[[0,0],[10,0],[10,10],[0,10],[0,0]]]}]}]}`, op: SpatialOpOverlaps},
	}
	parser := NewParser(WithAllowedProperties(PropertyDefinition{Name: "geom", Type: PropertyTypeGeometry}))
	for _, tc := range cases {
		expr, err := parser.ParseJSON([]byte(tc.input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", tc.input, err)
		}
		spatial, ok := expr.(*SpatialPredicateExpression)
		if !ok || spatial.Op != tc.op {
			t.Fatalf("%s parsed as %#v, want spatial predicate %s", tc.input, expr, tc.op)
		}
	}
}

func TestParseJSONSpatialOpNamesAreCaseSensitive(t *testing.T) {
	cases := []string{
		`{"op":"S_INTERSECTS","args":[{"property":"geom"},{"type":"Point","coordinates":[7,50]}]}`,
		`{"op":"S_Intersects","args":[{"property":"geom"},{"type":"Point","coordinates":[7,50]}]}`,
	}
	parser := NewParser(
		WithAllowedProperties(PropertyDefinition{Name: "geom", Type: PropertyTypeGeometry}),
		WithConformance(ConformanceSpatialFunctions),
	)
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
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"Feature","geometry":{"type":"Point","coordinates":[7,50]}}]}`: "unexpected GeoJSON member",
		`{"op":"s_intersects","args":[{"property":"name"},{"type":"Point","coordinates":[7,50]}]}`:                               "cannot be used as a spatial operand",
		`{"op":"s_intersects","args":[{"property":"geom"},"POINT(7 50)"]}`:                                                       "expected spatial operand",
	}
	parser := NewParser(WithAllowedProperties(
		PropertyDefinition{Name: "geom", Type: PropertyTypeGeometry},
		PropertyDefinition{Name: "name", Type: PropertyTypeString},
	))
	for input, want := range cases {
		_, err := parser.ParseJSON([]byte(input))
		assertParseErrorContains(t, err, want)
	}
}
