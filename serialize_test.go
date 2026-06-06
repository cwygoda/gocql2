package gocql2

import (
	"reflect"
	"testing"

	"github.com/cwygoda/gocql2/api"
)

func TestSerializeRootFunctions(t *testing.T) {
	parser := serializerTestParser()
	expr, err := parser.ParseText("name = 'x'")
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if got, err := SerializeText(expr); err != nil || got == "" {
		t.Fatalf("SerializeText = %q, %v", got, err)
	}
	if got, err := SerializeJSON(expr); err != nil || len(got) == 0 {
		t.Fatalf("SerializeJSON = %q, %v", got, err)
	}
}

func TestSerializeTextRoundTrip(t *testing.T) {
	cases := []string{
		`a = 1 OR b = 2 AND NOT c = 3`,
		`name NOT LIKE CASEI('A%')`,
		`height NOT BETWEEN 1 AND 2`,
		`status NOT IN ('old')`,
		`deleted_at IS NOT NULL`,
		`(height + 1) * 2 >= other DIV -3`,
		`A_CONTAINS(tags, ('foo'))`,
		`S_INTERSECTS(geom, POINT(1 2))`,
		`T_EQUALS(INTERVAL('2021-01-01', '..'), INTERVAL(start_time, end_time))`,
		`"AND" = 'reserved property'`,
	}
	parser := serializerTestParser()
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			expr, err := parser.ParseText(input)
			if err != nil {
				t.Fatalf("ParseText: %v", err)
			}
			text, err := SerializeText(expr)
			if err != nil {
				t.Fatalf("SerializeText: %v", err)
			}
			roundTrip, err := parser.ParseText(text)
			if err != nil {
				t.Fatalf("ParseText(%q): %v", text, err)
			}
			assertSameStructure(t, expr, roundTrip)
		})
	}
}

func TestSerializeRegisteredFunctionsRoundTrip(t *testing.T) {
	textParser := serializerTestParser().WithAllowedFunctions(
		api.FunctionDefinition{
			Name: "Normalize",
			Args: []api.FunctionArgument{{
				Name:  "value",
				Types: []api.FunctionType{api.FunctionTypeString},
			}},
			Returns: []api.FunctionType{api.FunctionTypeString},
		},
		api.FunctionDefinition{
			Name: "and",
			Args: []api.FunctionArgument{{
				Name:  "value",
				Types: []api.FunctionType{api.FunctionTypeString},
			}},
			Returns: []api.FunctionType{api.FunctionTypeString},
		},
	)
	textExpr, err := textParser.ParseText(`Normalize(name) = "and"('Oak')`)
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	text, err := SerializeText(textExpr)
	if err != nil {
		t.Fatalf("SerializeText: %v", err)
	}
	textRoundTrip, err := textParser.ParseText(text)
	if err != nil {
		t.Fatalf("ParseText(%q): %v", text, err)
	}
	assertSameStructure(t, textExpr, textRoundTrip)

	jsonParser := serializerTestParser().WithAllowedFunctions(api.FunctionDefinition{
		Name: "vendor-func",
		Args: []api.FunctionArgument{{
			Name:  "value",
			Types: []api.FunctionType{api.FunctionTypeNumber},
		}},
		Returns: []api.FunctionType{api.FunctionTypeNumber},
	})
	jsonExpr, err := jsonParser.ParseJSON([]byte(`{"op":"=","args":[{"op":"vendor-func","args":[1]},2]}`))
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	jsonBytes, err := SerializeJSON(jsonExpr)
	if err != nil {
		t.Fatalf("SerializeJSON: %v", err)
	}
	jsonRoundTrip, err := jsonParser.ParseJSON(jsonBytes)
	if err != nil {
		t.Fatalf("ParseJSON(%s): %v", jsonBytes, err)
	}
	assertSameStructure(t, jsonExpr, jsonRoundTrip)
}

func TestSerializeTextToJSONRoundTrip(t *testing.T) {
	parser := serializerTestParser()
	expr, err := parser.ParseText(`name NOT LIKE CASEI('A%') AND deleted_at IS NOT NULL`)
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	jsonBytes, err := SerializeJSON(expr)
	if err != nil {
		t.Fatalf("SerializeJSON: %v", err)
	}
	roundTrip, err := parser.ParseJSON(jsonBytes)
	if err != nil {
		t.Fatalf("ParseJSON(%s): %v", jsonBytes, err)
	}
	assertSameStructure(t, expr, roundTrip)
}

