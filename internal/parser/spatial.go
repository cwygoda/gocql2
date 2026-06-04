package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cwygoda/cql2/api"
)

var spatialPredicateOps = map[string]api.SpatialPredicateOp{
	"s_contains":   api.SpatialOpContains,
	"s_crosses":    api.SpatialOpCrosses,
	"s_disjoint":   api.SpatialOpDisjoint,
	"s_equals":     api.SpatialOpEquals,
	"s_intersects": api.SpatialOpIntersects,
	"s_overlaps":   api.SpatialOpOverlaps,
	"s_touches":    api.SpatialOpTouches,
	"s_within":     api.SpatialOpWithin,
}

func isSpatialPredicateOp(name string) (api.SpatialPredicateOp, bool) {
	op, ok := spatialPredicateOps[strings.ToLower(name)]
	return op, ok
}

func isJSONSpatialPredicateOp(name string) (api.SpatialPredicateOp, bool) {
	op, ok := spatialPredicateOps[name]
	return op, ok
}

func (p *textParser) parseSpatialPredicate(op api.SpatialPredicateOp, depth int) (*api.SpatialPredicateExpression, error) {
	nameTok := p.advance()
	if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
		return nil, err
	}
	left, err := p.parseSpatialOperand(depth + 1)
	if err != nil {
		return nil, err
	}
	if _, expectErr := p.expect(tokenComma, "comma"); expectErr != nil {
		return nil, expectErr
	}
	right, err := p.parseSpatialOperand(depth + 1)
	if err != nil {
		return nil, err
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	if err := validateSpatialPredicateOperands(left, right, api.LanguageText); err != nil {
		return nil, err
	}
	return &api.SpatialPredicateExpression{Op: op, Left: left, Right: right, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseSpatialOperand(depth int) (api.Node, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	if isGeometryKeyword(p.peek()) {
		return p.parseTextGeometryLiteral(depth + 1)
	}
	node, err := p.parseScalarPrimary(depth + 1)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func isGeometryKeyword(tok token) bool {
	if tok.kind != tokenKeyword {
		return false
	}
	switch tok.text {
	case "POINT", "LINESTRING", "POLYGON", "MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON", "GEOMETRYCOLLECTION", "BBOX":
		return true
	default:
		return false
	}
}

func (p *textParser) matchGeometryDimensionMarker() bool {
	tok := p.peek()
	if tok.kind != tokenIdentifier && tok.kind != tokenKeyword {
		return false
	}
	if !strings.EqualFold(tok.text, "Z") {
		return false
	}
	p.advance()
	return true
}

func (p *textParser) parseTextGeometryLiteral(depth int) (*api.GeometryLiteral, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	nameTok := p.advance()
	if nameTok.text != "BBOX" {
		p.matchGeometryDimensionMarker()
	}
	if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
		return nil, err
	}

	var geom *api.GeometryLiteral
	var err error
	switch nameTok.text {
	case "BBOX":
		geom, err = p.finishTextBBox(nameTok)
	case "POINT":
		geom, err = p.finishTextPoint(nameTok)
	case "LINESTRING":
		geom, err = p.finishTextLineString(nameTok, api.GeometryTypeLineString, 2)
	case "POLYGON":
		geom, err = p.finishTextPolygon(nameTok)
	case "MULTIPOINT":
		geom, err = p.finishTextMultiPoint(nameTok)
	case "MULTILINESTRING":
		geom, err = p.finishTextMultiLineString(nameTok)
	case "MULTIPOLYGON":
		geom, err = p.finishTextMultiPolygon(nameTok)
	case "GEOMETRYCOLLECTION":
		geom, err = p.finishTextGeometryCollection(nameTok, depth+1)
	default:
		err = parseError(api.LanguageText, nameTok.span.Start, "expected geometry literal")
	}
	if err != nil {
		return nil, err
	}
	return geom, nil
}

func (p *textParser) finishTextBBox(nameTok token) (*api.GeometryLiteral, error) {
	values := make([]float64, 0, 6)
	for {
		value, err := p.parseTextNumber()
		if err != nil {
			return nil, err
		}
		values = append(values, value)
		if !p.match(tokenComma, "") {
			break
		}
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	if err := validateBBox(values); err != nil {
		return nil, parseError(api.LanguageText, nameTok.span.Start, err.Error())
	}
	return &api.GeometryLiteral{Type: api.GeometryTypeBBox, BBox: values, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) finishTextPoint(nameTok token) (*api.GeometryLiteral, error) {
	coord, err := p.parseTextCoordinate()
	if err != nil {
		return nil, err
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	return &api.GeometryLiteral{Type: api.GeometryTypePoint, Coordinates: coord, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) finishTextLineString(nameTok token, typ api.GeometryType, minCoords int) (*api.GeometryLiteral, error) {
	coords, err := p.parseTextCoordinateList(minCoords)
	if err != nil {
		return nil, err
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	return &api.GeometryLiteral{Type: typ, Coordinates: coords, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) finishTextMultiPoint(nameTok token) (*api.GeometryLiteral, error) {
	coords := []api.Coordinate{}
	if p.at(tokenLParen, "") {
		for {
			if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
				return nil, err
			}
			coord, err := p.parseTextCoordinate()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(tokenRParen, "closing parenthesis"); err != nil {
				return nil, err
			}
			coords = append(coords, coord)
			if !p.match(tokenComma, "") {
				break
			}
		}
	} else {
		var err error
		coords, err = p.parseTextCoordinateList(1)
		if err != nil {
			return nil, err
		}
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	return &api.GeometryLiteral{Type: api.GeometryTypeMultiPoint, Coordinates: coords, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) finishTextPolygon(nameTok token) (*api.GeometryLiteral, error) {
	rings, err := p.parseTextRingList()
	if err != nil {
		return nil, err
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	return &api.GeometryLiteral{Type: api.GeometryTypePolygon, Coordinates: rings, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) finishTextMultiLineString(nameTok token) (*api.GeometryLiteral, error) {
	lines := [][]api.Coordinate{}
	for {
		if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
			return nil, err
		}
		line, err := p.parseTextCoordinateList(2)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenRParen, "closing parenthesis"); err != nil {
			return nil, err
		}
		lines = append(lines, line)
		if !p.match(tokenComma, "") {
			break
		}
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	return &api.GeometryLiteral{Type: api.GeometryTypeMultiLineString, Coordinates: lines, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) finishTextMultiPolygon(nameTok token) (*api.GeometryLiteral, error) {
	polygons := [][][]api.Coordinate{}
	for {
		if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
			return nil, err
		}
		rings, err := p.parseTextRingList()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenRParen, "closing parenthesis"); err != nil {
			return nil, err
		}
		polygons = append(polygons, rings)
		if !p.match(tokenComma, "") {
			break
		}
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	return &api.GeometryLiteral{Type: api.GeometryTypeMultiPolygon, Coordinates: polygons, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) finishTextGeometryCollection(nameTok token, depth int) (*api.GeometryLiteral, error) {
	if depth > p.cfg.MaxDepth {
		return nil, p.errorHere("maximum parse depth exceeded")
	}
	geoms := []*api.GeometryLiteral{}
	if p.at(tokenRParen, "") {
		return nil, parseError(api.LanguageText, p.peek().span.Start, "geometry collection must not be empty")
	}
	for {
		geom, err := p.parseTextGeometryLiteral(depth + 1)
		if err != nil {
			return nil, err
		}
		if err := validateGeometryCollectionChild(geom, api.LanguageText); err != nil {
			return nil, err
		}
		geoms = append(geoms, geom)
		if !p.match(tokenComma, "") {
			break
		}
	}
	end, err := p.expect(tokenRParen, "closing parenthesis")
	if err != nil {
		return nil, err
	}
	return &api.GeometryLiteral{Type: api.GeometryTypeGeometryCollection, Geometries: geoms, Src: api.Span{Start: nameTok.span.Start, End: end.span.End}}, nil
}

func (p *textParser) parseTextRingList() ([][]api.Coordinate, error) {
	rings := [][]api.Coordinate{}
	for {
		if _, err := p.expect(tokenLParen, "opening parenthesis"); err != nil {
			return nil, err
		}
		ring, err := p.parseTextCoordinateList(4)
		if err != nil {
			return nil, err
		}
		if err := validateLinearRing(ring); err != nil {
			return nil, parseError(api.LanguageText, p.previous().span.Start, err.Error())
		}
		if _, err := p.expect(tokenRParen, "closing parenthesis"); err != nil {
			return nil, err
		}
		rings = append(rings, ring)
		if !p.match(tokenComma, "") {
			break
		}
	}
	return rings, nil
}

func (p *textParser) parseTextCoordinateList(minCoords int) ([]api.Coordinate, error) {
	coords := []api.Coordinate{}
	for {
		coord, err := p.parseTextCoordinate()
		if err != nil {
			return nil, err
		}
		coords = append(coords, coord)
		if !p.match(tokenComma, "") {
			break
		}
	}
	if len(coords) < minCoords {
		return nil, parseError(api.LanguageText, p.previous().span.Start, fmt.Sprintf("geometry requires at least %d coordinates", minCoords))
	}
	return coords, nil
}

func (p *textParser) parseTextCoordinate() (api.Coordinate, error) {
	x, err := p.parseTextNumber()
	if err != nil {
		return api.Coordinate{}, err
	}
	y, err := p.parseTextNumber()
	if err != nil {
		return api.Coordinate{}, err
	}
	coord := api.Coordinate{X: x, Y: y}
	if p.atTextNumber() {
		z, err := p.parseTextNumber()
		if err != nil {
			return api.Coordinate{}, err
		}
		coord.Z = z
		coord.HasZ = true
	}
	if err := validateCoordinate(coord); err != nil {
		return api.Coordinate{}, parseError(api.LanguageText, p.previous().span.Start, err.Error())
	}
	return coord, nil
}

func (p *textParser) atTextNumber() bool {
	if p.at(tokenNumber, "") {
		return true
	}
	return p.peekSignedTextNumber()
}

func (p *textParser) peekSignedTextNumber() bool {
	if p.pos+1 >= len(p.tokens) {
		return false
	}
	sign := p.peek()
	number := p.tokens[p.pos+1]
	return sign.kind == tokenOperator && (sign.text == "+" || sign.text == "-") && number.kind == tokenNumber && sign.span.End.ByteOffset == number.span.Start.ByteOffset
}

func (p *textParser) parseTextNumber() (float64, error) {
	start := p.peek().span.Start
	sign := ""
	if p.peekSignedTextNumber() {
		sign = p.advance().text
	}
	tok, err := p.expect(tokenNumber, "number")
	if err != nil {
		return 0, err
	}
	value, parseErr := strconv.ParseFloat(sign+tok.text, 64)
	if parseErr != nil {
		return 0, parseError(api.LanguageText, start, "invalid numeric literal")
	}
	return value, nil
}

func validateSpatialPredicateOperands(left, right api.Node, source api.Language) error {
	if err := validateSpatialOperand(left, source); err != nil {
		return err
	}
	return validateSpatialOperand(right, source)
}

func validateSpatialOperand(node api.Node, source api.Language) error {
	switch value := node.(type) {
	case *api.GeometryLiteral:
		return validateGeometryLiteral(value, source)
	case *api.PropertyRef:
		if value.Type == api.PropertyTypeAny || isGeometryPropertyType(value.Type) {
			return nil
		}
		return parseError(source, value.Span().Start, fmt.Sprintf("property %q of type %q cannot be used as a spatial operand", value.Name, value.Type))
	case *api.FunctionCall:
		if functionCallReturns(value, api.FunctionTypeGeometry) || functionCallReturnsExact(value, api.FunctionTypeAny) {
			return nil
		}
		return parseError(source, value.Span().Start, fmt.Sprintf("function %q does not return geometry", value.Name))
	default:
		return parseError(source, node.Span().Start, "expected spatial operand", "geometry", "geometry property", "geometry function")
	}
}

func isGeometryPropertyType(typ api.PropertyType) bool {
	switch typ {
	case api.PropertyTypePoint, api.PropertyTypeMultiPoint, api.PropertyTypeLineString, api.PropertyTypeMultiLineString,
		api.PropertyTypePolygon, api.PropertyTypeMultiPolygon, api.PropertyTypeGeometry, api.PropertyTypeGeometryCollection:
		return true
	default:
		return false
	}
}

func validateGeometryLiteral(geom *api.GeometryLiteral, source api.Language) error {
	if geom == nil {
		return parseError(source, api.NoLocation(), "expected geometry literal")
	}
	switch geom.Type {
	case api.GeometryTypeBBox:
		return validateBBox(geom.BBox)
	case api.GeometryTypePoint:
		coord, ok := geom.Coordinates.(api.Coordinate)
		if !ok {
			return parseError(source, geom.Span().Start, "invalid point coordinates")
		}
		return validateCoordinate(coord)
	case api.GeometryTypeMultiPoint:
		coords, ok := geom.Coordinates.([]api.Coordinate)
		if !ok || (source == api.LanguageText && len(coords) == 0) {
			return parseError(source, geom.Span().Start, "invalid geometry coordinates")
		}
		for _, coord := range coords {
			if err := validateCoordinate(coord); err != nil {
				return parseError(source, geom.Span().Start, err.Error())
			}
		}
	case api.GeometryTypeLineString:
		coords, ok := geom.Coordinates.([]api.Coordinate)
		if !ok || len(coords) < 2 {
			return parseError(source, geom.Span().Start, "linestring requires at least two coordinates")
		}
		for _, coord := range coords {
			if err := validateCoordinate(coord); err != nil {
				return parseError(source, geom.Span().Start, err.Error())
			}
		}
	case api.GeometryTypePolygon:
		rings, ok := geom.Coordinates.([][]api.Coordinate)
		if !ok || (source == api.LanguageText && len(rings) == 0) {
			return parseError(source, geom.Span().Start, "polygon requires at least one ring")
		}
		for _, ring := range rings {
			if err := validateLinearRing(ring); err != nil {
				return parseError(source, geom.Span().Start, err.Error())
			}
		}
	case api.GeometryTypeMultiLineString:
		lines, ok := geom.Coordinates.([][]api.Coordinate)
		if !ok || (source == api.LanguageText && len(lines) == 0) {
			return parseError(source, geom.Span().Start, "multilinestring requires at least one line")
		}
		for _, line := range lines {
			if len(line) < 2 {
				return parseError(source, geom.Span().Start, "linestring requires at least two coordinates")
			}
			for _, coord := range line {
				if err := validateCoordinate(coord); err != nil {
					return parseError(source, geom.Span().Start, err.Error())
				}
			}
		}
	case api.GeometryTypeMultiPolygon:
		polygons, ok := geom.Coordinates.([][][]api.Coordinate)
		if !ok || (source == api.LanguageText && len(polygons) == 0) {
			return parseError(source, geom.Span().Start, "multipolygon requires at least one polygon")
		}
		for _, polygon := range polygons {
			if len(polygon) == 0 {
				return parseError(source, geom.Span().Start, "polygon requires at least one ring")
			}
			for _, ring := range polygon {
				if err := validateLinearRing(ring); err != nil {
					return parseError(source, geom.Span().Start, err.Error())
				}
			}
		}
	case api.GeometryTypeGeometryCollection:
		if len(geom.Geometries) == 0 || (source == api.LanguageJSON && len(geom.Geometries) < 2) {
			return parseError(source, geom.Span().Start, "geometry collection requires at least two geometries")
		}
		for _, child := range geom.Geometries {
			if err := validateGeometryCollectionChild(child, source); err != nil {
				return err
			}
			if err := validateGeometryLiteral(child, source); err != nil {
				return err
			}
		}
	default:
		return parseError(source, geom.Span().Start, fmt.Sprintf("unsupported geometry type %q", geom.Type))
	}
	return nil
}

func validateGeometryCollectionChild(geom *api.GeometryLiteral, source api.Language) error {
	if geom == nil {
		return parseError(source, api.NoLocation(), "expected geometry literal")
	}
	switch geom.Type {
	case api.GeometryTypeBBox:
		return parseError(source, geom.Span().Start, "geometry collection cannot contain BBOX")
	case api.GeometryTypeGeometryCollection:
		return parseError(source, geom.Span().Start, "geometry collection cannot contain GeometryCollection")
	default:
		return nil
	}
}

func validateCoordinate(coord api.Coordinate) error {
	if coord.X < -180 || coord.X > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	if coord.Y < -90 || coord.Y > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	return nil
}

func validateLinearRing(ring []api.Coordinate) error {
	if len(ring) < 4 {
		return fmt.Errorf("linear ring requires at least four coordinates")
	}
	for _, coord := range ring {
		if err := validateCoordinate(coord); err != nil {
			return err
		}
	}
	if ring[0] != ring[len(ring)-1] {
		return fmt.Errorf("linear ring must be closed")
	}
	return nil
}

func validateBBox(values []float64) error {
	if len(values) != 4 && len(values) != 6 {
		return fmt.Errorf("BBOX requires exactly four or six numbers")
	}
	maxOffset := 2
	if len(values) == 6 {
		maxOffset = 3
	}
	minCoord := api.Coordinate{X: values[0], Y: values[1]}
	maxCoord := api.Coordinate{X: values[maxOffset], Y: values[maxOffset+1]}
	if err := validateCoordinate(minCoord); err != nil {
		return err
	}
	if err := validateCoordinate(maxCoord); err != nil {
		return err
	}
	if minCoord.X > maxCoord.X {
		return fmt.Errorf("BBOX minimum longitude must not exceed maximum longitude")
	}
	if minCoord.Y > maxCoord.Y {
		return fmt.Errorf("BBOX minimum latitude must not exceed maximum latitude")
	}
	if len(values) == 6 && values[2] > values[5] {
		return fmt.Errorf("BBOX minimum elevation must not exceed maximum elevation")
	}
	return nil
}

func parseJSONSpatialPredicate(op api.SpatialPredicateOp, rawArgs []json.RawMessage, path api.JSONPath, depth int, cfg ParseConfig) (*api.SpatialPredicateExpression, error) {
	if len(rawArgs) != 2 {
		return nil, jsonPathError(path.Key("args"), "expected exactly 2 arguments")
	}
	left, err := parseJSONSpatialOperand(rawArgs[0], path.Key("args").Index(0), depth+1, cfg)
	if err != nil {
		return nil, err
	}
	right, err := parseJSONSpatialOperand(rawArgs[1], path.Key("args").Index(1), depth+1, cfg)
	if err != nil {
		return nil, err
	}
	if err := validateSpatialPredicateOperands(left, right, api.LanguageJSON); err != nil {
		return nil, err
	}
	return &api.SpatialPredicateExpression{Op: op, Left: left, Right: right, Src: jsonSpan(path)}, nil
}

func parseJSONSpatialOperand(raw json.RawMessage, path api.JSONPath, depth int, cfg ParseConfig) (api.Node, error) {
	if geom, err := parseJSONGeometryLiteral(raw, path, depth+1, cfg); err == nil {
		return geom, nil
	} else if hasJSONGeometryLiteralKey(raw, path) {
		return nil, err
	}
	node, err := parseJSONScalar(raw, path, depth+1, cfg)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func hasJSONGeometryLiteralKey(raw json.RawMessage, path api.JSONPath) bool {
	var obj rawObject
	if err := unmarshalAt(raw, path, &obj); err != nil {
		return false
	}
	_, hasType := obj["type"]
	_, hasCoordinates := obj["coordinates"]
	_, hasGeometries := obj["geometries"]
	_, hasBBox := obj["bbox"]
	return hasType || hasCoordinates || hasGeometries || hasBBox
}

func parseJSONGeometryLiteral(raw json.RawMessage, path api.JSONPath, depth int, cfg ParseConfig) (*api.GeometryLiteral, error) {
	if depth > cfg.MaxDepth {
		return nil, jsonPathError(path, "maximum parse depth exceeded")
	}
	var obj rawObject
	if err := unmarshalAt(raw, path, &obj); err != nil {
		return nil, err
	}

	if _, hasOp := obj["op"]; hasOp {
		return nil, jsonPathError(path, "expected geometry literal")
	}

	if rawBBox, hasBBox := obj["bbox"]; hasBBox && obj["type"] == nil {
		bbox, err := parseJSONBBox(rawBBox, path.Key("bbox"))
		if err != nil {
			return nil, err
		}
		return &api.GeometryLiteral{Type: api.GeometryTypeBBox, BBox: bbox, Src: jsonSpan(path)}, nil
	}

	rawType, hasType := obj["type"]
	if !hasType {
		return nil, jsonPathError(path.Key("type"), "missing GeoJSON type")
	}
	var typ string
	if err := unmarshalAt(rawType, path.Key("type"), &typ); err != nil {
		return nil, jsonPathError(path.Key("type"), "expected string GeoJSON type")
	}

	geom := &api.GeometryLiteral{Type: api.GeometryType(typ), Src: jsonSpan(path)}
	if rawBBox, hasBBox := obj["bbox"]; hasBBox {
		bbox, err := parseJSONBBox(rawBBox, path.Key("bbox"))
		if err != nil {
			return nil, err
		}
		geom.BBox = bbox
	}

	switch api.GeometryType(typ) {
	case api.GeometryTypePoint:
		coord, err := parseJSONCoordinate(obj["coordinates"], path.Key("coordinates"))
		if err != nil {
			return nil, err
		}
		geom.Coordinates = coord
	case api.GeometryTypeMultiPoint:
		coords, err := parseJSONCoordinateArray(obj["coordinates"], path.Key("coordinates"), 0)
		if err != nil {
			return nil, err
		}
		geom.Coordinates = coords
	case api.GeometryTypeLineString:
		coords, err := parseJSONCoordinateArray(obj["coordinates"], path.Key("coordinates"), 2)
		if err != nil {
			return nil, err
		}
		geom.Coordinates = coords
	case api.GeometryTypePolygon:
		rings, err := parseJSONPolygonCoordinates(obj["coordinates"], path.Key("coordinates"))
		if err != nil {
			return nil, err
		}
		geom.Coordinates = rings
	case api.GeometryTypeMultiLineString:
		lines, err := parseJSONMultiLineStringCoordinates(obj["coordinates"], path.Key("coordinates"))
		if err != nil {
			return nil, err
		}
		geom.Coordinates = lines
	case api.GeometryTypeMultiPolygon:
		polygons, err := parseJSONMultiPolygonCoordinates(obj["coordinates"], path.Key("coordinates"))
		if err != nil {
			return nil, err
		}
		geom.Coordinates = polygons
	case api.GeometryTypeGeometryCollection:
		rawGeoms, ok := obj["geometries"]
		if !ok {
			return nil, jsonPathError(path.Key("geometries"), "missing geometries")
		}
		var items []json.RawMessage
		if err := unmarshalAt(rawGeoms, path.Key("geometries"), &items); err != nil {
			return nil, jsonPathError(path.Key("geometries"), "expected array")
		}
		if len(items) < 2 {
			return nil, jsonPathError(path.Key("geometries"), "geometry collection requires at least two geometries")
		}
		geoms := make([]*api.GeometryLiteral, 0, len(items))
		for i, item := range items {
			childPath := path.Key("geometries").Index(i)
			child, err := parseJSONGeometryLiteral(item, childPath, depth+1, cfg)
			if err != nil {
				return nil, err
			}
			if err := validateGeometryCollectionChild(child, api.LanguageJSON); err != nil {
				return nil, err
			}
			geoms = append(geoms, child)
		}
		geom.Geometries = geoms
	default:
		return nil, jsonPathError(path.Key("type"), fmt.Sprintf("unsupported GeoJSON geometry type %q", typ))
	}

	if err := validateGeometryLiteral(geom, api.LanguageJSON); err != nil {
		return nil, err
	}
	return geom, nil
}

func requireJSONArray(raw json.RawMessage, path api.JSONPath, message string) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return jsonPathError(path, message)
	}
	return nil
}

func parseJSONBBox(raw json.RawMessage, path api.JSONPath) ([]float64, error) {
	var items []json.RawMessage
	if err := unmarshalAt(raw, path, &items); err != nil {
		return nil, jsonPathError(path, "expected bbox array")
	}
	if len(items) != 4 && len(items) != 6 {
		return nil, jsonPathError(path, "bbox requires exactly four or six numbers")
	}
	values := make([]float64, 0, len(items))
	for i, item := range items {
		value, err := parseJSONNumber(item, path.Index(i))
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := validateBBox(values); err != nil {
		return nil, jsonPathError(path, err.Error())
	}
	return values, nil
}

func parseJSONCoordinate(raw json.RawMessage, path api.JSONPath) (api.Coordinate, error) {
	if len(raw) == 0 {
		return api.Coordinate{}, jsonPathError(path, "missing coordinates")
	}
	if err := requireJSONArray(raw, path, "expected coordinate array"); err != nil {
		return api.Coordinate{}, err
	}
	var items []json.RawMessage
	if err := unmarshalAt(raw, path, &items); err != nil {
		return api.Coordinate{}, jsonPathError(path, "expected coordinate array")
	}
	if len(items) != 2 && len(items) != 3 {
		return api.Coordinate{}, jsonPathError(path, "coordinate must have exactly two or three numbers")
	}
	x, err := parseJSONNumber(items[0], path.Index(0))
	if err != nil {
		return api.Coordinate{}, err
	}
	y, err := parseJSONNumber(items[1], path.Index(1))
	if err != nil {
		return api.Coordinate{}, err
	}
	coord := api.Coordinate{X: x, Y: y}
	if len(items) == 3 {
		z, err := parseJSONNumber(items[2], path.Index(2))
		if err != nil {
			return api.Coordinate{}, err
		}
		coord.Z = z
		coord.HasZ = true
	}
	if err := validateCoordinate(coord); err != nil {
		return api.Coordinate{}, jsonPathError(path, err.Error())
	}
	return coord, nil
}

func parseJSONCoordinateArray(raw json.RawMessage, path api.JSONPath, minCoords int) ([]api.Coordinate, error) {
	if len(raw) == 0 {
		return nil, jsonPathError(path, "missing coordinates")
	}
	if err := requireJSONArray(raw, path, "expected coordinate array"); err != nil {
		return nil, err
	}
	var items []json.RawMessage
	if err := unmarshalAt(raw, path, &items); err != nil {
		return nil, jsonPathError(path, "expected coordinate array")
	}
	if len(items) < minCoords {
		return nil, jsonPathError(path, fmt.Sprintf("geometry requires at least %d coordinates", minCoords))
	}
	coords := make([]api.Coordinate, 0, len(items))
	for i, item := range items {
		coord, err := parseJSONCoordinate(item, path.Index(i))
		if err != nil {
			return nil, err
		}
		coords = append(coords, coord)
	}
	return coords, nil
}

func parseJSONPolygonCoordinates(raw json.RawMessage, path api.JSONPath) ([][]api.Coordinate, error) {
	if err := requireJSONArray(raw, path, "expected polygon coordinates"); err != nil {
		return nil, err
	}
	var items []json.RawMessage
	if err := unmarshalAt(raw, path, &items); err != nil {
		return nil, jsonPathError(path, "expected polygon coordinates")
	}
	rings := make([][]api.Coordinate, 0, len(items))
	for i, item := range items {
		ring, err := parseJSONCoordinateArray(item, path.Index(i), 4)
		if err != nil {
			return nil, err
		}
		if err := validateLinearRing(ring); err != nil {
			return nil, jsonPathError(path.Index(i), err.Error())
		}
		rings = append(rings, ring)
	}
	return rings, nil
}

func parseJSONMultiLineStringCoordinates(raw json.RawMessage, path api.JSONPath) ([][]api.Coordinate, error) {
	if err := requireJSONArray(raw, path, "expected multilinestring coordinates"); err != nil {
		return nil, err
	}
	var items []json.RawMessage
	if err := unmarshalAt(raw, path, &items); err != nil {
		return nil, jsonPathError(path, "expected multilinestring coordinates")
	}
	lines := make([][]api.Coordinate, 0, len(items))
	for i, item := range items {
		line, err := parseJSONCoordinateArray(item, path.Index(i), 2)
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func parseJSONMultiPolygonCoordinates(raw json.RawMessage, path api.JSONPath) ([][][]api.Coordinate, error) {
	if err := requireJSONArray(raw, path, "expected multipolygon coordinates"); err != nil {
		return nil, err
	}
	var items []json.RawMessage
	if err := unmarshalAt(raw, path, &items); err != nil {
		return nil, jsonPathError(path, "expected multipolygon coordinates")
	}
	polygons := make([][][]api.Coordinate, 0, len(items))
	for i, item := range items {
		polygon, err := parseJSONPolygonCoordinates(item, path.Index(i))
		if err != nil {
			return nil, err
		}
		polygons = append(polygons, polygon)
	}
	return polygons, nil
}

func parseJSONNumber(raw json.RawMessage, path api.JSONPath) (float64, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		return 0, jsonPathError(path, "invalid number")
	}
	number, ok := value.(json.Number)
	if !ok {
		return 0, jsonPathError(path, "expected number")
	}
	if _, err := canonicalNumber(number.String()); err != nil {
		return 0, jsonPathError(path, err.Error())
	}
	parsed, err := strconv.ParseFloat(number.String(), 64)
	if err != nil {
		return 0, jsonPathError(path, "invalid number")
	}
	return parsed, nil
}
