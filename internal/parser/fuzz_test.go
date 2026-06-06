package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/cwygoda/gocql2/api"
)

func FuzzParseTextDoesNotPanic(f *testing.F) {
	for _, seed := range []string{
		`name = 'alice'`,
		`name = 'O\'Brien'`,
		`height BETWEEN 1 AND 2`,
		`status IN ('new','done')`,
		`NOT (a = 1 OR b <> 2)`,
		`CASEI(name) = casei('alice')`,
		`S_INTERSECTS(geom,POINT(7.02 49.92))`,
		`S_WITHIN(geom,BBOX(-180,-90,180,90))`,
		`A_CONTAINS(tags, ('foo', 'bar'))`,
		`A_OVERLAPS(tags, (1, TRUE, status = 'new'))`,
		`T_AFTER(event_time,TIMESTAMP('2022-04-24T07:59:57Z'))`,
		`T_DURING(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`,
		`"AND" = 1`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if _, err := NewParser().WithMaxDepth(32).ParseText(input); err != nil {
			return
		}
	})
}

func FuzzParseTextWithFunctionRegistryDoesNotPanic(f *testing.F) {
	for _, seed := range []string{
		`has_text(name, 'alice')`,
		`has_text(name)`,
		`has_text(name, 1)`,
		`unknown(name)`,
	} {
		f.Add(seed)
	}
	parser := NewParser().WithAllowedFunctions(api.FunctionDefinition{
		Name: "has_text",
		Args: []api.FunctionArgument{
			{Name: "value", Types: []api.FunctionType{api.FunctionTypeString}},
			{Name: "needle", Types: []api.FunctionType{api.FunctionTypeString}, Variadic: true},
		},
		Returns: []api.FunctionType{api.FunctionTypeBoolean},
	})
	f.Fuzz(func(t *testing.T, input string) {
		if _, err := parser.ParseText(input); err != nil {
			return
		}
	})
}

func FuzzStandardTextFunctionsDoNotPanic(f *testing.F) {
	for _, seed := range []struct {
		fn    string
		value string
	}{
		{api.FunctionNameCaseI, "Alice%"},
		{api.FunctionNameAccenti, "Äé%"},
		{api.FunctionNameCaseI, "O'Brien"},
		{api.FunctionNameAccenti, `slash\\percent%_`},
	} {
		f.Add(seed.fn, seed.value)
	}
	f.Fuzz(func(t *testing.T, fn, value string) {
		fn = strings.ToLower(fn)
		if fn != api.FunctionNameCaseI && fn != api.FunctionNameAccenti {
			fn = api.FunctionNameCaseI
		}

		textLiteral := cqlTextString(value)
		textInputs := []string{
			fmt.Sprintf("%s(name) = %s(%s)", strings.ToUpper(fn), fn, textLiteral),
			fmt.Sprintf("name LIKE %s(%s)", fn, textLiteral),
			fmt.Sprintf("ACCENTI(CASEI(name)) LIKE accenti(casei(%s))", textLiteral),
		}
		for _, input := range textInputs {
			if _, err := NewParser().WithMaxDepth(32).WithConformance(api.ConformanceAdvancedComparisonOperators).WithAllowedFunctions(api.StandardTextFunctions()...).ParseText(input); err != nil {
				t.Fatalf("ParseText(%q): %v", input, err)
			}
		}

		jsonLiteral, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("json.Marshal(%q): %v", value, err)
		}
		jsonInputs := []string{
			fmt.Sprintf(`{"op":"=","args":[{"op":%q,"args":[{"property":"name"}]},{"op":%q,"args":[%s]}]}`, fn, fn, jsonLiteral),
			fmt.Sprintf(`{"op":"like","args":[{"property":"name"},{"op":%q,"args":[%s]}]}`, fn, jsonLiteral),
			fmt.Sprintf(`{"op":"like","args":[{"op":"accenti","args":[{"op":"casei","args":[{"property":"name"}]}]},{"op":"accenti","args":[{"op":"casei","args":[%s]}]}]}`, jsonLiteral),
		}
		for _, input := range jsonInputs {
			if _, err := NewParser().WithMaxDepth(32).WithConformance(api.ConformanceAdvancedComparisonOperators).WithAllowedFunctions(api.StandardTextFunctions()...).ParseJSON([]byte(input)); err != nil {
				t.Fatalf("ParseJSON(%s): %v", input, err)
			}
		}
	})
}

func cqlTextString(value string) string {
	return "'" + strings.NewReplacer(`\\`, `\\\\`, `'`, `\'`).Replace(value) + "'"
}

func FuzzSpatialPredicatesDoNotPanic(f *testing.F) {
	for _, seed := range []struct {
		op string
		x  float64
		y  float64
	}{
		{"s_intersects", 7.02, 49.92},
		{"s_disjoint", -180, -90},
		{"s_equals", 180, 90},
		{"s_within", 10, 51},
	} {
		f.Add(seed.op, seed.x, seed.y)
	}
	f.Fuzz(func(t *testing.T, op string, x, y float64) {
		op = strings.ToLower(op)
		if _, ok := spatialPredicateOps[op]; !ok {
			op = string(api.SpatialOpIntersects)
		}
		if x < -180 || x > 180 || y < -90 || y > 90 {
			return
		}

		text := fmt.Sprintf("%s(geom,POINT(%g %g))", strings.ToUpper(op), x, y)
		if _, err := NewParser().WithMaxDepth(32).WithConformance(api.ConformanceSpatialFunctions).ParseText(text); err != nil {
			t.Fatalf("ParseText(%q): %v", text, err)
		}

		input := fmt.Sprintf(`{"op":%q,"args":[{"property":"geom"},{"type":"Point","coordinates":[%g,%g]}]}`, op, x, y)
		if _, err := NewParser().WithMaxDepth(32).WithConformance(api.ConformanceSpatialFunctions).ParseJSON([]byte(input)); err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
	})
}

