package api

import (
	"errors"
	"reflect"
	"testing"
)

func TestASTNodeMarkersAndSpans(t *testing.T) {
	src := Span{Start: Location{Line: 1, Column: 2}, End: Location{Line: 1, Column: 3}}

	logical := &LogicalExpression{Src: src}
	logical.expressionNode()
	assertSpan(t, logical.Span(), src)

	comparison := &ComparisonExpression{Src: src}
	comparison.expressionNode()
	assertSpan(t, comparison.Span(), src)

	arithmetic := &ArithmeticExpression{Src: src}
	arithmetic.scalarNode()
	assertSpan(t, arithmetic.Span(), src)

	like := &LikeExpression{Src: src}
	like.expressionNode()
	assertSpan(t, like.Span(), src)

	between := &BetweenExpression{Src: src}
	between.expressionNode()
	assertSpan(t, between.Span(), src)

	in := &InExpression{Src: src}
	in.expressionNode()
	assertSpan(t, in.Span(), src)

	isNull := &IsNullExpression{Src: src}
	isNull.expressionNode()
	assertSpan(t, isNull.Span(), src)

	temporalPredicate := &TemporalPredicateExpression{Src: src}
	temporalPredicate.expressionNode()
	assertSpan(t, temporalPredicate.Span(), src)

	temporalInstant := &TemporalInstant{Src: src}
	temporalInstant.scalarNode()
	assertSpan(t, temporalInstant.Span(), src)

	assertSpan(t, (&TemporalUnbounded{Src: src}).Span(), src)
	assertSpan(t, (&TemporalInterval{Src: src}).Span(), src)

	arrayPredicate := &ArrayPredicateExpression{Src: src}
	arrayPredicate.expressionNode()
	assertSpan(t, arrayPredicate.Span(), src)

	spatialPredicate := &SpatialPredicateExpression{Src: src}
	spatialPredicate.expressionNode()
	assertSpan(t, spatialPredicate.Span(), src)

	assertSpan(t, (&GeometryLiteral{Src: src}).Span(), src)

	property := &PropertyRef{Src: src}
	property.scalarNode()
	assertSpan(t, property.Span(), src)

	literal := &Literal{Src: src}
	literal.expressionNode()
	literal.scalarNode()
	assertSpan(t, literal.Span(), src)

	function := &FunctionCall{Src: src}
	function.expressionNode()
	function.scalarNode()
	assertSpan(t, function.Span(), src)

	array := &ArrayLiteral{Src: src}
	array.scalarNode()
	assertSpan(t, array.Span(), src)
}