func TestSerializeJSONRoundTrip(t *testing.T) {
	cases := []string{
		`{"op":"and","args":[{"op":"=","args":[{"property":"name"},"Oak"]},{"op":">=","args":[{"property":"height"},10]}]}`,
		`{"op":"not","args":[{"op":"like","args":[{"op":"casei","args":[{"property":"name"}]},{"op":"casei","args":["A%"]}]}]}`,
		`{"op":">=","args":[{"op":"*","args":[{"op":"+","args":[{"property":"height"},1]},2]},{"op":"div","args":[{"property":"other"},3]}]}`,
		`{"op":"a_containedBy","args":[["foo"],{"property":"tags"}]}`,
		`{"op":"s_equals","args":[{"property":"geom"},{"type":"LineString","bbox":[7,50,10,51],"coordinates":[[7,50,1],[10,51,2]]}]}`,
		`{"op":"t_finishedBy","args":[{"interval":["2021-01-01","2021-12-31"]},{"interval":[{"property":"start_time"},{"property":"end_time"}]}]}`,
		`{"op":"isNull","args":[{"bbox":[0,0,1,1]}]}`,
	}
	parser := serializerTestParser()
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			expr, err := parser.ParseJSON([]byte(input))
			if err != nil {
				t.Fatalf("ParseJSON: %v", err)
			}
			jsonBytes, err := SerializeJSON(expr)
			if err != nil {
				t.Fatalf("SerializeJSON: %v", err)
			}
			roundTrip, err := parser.ParseJSON(jsonBytes)
			if err != nil {
				t.Fatalf("ParseJSON(%s): %v", jsonBytes, err)
			}
			assertSameStructure(t, expr, roundTrip)
		})
	}
}

func serializerTestParser() *Parser {
	return NewParser().WithConformance(
		api.ConformanceAdvancedComparisonOperators,
		api.ConformanceCaseInsensitiveComparison,
		api.ConformanceAccentInsensitiveComparison,
		api.ConformancePropertyProperty,
		api.ConformanceArithmetic,
		api.ConformanceArrayFunctions,
		api.ConformanceSpatialFunctions,
		api.ConformanceTemporalFunctions,
	)
}

func assertSameStructure(t *testing.T, left, right api.Node) {
	t.Helper()
	leftSem := structure(left)
	rightSem := structure(right)
	if !reflect.DeepEqual(leftSem, rightSem) {
		t.Fatalf("structure mismatch\nleft:  %#v\nright: %#v", leftSem, rightSem)
	}
}

func structure(node api.Node) any {
	switch n := node.(type) {
	case *api.LogicalExpression:
		args := make([]any, len(n.Args))
		for i, arg := range n.Args {
			args[i] = structure(arg)
		}
		return map[string]any{"type": "logical", "op": string(n.Op), "args": args}
	case *api.ComparisonExpression:
		return map[string]any{"type": "comparison", "op": string(n.Op), "left": structure(n.Left), "right": structure(n.Right)}
	case *api.ArithmeticExpression:
		return map[string]any{"type": "arithmetic", "op": string(n.Op), "left": structure(n.Left), "right": structure(n.Right)}
	case *api.LikeExpression:
		return map[string]any{"type": "like", "expr": structure(n.Expr), "pattern": structure(n.Pattern), "not": n.Not}
	case *api.BetweenExpression:
		return map[string]any{"type": "between", "expr": structure(n.Expr), "lower": structure(n.Lower), "upper": structure(n.Upper), "not": n.Not}
	case *api.InExpression:
		values := make([]any, len(n.Values))
		for i, value := range n.Values {
			values[i] = structure(value)
		}
		return map[string]any{"type": "in", "expr": structure(n.Expr), "values": values, "not": n.Not}
	case *api.IsNullExpression:
		return map[string]any{"type": "isNull", "expr": structure(n.Expr), "not": n.Not}
	case *api.SpatialPredicateExpression:
		return map[string]any{"type": "spatial", "op": string(n.Op), "left": structure(n.Left), "right": structure(n.Right)}
	case *api.TemporalPredicateExpression:
		return map[string]any{"type": "temporal", "op": string(n.Op), "left": structure(n.Left), "right": structure(n.Right)}
	case *api.ArrayPredicateExpression:
		return map[string]any{"type": "arrayPredicate", "op": string(n.Op), "left": structure(n.Left), "right": structure(n.Right)}
	case *api.TemporalInstant:
		return map[string]any{"type": "temporalInstant", "kind": string(n.Kind), "value": n.Value}
	case *api.TemporalUnbounded:
		return map[string]any{"type": "temporalUnbounded"}
	case *api.TemporalInterval:
		return map[string]any{"type": "temporalInterval", "start": structure(n.Start), "end": structure(n.End)}
	case *api.GeometryLiteral:
		geoms := make([]any, len(n.Geometries))
		for i, geom := range n.Geometries {
			geoms[i] = structure(geom)
		}
		return map[string]any{"type": "geometry", "geometryType": string(n.Type), "coordinates": n.Coordinates, "bbox": n.BBox, "geometries": geoms}
	case *api.PropertyRef:
		return map[string]any{"type": "property", "name": n.Name}
	case *api.Literal:
		return map[string]any{"type": "literal", "kind": string(n.Kind), "value": n.Value}
	case *api.FunctionCall:
		args := make([]any, len(n.Args))
		for i, arg := range n.Args {
			args[i] = structure(arg)
		}
		return map[string]any{"type": "function", "name": n.Name, "args": args, "returns": n.ReturnTypes}
	case *api.ArrayLiteral:
		values := make([]any, len(n.Values))
		for i, value := range n.Values {
			values[i] = structure(value)
		}
		return map[string]any{"type": "array", "values": values}
	default:
		return map[string]any{"type": "unknown"}
	}
}
