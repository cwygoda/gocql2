package serializer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cwygoda/gocql2/api"
)

// ToText serializes a CQL2 AST to CQL2 Text.
func ToText(expr api.Expression) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("cannot serialize nil expression")
	}
	return textExpression(expr)
}

// ToJSON serializes a CQL2 AST to CQL2 JSON.
func ToJSON(expr api.Expression) ([]byte, error) {
	if expr == nil {
		return nil, fmt.Errorf("cannot serialize nil expression")
	}
	value, err := jsonExpression(expr)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func textExpression(expr api.Expression) (string, error) {
	switch n := expr.(type) {
	case *api.LogicalExpression:
		return textLogical(n)
	case *api.ComparisonExpression:
		left, err := textScalar(n.Left)
		if err != nil {
			return "", err
		}
		right, err := textScalar(n.Right)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %s %s", left, n.Op, right), nil
	case *api.LikeExpression:
		left, err := textScalar(n.Expr)
		if err != nil {
			return "", err
		}
		right, err := textScalar(n.Pattern)
		if err != nil {
			return "", err
		}
		op := "LIKE"
		if n.Not {
			op = "NOT LIKE"
		}
		return fmt.Sprintf("%s %s %s", left, op, right), nil
	case *api.BetweenExpression:
		value, err := textScalar(n.Expr)
		if err != nil {
			return "", err
		}
		lower, err := textScalar(n.Lower)
		if err != nil {
			return "", err
		}
		upper, err := textScalar(n.Upper)
		if err != nil {
			return "", err
		}
		not := ""
		if n.Not {
			not = " NOT"
		}
		return fmt.Sprintf("%s%s BETWEEN %s AND %s", value, not, lower, upper), nil
	case *api.InExpression:
		value, err := textScalar(n.Expr)
		if err != nil {
			return "", err
		}
		values := make([]string, len(n.Values))
		for i, item := range n.Values {
			values[i], err = textScalar(item)
			if err != nil {
				return "", err
			}
		}
		not := ""
		if n.Not {
			not = " NOT"
		}
		return fmt.Sprintf("%s%s IN (%s)", value, not, strings.Join(values, ", ")), nil
	case *api.IsNullExpression:
		value, err := textIsNullOperand(n.Expr)
		if err != nil {
			return "", err
		}
		not := ""
		if n.Not {
			not = " NOT"
		}
		return fmt.Sprintf("%s IS%s NULL", value, not), nil
	case *api.SpatialPredicateExpression:
		return textBinaryPredicate(string(n.Op), n.Left, n.Right)
	case *api.TemporalPredicateExpression:
		return textBinaryPredicate(string(n.Op), n.Left, n.Right)
	case *api.ArrayPredicateExpression:
		return textBinaryPredicate(string(n.Op), n.Left, n.Right)
	case *api.FunctionCall:
		if functionReturnsBoolean(n) {
			return textFunction(n)
		}
		return "", fmt.Errorf("function %q is not known to return boolean", n.Name)
	case *api.Literal:
		if n.Kind == api.LiteralBool {
			return textLiteral(n)
		}
		return "", fmt.Errorf("%s literal is not a CQL2 Text expression", n.Kind)
	default:
		return "", fmt.Errorf("unsupported expression node %T", expr)
	}
}

func textLogical(n *api.LogicalExpression) (string, error) {
	switch n.Op {
	case api.LogicalNot:
		if len(n.Args) != 1 {
			return "", fmt.Errorf("NOT expression has %d arguments, want 1", len(n.Args))
		}
		arg, err := textExpression(n.Args[0])
		if err != nil {
			return "", err
		}
		return "NOT (" + arg + ")", nil
	case api.LogicalAnd, api.LogicalOr:
		if len(n.Args) == 0 {
			return "", fmt.Errorf("%s expression has no arguments", n.Op)
		}
		parts := make([]string, len(n.Args))
		for i, arg := range n.Args {
			part, err := textExpression(arg)
			if err != nil {
				return "", err
			}
			parts[i] = "(" + part + ")"
		}
		return strings.Join(parts, " "+strings.ToUpper(string(n.Op))+" "), nil
	default:
		return "", fmt.Errorf("unsupported logical operator %q", n.Op)
	}
}

func textIsNullOperand(node api.Node) (string, error) {
	text, err := textNode(node)
	if err != nil {
		return "", err
	}
	if _, isScalar := node.(api.ScalarExpression); isScalar {
		return text, nil
	}
	if _, isExpr := node.(api.Expression); isExpr {
		return "(" + text + ")", nil
	}
	return text, nil
}

func textBinaryPredicate(name string, left, right api.Node) (string, error) {
	l, err := textNode(left)
	if err != nil {
		return "", err
	}
	r, err := textNode(right)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s(%s, %s)", strings.ToUpper(name), l, r), nil
}

func textNode(node api.Node) (string, error) {
	switch n := node.(type) {
	case api.ScalarExpression:
		return textScalar(n)
	case api.Expression:
		return textExpression(n)
	case *api.TemporalUnbounded:
		return quoteString(".."), nil
	case *api.TemporalInterval:
		return textTemporalInterval(n)
	case *api.GeometryLiteral:
		return textGeometry(n)
	default:
		return "", fmt.Errorf("unsupported node %T", node)
	}
}

func textScalar(scalar api.ScalarExpression) (string, error) {
	switch n := scalar.(type) {
	case *api.PropertyRef:
		return quoteIdentifier(n.Name)
	case *api.Literal:
		return textLiteral(n)
	case *api.FunctionCall:
		return textFunction(n)
	case *api.ArithmeticExpression:
		left, err := textScalar(n.Left)
		if err != nil {
			return "", err
		}
		right, err := textScalar(n.Right)
		if err != nil {
			return "", err
		}
		op := string(n.Op)
		if n.Op == api.ArithmeticIntDiv {
			op = "DIV"
		}
		return fmt.Sprintf("(%s %s %s)", left, op, right), nil
	case *api.TemporalInstant:
		return textTemporalInstant(n)
	case *api.ArrayLiteral:
		return textArray(n)
	default:
		return "", fmt.Errorf("unsupported scalar node %T", scalar)
	}
}

func textLiteral(n *api.Literal) (string, error) {
	switch n.Kind {
	case api.LiteralString:
		value, ok := n.Value.(string)
		if !ok {
			return "", fmt.Errorf("string literal value has type %T", n.Value)
		}
		return quoteString(value), nil
	case api.LiteralNumber:
		return numberString(n.Value)
	case api.LiteralBool:
		value, ok := n.Value.(bool)
		if !ok {
			return "", fmt.Errorf("bool literal value has type %T", n.Value)
		}
		if value {
			return "TRUE", nil
		}
		return "FALSE", nil
	case api.LiteralNull:
		return "NULL", nil
	default:
		return "", fmt.Errorf("unsupported literal kind %q", n.Kind)
	}
}

func textFunction(n *api.FunctionCall) (string, error) {
	name, err := textFunctionName(n.Name)
	if err != nil {
		return "", err
	}
	args := make([]string, len(n.Args))
	for i, arg := range n.Args {
		if array, ok := arg.(*api.ArrayLiteral); ok && len(array.Values) == 1 {
			return "", fmt.Errorf("single-element array function arguments cannot be represented unambiguously as CQL2 Text")
		}
		args[i], err = textNode(arg)
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s(%s)", name, strings.Join(args, ", ")), nil
}

func textTemporalInstant(n *api.TemporalInstant) (string, error) {
	switch n.Kind {
	case api.TemporalInstantDate:
		return "DATE(" + quoteString(n.Value) + ")", nil
	case api.TemporalInstantTimestamp:
		return "TIMESTAMP(" + quoteString(n.Value) + ")", nil
	default:
		return "", fmt.Errorf("unsupported temporal instant kind %q", n.Kind)
	}
}

func textTemporalInterval(n *api.TemporalInterval) (string, error) {
	start, err := textTemporalIntervalEndpoint(n.Start)
	if err != nil {
		return "", err
	}
	end, err := textTemporalIntervalEndpoint(n.End)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("INTERVAL(%s, %s)", start, end), nil
}

func textTemporalIntervalEndpoint(node api.Node) (string, error) {
	switch n := node.(type) {
	case *api.TemporalUnbounded:
		return quoteString(".."), nil
	case *api.TemporalInstant:
		return quoteString(n.Value), nil
	case api.ScalarExpression:
		return textScalar(n)
	default:
		return "", fmt.Errorf("unsupported interval endpoint %T", node)
	}
}

func textArray(n *api.ArrayLiteral) (string, error) {
	values := make([]string, len(n.Values))
	for i, value := range n.Values {
		text, err := textNode(value)
		if err != nil {
			return "", err
		}
		values[i] = text
	}
	return "(" + strings.Join(values, ", ") + ")", nil
}

func textGeometry(n *api.GeometryLiteral) (string, error) {
	switch n.Type {
	case api.GeometryTypeBBox:
		return "BBOX(" + joinFloat64(n.BBox, ", ") + ")", nil
	case api.GeometryTypePoint:
		coord, ok := n.Coordinates.(api.Coordinate)
		if !ok {
			return "", fmt.Errorf("point coordinates have type %T", n.Coordinates)
		}
		return "POINT(" + textCoordinate(coord) + ")", nil
	case api.GeometryTypeMultiPoint:
		coords, ok := n.Coordinates.([]api.Coordinate)
		if !ok {
			return "", fmt.Errorf("multipoint coordinates have type %T", n.Coordinates)
		}
		return "MULTIPOINT(" + textCoordinateList(coords) + ")", nil
	case api.GeometryTypeLineString:
		coords, ok := n.Coordinates.([]api.Coordinate)
		if !ok {
			return "", fmt.Errorf("linestring coordinates have type %T", n.Coordinates)
		}
		return "LINESTRING(" + textCoordinateList(coords) + ")", nil
	case api.GeometryTypePolygon:
		rings, ok := n.Coordinates.([][]api.Coordinate)
		if !ok {
			return "", fmt.Errorf("polygon coordinates have type %T", n.Coordinates)
		}
		return "POLYGON(" + textRingList(rings) + ")", nil
	case api.GeometryTypeMultiLineString:
		lines, ok := n.Coordinates.([][]api.Coordinate)
		if !ok {
			return "", fmt.Errorf("multilinestring coordinates have type %T", n.Coordinates)
		}
		parts := make([]string, len(lines))
		for i, line := range lines {
			parts[i] = "(" + textCoordinateList(line) + ")"
		}
		return "MULTILINESTRING(" + strings.Join(parts, ", ") + ")", nil
	case api.GeometryTypeMultiPolygon:
		polygons, ok := n.Coordinates.([][][]api.Coordinate)
		if !ok {
			return "", fmt.Errorf("multipolygon coordinates have type %T", n.Coordinates)
		}
		parts := make([]string, len(polygons))
		for i, polygon := range polygons {
			parts[i] = "(" + textRingList(polygon) + ")"
		}
		return "MULTIPOLYGON(" + strings.Join(parts, ", ") + ")", nil
	case api.GeometryTypeGeometryCollection:
		parts := make([]string, len(n.Geometries))
		for i, geom := range n.Geometries {
			text, err := textGeometry(geom)
			if err != nil {
				return "", err
			}
			parts[i] = text
		}
		return "GEOMETRYCOLLECTION(" + strings.Join(parts, ", ") + ")", nil
	default:
		return "", fmt.Errorf("unsupported geometry type %q", n.Type)
	}
}

func textRingList(rings [][]api.Coordinate) string {
	parts := make([]string, len(rings))
	for i, ring := range rings {
		parts[i] = "(" + textCoordinateList(ring) + ")"
	}
	return strings.Join(parts, ", ")
}

func textCoordinateList(coords []api.Coordinate) string {
	parts := make([]string, len(coords))
	for i, coord := range coords {
		parts[i] = textCoordinate(coord)
	}
	return strings.Join(parts, ", ")
}

func textCoordinate(coord api.Coordinate) string {
	parts := []string{formatFloat(coord.X), formatFloat(coord.Y)}
	if coord.HasZ {
		parts = append(parts, formatFloat(coord.Z))
	}
	return strings.Join(parts, " ")
}

func jsonExpression(expr api.Expression) (any, error) {
	switch n := expr.(type) {
	case *api.LogicalExpression:
		return jsonLogical(n)
	case *api.ComparisonExpression:
		return jsonOp(string(n.Op), n.Left, n.Right)
	case *api.LikeExpression:
		value, err := jsonOp("like", n.Expr, n.Pattern)
		if err != nil || !n.Not {
			return value, err
		}
		return map[string]any{"op": "not", "args": []any{value}}, nil
	case *api.BetweenExpression:
		value, err := jsonOp("between", n.Expr, n.Lower, n.Upper)
		if err != nil || !n.Not {
			return value, err
		}
		return map[string]any{"op": "not", "args": []any{value}}, nil
	case *api.InExpression:
		exprValue, err := jsonScalar(n.Expr)
		if err != nil {
			return nil, err
		}
		values := make([]any, len(n.Values))
		for i, item := range n.Values {
			values[i], err = jsonScalar(item)
			if err != nil {
				return nil, err
			}
		}
		value := map[string]any{"op": "in", "args": []any{exprValue, values}}
		if !n.Not {
			return value, nil
		}
		return map[string]any{"op": "not", "args": []any{value}}, nil
	case *api.IsNullExpression:
		value, err := jsonOp("isNull", n.Expr)
		if err != nil || !n.Not {
			return value, err
		}
		return map[string]any{"op": "not", "args": []any{value}}, nil
	case *api.SpatialPredicateExpression:
		return jsonOp(string(n.Op), n.Left, n.Right)
	case *api.TemporalPredicateExpression:
		return jsonOp(jsonTemporalOp(n.Op), n.Left, n.Right)
	case *api.ArrayPredicateExpression:
		return jsonOp(jsonArrayOp(n.Op), n.Left, n.Right)
	case *api.FunctionCall:
		if functionReturnsBoolean(n) {
			return jsonFunction(n)
		}
		return nil, fmt.Errorf("function %q is not known to return boolean", n.Name)
	case *api.Literal:
		if n.Kind == api.LiteralBool {
			return jsonLiteral(n)
		}
		return nil, fmt.Errorf("%s literal is not a CQL2 JSON expression", n.Kind)
	default:
		return nil, fmt.Errorf("unsupported expression node %T", expr)
	}
}

func jsonLogical(n *api.LogicalExpression) (any, error) {
	if n.Op != api.LogicalAnd && n.Op != api.LogicalOr && n.Op != api.LogicalNot {
		return nil, fmt.Errorf("unsupported logical operator %q", n.Op)
	}
	args := make([]any, len(n.Args))
	for i, arg := range n.Args {
		value, err := jsonExpression(arg)
		if err != nil {
			return nil, err
		}
		args[i] = value
	}
	return map[string]any{"op": string(n.Op), "args": args}, nil
}

func jsonOp(op string, args ...api.Node) (any, error) {
	values := make([]any, len(args))
	for i, arg := range args {
		value, err := jsonNode(arg)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	return map[string]any{"op": op, "args": values}, nil
}

func jsonNode(node api.Node) (any, error) {
	switch n := node.(type) {
	case api.ScalarExpression:
		return jsonScalar(n)
	case api.Expression:
		return jsonExpression(n)
	case *api.TemporalUnbounded:
		return "..", nil
	case *api.TemporalInterval:
		return jsonTemporalInterval(n)
	case *api.GeometryLiteral:
		return jsonGeometry(n)
	default:
		return nil, fmt.Errorf("unsupported node %T", node)
	}
}

func jsonScalar(scalar api.ScalarExpression) (any, error) {
	switch n := scalar.(type) {
	case *api.PropertyRef:
		return map[string]any{"property": n.Name}, nil
	case *api.Literal:
		return jsonLiteral(n)
	case *api.FunctionCall:
		return jsonFunction(n)
	case *api.ArithmeticExpression:
		return jsonOp(string(n.Op), n.Left, n.Right)
	case *api.TemporalInstant:
		return jsonTemporalInstant(n)
	case *api.ArrayLiteral:
		values := make([]any, len(n.Values))
		for i, item := range n.Values {
			value, err := jsonNode(item)
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported scalar node %T", scalar)
	}
}

func jsonLiteral(n *api.Literal) (any, error) {
	switch n.Kind {
	case api.LiteralString:
		value, ok := n.Value.(string)
		if !ok {
			return nil, fmt.Errorf("string literal value has type %T", n.Value)
		}
		return value, nil
	case api.LiteralNumber:
		raw, err := numberRaw(n.Value)
		if err != nil {
			return nil, err
		}
		return raw, nil
	case api.LiteralBool:
		value, ok := n.Value.(bool)
		if !ok {
			return nil, fmt.Errorf("bool literal value has type %T", n.Value)
		}
		return value, nil
	case api.LiteralNull:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported literal kind %q", n.Kind)
	}
}

func jsonFunction(n *api.FunctionCall) (any, error) {
	args := make([]any, len(n.Args))
	for i, arg := range n.Args {
		value, err := jsonNode(arg)
		if err != nil {
			return nil, err
		}
		args[i] = value
	}
	return map[string]any{"op": n.Name, "args": args}, nil
}

func jsonTemporalInstant(n *api.TemporalInstant) (any, error) {
	switch n.Kind {
	case api.TemporalInstantDate:
		return map[string]any{"date": n.Value}, nil
	case api.TemporalInstantTimestamp:
		return map[string]any{"timestamp": n.Value}, nil
	default:
		return nil, fmt.Errorf("unsupported temporal instant kind %q", n.Kind)
	}
}

func jsonTemporalInterval(n *api.TemporalInterval) (any, error) {
	start, err := jsonTemporalIntervalEndpoint(n.Start)
	if err != nil {
		return nil, err
	}
	end, err := jsonTemporalIntervalEndpoint(n.End)
	if err != nil {
		return nil, err
	}
	return map[string]any{"interval": []any{start, end}}, nil
}

func jsonTemporalIntervalEndpoint(node api.Node) (any, error) {
	switch n := node.(type) {
	case *api.TemporalUnbounded:
		return "..", nil
	case *api.TemporalInstant:
		return n.Value, nil
	case api.ScalarExpression:
		return jsonScalar(n)
	default:
		return nil, fmt.Errorf("unsupported interval endpoint %T", node)
	}
}

func jsonGeometry(n *api.GeometryLiteral) (any, error) {
	switch n.Type {
	case api.GeometryTypeBBox:
		return map[string]any{"bbox": n.BBox}, nil
	case api.GeometryTypePoint:
		coord, ok := n.Coordinates.(api.Coordinate)
		if !ok {
			return nil, fmt.Errorf("point coordinates have type %T", n.Coordinates)
		}
		return jsonGeometryObject(n, jsonCoordinate(coord)), nil
	case api.GeometryTypeMultiPoint, api.GeometryTypeLineString:
		coords, ok := n.Coordinates.([]api.Coordinate)
		if !ok {
			return nil, fmt.Errorf("%s coordinates have type %T", n.Type, n.Coordinates)
		}
		return jsonGeometryObject(n, jsonCoordinateList(coords)), nil
	case api.GeometryTypePolygon, api.GeometryTypeMultiLineString:
		coords, ok := n.Coordinates.([][]api.Coordinate)
		if !ok {
			return nil, fmt.Errorf("%s coordinates have type %T", n.Type, n.Coordinates)
		}
		return jsonGeometryObject(n, jsonCoordinateMatrix(coords)), nil
	case api.GeometryTypeMultiPolygon:
		coords, ok := n.Coordinates.([][][]api.Coordinate)
		if !ok {
			return nil, fmt.Errorf("multipolygon coordinates have type %T", n.Coordinates)
		}
		return jsonGeometryObject(n, jsonCoordinateCube(coords)), nil
	case api.GeometryTypeGeometryCollection:
		geoms := make([]any, len(n.Geometries))
		for i, geom := range n.Geometries {
			value, err := jsonGeometry(geom)
			if err != nil {
				return nil, err
			}
			geoms[i] = value
		}
		obj := map[string]any{"type": string(n.Type), "geometries": geoms}
		addJSONBBox(obj, n.BBox)
		return obj, nil
	default:
		return nil, fmt.Errorf("unsupported geometry type %q", n.Type)
	}
}

func jsonGeometryObject(geom *api.GeometryLiteral, coordinates any) map[string]any {
	obj := map[string]any{"type": string(geom.Type), "coordinates": coordinates}
	addJSONBBox(obj, geom.BBox)
	return obj
}

func addJSONBBox(obj map[string]any, bbox []float64) {
	if len(bbox) > 0 {
		obj["bbox"] = bbox
	}
}

func jsonCoordinate(coord api.Coordinate) []float64 {
	values := []float64{coord.X, coord.Y}
	if coord.HasZ {
		values = append(values, coord.Z)
	}
	return values
}

func jsonCoordinateList(coords []api.Coordinate) [][]float64 {
	values := make([][]float64, len(coords))
	for i, coord := range coords {
		values[i] = jsonCoordinate(coord)
	}
	return values
}

func jsonCoordinateMatrix(coords [][]api.Coordinate) [][][]float64 {
	values := make([][][]float64, len(coords))
	for i, list := range coords {
		values[i] = jsonCoordinateList(list)
	}
	return values
}

func jsonCoordinateCube(coords [][][]api.Coordinate) [][][][]float64 {
	values := make([][][][]float64, len(coords))
	for i, matrix := range coords {
		values[i] = jsonCoordinateMatrix(matrix)
	}
	return values
}

func quoteString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func textFunctionName(name string) (string, error) {
	if name == api.FunctionNameCaseI || name == api.FunctionNameAccenti {
		return strings.ToUpper(name), nil
	}
	text, err := quoteIdentifier(name)
	if err != nil {
		return "", fmt.Errorf("function name %q cannot be serialized as CQL2 Text: %w", name, err)
	}
	return text, nil
}

func quoteIdentifier(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("identifier must not be empty")
	}
	if !validIdentifier(name) {
		return "", fmt.Errorf("identifier contains characters that are not valid in CQL2 Text")
	}
	if _, reserved := reservedTextKeywords[strings.ToUpper(name)]; reserved {
		return `"` + name + `"`, nil
	}
	return name, nil
}

func validIdentifier(name string) bool {
	for i, r := range name {
		if r == utf8.RuneError {
			return false
		}
		if i == 0 {
			if !isIdentifierStart(r) {
				return false
			}
		} else if !isIdentifierPart(r) {
			return false
		}
	}
	return true
}

func isIdentifierStart(r rune) bool {
	switch {
	case r == ':' || r == '_':
		return true
	case 'A' <= r && r <= 'Z':
		return true
	case 'a' <= r && r <= 'z':
		return true
	case '\u00c0' <= r && r <= '\u00d6':
		return true
	case '\u00d8' <= r && r <= '\u00f6':
		return true
	case '\u00f8' <= r && r <= '\u02ff':
		return true
	case '\u0370' <= r && r <= '\u037d':
		return true
	case '\u037f' <= r && r <= '\u1ffe':
		return true
	case '\u200c' <= r && r <= '\u200d':
		return true
	case '\u2070' <= r && r <= '\u218f':
		return true
	case '\u2c00' <= r && r <= '\u2fef':
		return true
	case '\u3001' <= r && r <= '\ud7ff':
		return true
	case '\uf900' <= r && r <= '\ufdcf':
		return true
	case '\ufdf0' <= r && r <= '\ufffd':
		return true
	case 0x10000 <= r && r <= 0xeffff:
		return true
	default:
		return false
	}
}

func isIdentifierPart(r rune) bool {
	return isIdentifierStart(r) || '0' <= r && r <= '9' || r == '.' || '\u0300' <= r && r <= '\u036f' || '\u203f' <= r && r <= '\u2040'
}

var reservedTextKeywords = map[string]struct{}{
	"AND": {}, "OR": {}, "NOT": {}, "LIKE": {}, "BETWEEN": {}, "IN": {}, "IS": {}, "NULL": {}, "TRUE": {}, "FALSE": {},
	"CASEI": {}, "ACCENTI": {}, "DIV": {}, "DATE": {}, "TIMESTAMP": {}, "INTERVAL": {},
	"POINT": {}, "LINESTRING": {}, "POLYGON": {}, "MULTIPOINT": {}, "MULTILINESTRING": {}, "MULTIPOLYGON": {}, "GEOMETRYCOLLECTION": {}, "BBOX": {},
	"S_INTERSECTS": {}, "S_EQUALS": {}, "S_DISJOINT": {}, "S_TOUCHES": {}, "S_WITHIN": {}, "S_OVERLAPS": {}, "S_CROSSES": {}, "S_CONTAINS": {},
	"T_AFTER": {}, "T_BEFORE": {}, "T_CONTAINS": {}, "T_DISJOINT": {}, "T_DURING": {}, "T_EQUALS": {}, "T_FINISHEDBY": {}, "T_FINISHES": {}, "T_INTERSECTS": {}, "T_MEETS": {}, "T_METBY": {}, "T_OVERLAPPEDBY": {}, "T_OVERLAPS": {}, "T_STARTEDBY": {}, "T_STARTS": {},
	"A_EQUALS": {}, "A_CONTAINS": {}, "A_CONTAINEDBY": {}, "A_OVERLAPS": {},
}

func numberString(value any) (string, error) {
	switch v := value.(type) {
	case string:
		if err := validateJSONNumber(v); err != nil {
			return "", err
		}
		return v, nil
	case json.Number:
		if err := validateJSONNumber(v.String()); err != nil {
			return "", err
		}
		return v.String(), nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		text := formatFloat(v)
		if err := validateJSONNumber(text); err != nil {
			return "", err
		}
		return text, nil
	default:
		return "", fmt.Errorf("number literal value has type %T", value)
	}
}

func numberRaw(value any) (json.RawMessage, error) {
	text, err := numberString(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(text), nil
}

func validateJSONNumber(text string) error {
	if text == "" || !json.Valid([]byte(text)) {
		return fmt.Errorf("invalid numeric literal %q", text)
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(text)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("invalid numeric literal %q", text)
	}
	if _, ok := value.(json.Number); !ok {
		return fmt.Errorf("invalid numeric literal %q", text)
	}
	if err := decoder.Decode(&struct{}{}); err == nil {
		return fmt.Errorf("invalid numeric literal %q", text)
	}
	return nil
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func joinFloat64(values []float64, sep string) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = formatFloat(value)
	}
	return strings.Join(parts, sep)
}

func functionReturnsBoolean(call *api.FunctionCall) bool {
	for _, typ := range call.ReturnTypes {
		if typ == api.FunctionTypeBoolean || typ == api.FunctionTypeAny {
			return true
		}
	}
	return len(call.ReturnTypes) == 0
}

func jsonTemporalOp(op api.TemporalPredicateOp) string {
	switch op {
	case api.TemporalOpFinishedBy:
		return "t_finishedBy"
	case api.TemporalOpMetBy:
		return "t_metBy"
	case api.TemporalOpOverlappedBy:
		return "t_overlappedBy"
	case api.TemporalOpStartedBy:
		return "t_startedBy"
	default:
		return string(op)
	}
}

func jsonArrayOp(op api.ArrayPredicateOp) string {
	if op == api.ArrayOpContainedBy {
		return "a_containedBy"
	}
	return string(op)
}
