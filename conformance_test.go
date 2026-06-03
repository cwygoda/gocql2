package gocql2

import (
	"reflect"
	"testing"
)

func TestWithConformanceCanonicalizesAndEnablesStandardFunctions(t *testing.T) {
	parser := NewParser(WithConformance(
		"case-insensitive-comparison",
		"/conf/accent-insensitive-comparison/accenti",
		ConformanceCaseInsensitiveComparison,
	))

	wantClasses := []string{ConformanceCaseInsensitiveComparison, ConformanceAccentInsensitiveComparison}
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

func TestWithConformanceUsesOnlyClassFunctionsWhenNoExplicitFunctions(t *testing.T) {
	parser := NewParser(WithConformance(ConformanceBasicCQL2))
	if got := parser.SupportedFunctions(); len(got) != 0 {
		t.Fatalf("SupportedFunctions() = %#v, want no functions", got)
	}

	_, err := parser.ParseText(`casei(name) = 'alice'`)
	assertParseErrorContains(t, err, `function "casei" is not allowed`)
}

func TestWithConformanceMergesExplicitFunctions(t *testing.T) {
	parser := NewParser(
		WithAllowedFunctions(FunctionDefinition{Name: "tolower", Args: []FunctionArgument{{Types: []FunctionType{FunctionTypeString}}}, Returns: []FunctionType{FunctionTypeString}}),
		WithConformance(ConformanceCaseInsensitiveComparison),
	)

	wantFunctions := []string{"casei", "tolower"}
	if got := parser.SupportedFunctions(); !reflect.DeepEqual(got, wantFunctions) {
		t.Fatalf("SupportedFunctions() = %#v, want %#v", got, wantFunctions)
	}
	if _, err := parser.ParseText(`casei(tolower(name)) = 'alice'`); err != nil {
		t.Fatalf("ParseText with merged functions: %v", err)
	}
}

func TestStandardFunctionsForConformance(t *testing.T) {
	spatial := StandardFunctionsForConformance(ConformanceSpatialFunctions)
	if got, want := functionNames(spatial), []string{"s_contains", "s_crosses", "s_disjoint", "s_equals", "s_intersects", "s_overlaps", "s_touches", "s_within"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("spatial functions = %#v, want %#v", got, want)
	}

	temporal := StandardFunctionsForConformance(ConformanceTemporalFunctions)
	if got, want := len(temporal), 15; got != want {
		t.Fatalf("temporal functions len = %d, want %d", got, want)
	}

	array := StandardFunctionsForConformance(ConformanceArrayFunctions)
	if got, want := functionNames(array), []string{"a_containedby", "a_contains", "a_equals", "a_overlaps"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("array functions = %#v, want %#v", got, want)
	}

	parser := NewParser(WithConformance(ConformanceBasicSpatialFunctions))
	if _, err := parser.ParseText(`TRUE = s_intersects(geom, BBOX(0, 0, 1, 1))`); err != nil {
		t.Fatalf("spatial function in scalar context: %v", err)
	}
}
