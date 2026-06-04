package parser

import (
	"reflect"
	"testing"

	"github.com/cwygoda/cql2/api"
)

func TestWithConformanceCanonicalizesAndEnablesStandardFunctions(t *testing.T) {
	parser := NewParser().WithConformance("case-insensitive-comparison",
		"/conf/accent-insensitive-comparison/accenti",
		api.ConformanceCaseInsensitiveComparison)

	wantClasses := []string{api.ConformanceCaseInsensitiveComparison, api.ConformanceAccentInsensitiveComparison}
	if got := parser.ConformanceClasses(); !reflect.DeepEqual(got, wantClasses) {
		t.Fatalf("ConformanceClasses() = %#v, want %#v", got, wantClasses)
	}

	wantFunctions := []string{"accenti", "casei"}
	if got := parser.SupportedFunctions(); !reflect.DeepEqual(got, wantFunctions) {
		t.Fatalf("SupportedFunctions() = %#v, want %#v", got, wantFunctions)
	}

	if _, err := parser.ParseText(`casei(name) = casei('ALICE') AND accenti(city) = 'Lodz'`); err != nil {
		t.Fatalf("ParseText with conformance functions: %v", err)
	}
	_, err := parser.ParseText(`tolower(name) = 'alice'`)
	assertParseErrorContains(t, err, `function "tolower" is not allowed`)
}

func TestDefaultParserRejectsOptionalConformanceSyntax(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: `name LIKE 'A%'`, want: `advanced-comparison-operators`},
		{input: `height BETWEEN 1 AND 2`, want: `advanced-comparison-operators`},
		{input: `status IN ('new')`, want: `advanced-comparison-operators`},
		{input: `height + 1 > 2`, want: `arithmetic conformance`},
		{input: `FALSE <> archived`, want: `property-property`},
		{input: `S_INTERSECTS(geom,POINT(1 2))`, want: `spatial conformance`},
		{input: `T_AFTER(event_time,TIMESTAMP('2022-01-01T00:00:00Z'))`, want: `temporal-functions`},
		{input: `A_CONTAINS(tags, ('foo'))`, want: `array-functions`},
	}
	for _, tc := range cases {
		_, err := ParseText(tc.input)
		assertParseErrorContains(t, err, tc.want)
	}

	_, err := ParseJSON([]byte(`{"op":"like","args":[{"property":"name"},"A%"]}`))
	assertParseErrorContains(t, err, `advanced-comparison-operators`)
	_, err = ParseJSON([]byte(`{"op":"s_intersects","args":[{"property":"geom"},{"type":"Point","coordinates":[1,2]}]}`))
	assertParseErrorContains(t, err, `spatial conformance`)
}

func TestWithConformanceUsesOnlyClassFunctionsWhenNoExplicitFunctions(t *testing.T) {
	parser := NewParser().WithConformance(api.ConformanceBasicCQL2)
	if got := parser.SupportedFunctions(); len(got) != 0 {
		t.Fatalf("SupportedFunctions() = %#v, want no functions", got)
	}

	_, err := parser.ParseText(`casei(name) = 'alice'`)
	assertParseErrorContains(t, err, `function "casei" is not allowed`)
}

func TestWithConformanceMergesExplicitFunctions(t *testing.T) {
	parser := NewParser().WithAllowedFunctions(api.FunctionDefinition{Name: "tolower", Args: []api.FunctionArgument{{Types: []api.FunctionType{api.FunctionTypeString}}}, Returns: []api.FunctionType{api.FunctionTypeString}}).WithConformance(api.ConformanceCaseInsensitiveComparison)

	wantFunctions := []string{"casei", "tolower"}
	if got := parser.SupportedFunctions(); !reflect.DeepEqual(got, wantFunctions) {
		t.Fatalf("SupportedFunctions() = %#v, want %#v", got, wantFunctions)
	}
	if _, err := parser.ParseText(`casei(tolower(name)) = 'alice'`); err != nil {
		t.Fatalf("ParseText with merged functions: %v", err)
	}
}

func TestStandardFunctionsForConformance(t *testing.T) {
	spatial := api.StandardFunctionsForConformance(api.ConformanceSpatialFunctions)
	if got, want := functionNames(spatial), []string{"s_contains", "s_crosses", "s_disjoint", "s_equals", "s_intersects", "s_overlaps", "s_touches", "s_within"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("spatial functions = %#v, want %#v", got, want)
	}

	temporal := api.StandardFunctionsForConformance(api.ConformanceTemporalFunctions)
	if got, want := len(temporal), 15; got != want {
		t.Fatalf("temporal functions len = %d, want %d", got, want)
	}

	array := api.StandardFunctionsForConformance(api.ConformanceArrayFunctions)
	if got, want := functionNames(array), []string{"a_containedby", "a_contains", "a_equals", "a_overlaps"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("array functions = %#v, want %#v", got, want)
	}

	parser := NewParser().WithConformance(api.ConformanceBasicSpatialFunctions, api.ConformancePropertyProperty)
	if _, err := parser.ParseText(`TRUE = s_intersects(geom, BBOX(0, 0, 1, 1))`); err != nil {
		t.Fatalf("spatial function in scalar context: %v", err)
	}
}