func FuzzArrayPredicatesDoNotPanic(f *testing.F) {
	for _, seed := range []struct {
		op     string
		first  string
		second string
	}{
		{"a_contains", "foo", "bar"},
		{"a_containedby", "", "quoted'value"},
		{"a_equals", "é", "slash\\value"},
		{"a_overlaps", "nested", "percent%_"},
	} {
		f.Add(seed.op, seed.first, seed.second)
	}
	f.Fuzz(func(t *testing.T, op, first, second string) {
		op = strings.ToLower(op)
		if _, ok := arrayPredicateOps[op]; !ok {
			op = string(api.ArrayOpContains)
		}

		text := fmt.Sprintf("%s(tags, (%s, %s))", strings.ToUpper(op), cqlTextString(first), cqlTextString(second))
		if _, err := NewParser().WithMaxDepth(32).WithConformance(api.ConformanceArrayFunctions).ParseText(text); err != nil {
			t.Fatalf("ParseText(%q): %v", text, err)
		}

		jsonFirst, err := json.Marshal(first)
		if err != nil {
			t.Fatalf("json.Marshal(%q): %v", first, err)
		}
		jsonSecond, err := json.Marshal(second)
		if err != nil {
			t.Fatalf("json.Marshal(%q): %v", second, err)
		}
		jsonOp := op
		if op == "a_containedby" {
			jsonOp = "a_containedBy"
		}
		input := fmt.Sprintf(`{"op":%q,"args":[{"property":"tags"},[%s,%s]]}`, jsonOp, jsonFirst, jsonSecond)
		if _, err := NewParser().WithMaxDepth(32).WithConformance(api.ConformanceArrayFunctions).ParseJSON([]byte(input)); err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
	})
}

func FuzzParseJSONDoesNotPanic(f *testing.F) {
	for _, seed := range []string{
		`{"op":"=","args":[{"property":"name"},"alice"]}`,
		`{"op":"and","args":[true,{"op":"not","args":[false]}]}`,
		`{"op":"in","args":[{"property":"status"},["new","done"]]}`,
		`{"op":"=","args":[{"op":"casei","args":[{"property":"name"}]},{"op":"casei","args":["alice"]}]}`,
		`{"op":"s_intersects","args":[{"property":"geom"},{"type":"Point","coordinates":[7.02,49.92]}]}`,
		`{"op":"s_within","args":[{"property":"geom"},{"bbox":[-180,-90,180,90]}]}`,
		`{"op":"a_contains","args":[{"property":"tags"},["foo","bar"]]}`,
		`{"op":"a_overlaps","args":[{"property":"tags"},[1,true,{"op":"=","args":[{"property":"status"},"new"]}]]}`,
		`{"op":"t_after","args":[{"property":"event_time"},{"timestamp":"2022-04-24T07:59:57Z"}]}`,
		`{"op":"t_during","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`,
		`null`,
		`{`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if _, err := NewParser().WithMaxDepth(32).ParseJSON([]byte(input)); err != nil {
			return
		}
	})
}

func FuzzTemporalPredicatesDoNotPanic(f *testing.F) {
	for _, seed := range []struct {
		op    string
		start string
		end   string
	}{
		{"t_after", "2021-01-01", "2021-12-31"},
		{"t_before", "2021-01-01T00:00:00Z", "2021-12-31T23:59:59Z"},
		{"t_intersects", "2021-01-01T00:00:00Z", "2021-12-31T23:59:59Z"},
		{"t_during", "2021-01-01", ".."},
		{"t_overlaps", "..", "2021-12-31"},
	} {
		f.Add(seed.op, seed.start, seed.end)
	}
	f.Fuzz(func(t *testing.T, op, start, end string) {
		op = strings.ToLower(op)
		temporalOp, ok := temporalPredicateOps[op]
		if !ok {
			temporalOp = api.TemporalOpIntersects
			op = string(temporalOp)
		}

		left := "event_time"
		if isIntervalOnlyTemporalPredicate(temporalOp) {
			left = "INTERVAL(start_time,end_time)"
		}
		text := fmt.Sprintf("%s(%s,INTERVAL(%s,%s))", strings.ToUpper(op), left, cqlTextString(start), cqlTextString(end))
		if _, err := NewParser().WithMaxDepth(32).ParseText(text); err != nil {
			return
		}

		jsonStart, err := json.Marshal(start)
		if err != nil {
			t.Fatalf("json.Marshal(%q): %v", start, err)
		}
		jsonEnd, err := json.Marshal(end)
		if err != nil {
			t.Fatalf("json.Marshal(%q): %v", end, err)
		}
		jsonLeft := `{"property":"event_time"}`
		if isIntervalOnlyTemporalPredicate(temporalOp) {
			jsonLeft = `{"interval":[{"property":"start_time"},{"property":"end_time"}]}`
		}
		input := fmt.Sprintf(`{"op":%q,"args":[%s,{"interval":[%s,%s]}]}`, op, jsonLeft, jsonStart, jsonEnd)
		if _, err := NewParser().WithMaxDepth(32).ParseJSON([]byte(input)); err != nil {
			return
		}
	})
}