func TestJSONPathAndParseErrorFormatting(t *testing.T) {
	if got := NoLocation(); got.ByteOffset != -1 || got.CharOffset != -1 {
		t.Fatalf("NoLocation = %#v", got)
	}

	path := JSONPathRoot().Key("op").Index(0).Key("needs escaping")
	if got, want := path.String(), `$.op[0]["needs escaping"]`; got != want {
		t.Fatalf("JSONPath string = %q, want %q", got, want)
	}

	cause := errors.New("cause")
	err := &ParseError{Source: LanguageText, Location: Location{Line: 2, Column: 3}, Message: "bad", Expected: []string{"identifier"}, Cause: cause}
	if got, want := err.Error(), "cql2-text parse error at line 2, column 3: bad; expected identifier"; got != want {
		t.Fatalf("ParseError = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Fatal("ParseError did not unwrap cause")
	}
	if got := (*ParseError)(nil).Error(); got != "<nil>" {
		t.Fatalf("nil ParseError = %q", got)
	}
	if (*ParseError)(nil).Unwrap() != nil {
		t.Fatal("nil ParseError unwrap should be nil")
	}

	jsonErr := &ParseError{Source: LanguageJSON, Location: Location{ByteOffset: -1, CharOffset: -1, JSONPath: path}, Message: "bad"}
	if got, want := jsonErr.Error(), `cql2-json parse error at $.op[0]["needs escaping"]: bad`; got != want {
		t.Fatalf("JSON ParseError = %q, want %q", got, want)
	}

	byteErr := &ParseError{Location: Location{ByteOffset: 5}, Message: "bad"}
	if got, want := byteErr.Error(), "parse error at byte 5: bad"; got != want {
		t.Fatalf("byte ParseError = %q, want %q", got, want)
	}
}

func TestFunctionDefinitions(t *testing.T) {
	casei := CaseIFunction()
	if casei.Name != FunctionNameCaseI || len(casei.Args) != 1 || casei.Returns[0] != FunctionTypeString {
		t.Fatalf("CaseIFunction = %#v", casei)
	}
	accenti := AccentiFunction()
	if accenti.Name != FunctionNameAccenti || len(accenti.Args) != 1 || accenti.Returns[0] != FunctionTypeString {
		t.Fatalf("AccentiFunction = %#v", accenti)
	}
	if got := StandardTextFunctions(); len(got) != 2 {
		t.Fatalf("StandardTextFunctions length = %d", len(got))
	}

	defs := map[string]FunctionDefinition{
		"B": {Name: "B", Args: []FunctionArgument{{Name: "value", Types: []FunctionType{FunctionTypeString}, Variadic: true}}, Returns: []FunctionType{FunctionTypeBoolean}},
		"a": {Name: "a", Returns: []FunctionType{FunctionTypeNumber}},
	}
	cloned := cloneFunctionDefinitions(defs)
	if len(cloned) != 2 || cloned[0].Name != "a" || cloned[1].Name != "b" {
		t.Fatalf("cloneFunctionDefinitions = %#v", cloned)
	}
	cloned[1].Args[0].Types[0] = FunctionTypeNumber
	if defs["B"].Args[0].Types[0] != FunctionTypeString {
		t.Fatal("cloneFunctionDefinitions did not deep clone")
	}
	if cloneFunctionDefinitions(nil) != nil || cloneFunctionTypes(nil) != nil {
		t.Fatal("nil clones should be nil")
	}

	merged := mergeFunctionDefinitions([]FunctionDefinition{{Name: "x", Returns: []FunctionType{FunctionTypeString}}}, []FunctionDefinition{{Name: "X", Returns: []FunctionType{FunctionTypeBoolean}}})
	if len(merged) != 1 || merged[0].Returns[0] != FunctionTypeBoolean {
		t.Fatalf("mergeFunctionDefinitions = %#v", merged)
	}
}

func TestConformanceHelpers(t *testing.T) {
	classes := CanonicalConformanceClasses(
		"case-insensitive-comparison",
		"/conf/array-functions",
		"http://www.opengis.net/spec/cql2/1.0/req/spatial-functions",
		"unknown",
		"",
		"case-insensitive-comparison",
	)
	want := []string{ConformanceCaseInsensitiveComparison, ConformanceArrayFunctions, ConformanceSpatialFunctions, "unknown"}
	if !reflect.DeepEqual(classes, want) {
		t.Fatalf("CanonicalConformanceClasses = %#v, want %#v", classes, want)
	}
	if CanonicalConformanceClasses() != nil {
		t.Fatal("empty conformance classes should be nil")
	}

	defs := StandardFunctionsForConformance(
		ConformanceCaseInsensitiveComparison,
		ConformanceAccentInsensitiveComparison,
		ConformanceBasicSpatialFunctions,
		ConformanceBasicSpatialFunctionsPlus,
		ConformanceSpatialFunctions,
		ConformanceTemporalFunctions,
		ConformanceArrayFunctions,
	)
	seen := map[string]FunctionDefinition{}
	for _, def := range defs {
		seen[def.Name] = def
	}
	for _, name := range []string{FunctionNameCaseI, FunctionNameAccenti, string(SpatialOpIntersects), string(SpatialOpWithin), string(TemporalOpAfter), string(TemporalOpMeets), string(ArrayOpContains)} {
		if _, ok := seen[name]; !ok {
			t.Fatalf("missing standard function %q in %#v", name, defs)
		}
	}
	if got := seen[string(TemporalOpMeets)].Args[0].Types; len(got) != 1 || got[0] != FunctionTypeInterval {
		t.Fatalf("interval-only temporal predicate types = %#v", got)
	}
	if got := seen[string(TemporalOpAfter)].Args[0].Types; len(got) != 2 {
		t.Fatalf("date-time temporal predicate types = %#v", got)
	}
}

func assertSpan(t *testing.T, got, want Span) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Span = %#v, want %#v", got, want)
	}
}
