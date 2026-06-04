package parser

import (
	"testing"

	"github.com/cwygoda/cql2/api"
)

type unknownScalarExpression struct{}

func (*unknownScalarExpression) Span() api.Span { return api.Span{} }

func TestFunctionRegistryValidationEdges(t *testing.T) {
	_, err := ParseText(`casei() = 'x'`, WithAllowedFunctions(api.CaseIFunction()))
	assertParseErrorContains(t, err, `function "casei" expects exactly 1 arguments`)

	variadic := api.FunctionDefinition{
		Name: "any_of",
		Args: []api.FunctionArgument{
			{Name: "value", Types: []api.FunctionType{api.FunctionTypeString}},
			{Name: "candidate", Types: []api.FunctionType{api.FunctionTypeString}, Variadic: true},
		},
		Returns: []api.FunctionType{api.FunctionTypeBoolean},
	}
	_, err = ParseText(`any_of()`, WithAllowedFunctions(variadic))
	assertParseErrorContains(t, err, `function "any_of" expects at least 1 arguments`)

	badDefs := []api.FunctionDefinition{
		{Name: "", Returns: []api.FunctionType{api.FunctionTypeString}},
		{Name: "bad_return", Returns: []api.FunctionType{"object"}},
		{Name: "bad_arg", Args: []api.FunctionArgument{{Types: []api.FunctionType{"object"}}}, Returns: []api.FunctionType{api.FunctionTypeString}},
		{Name: "bad_variadic", Args: []api.FunctionArgument{{Types: []api.FunctionType{api.FunctionTypeString}, Variadic: true}, {Types: []api.FunctionType{api.FunctionTypeString}}}, Returns: []api.FunctionType{api.FunctionTypeString}},
	}
	for _, def := range badDefs {
		if _, err := validateFunctionCall(def.Name, nil, ParseConfig{functions: newFunctionRegistry([]api.FunctionDefinition{def}), MaxDepth: defaultMaxDepth}, api.LanguageText, api.NoLocation()); err == nil {
			t.Fatalf("validateFunctionCall(%q) succeeded, want error", def.Name)
		}
	}

	if _, ok := (functionRegistry{}).lookup("casei"); ok {
		t.Fatal("zero-value function registry allowed standard functions by default")
	}
	if _, ok := newFunctionRegistry([]api.FunctionDefinition{{Name: "", Returns: []api.FunctionType{api.FunctionTypeString}}}).lookup(""); ok {
		t.Fatal("empty function name was registered")
	}
}

func TestFunctionTypeHelpers(t *testing.T) {
	nodes := []api.Node{
		&api.Literal{Kind: api.LiteralNull},
		&api.PropertyRef{Name: "geom", Type: api.PropertyTypeGeometry},
		&api.PropertyRef{Name: "when", Type: api.PropertyTypeTimestamp},
		&api.FunctionCall{Name: "unknown"},
		&api.ArrayLiteral{},
		&api.LogicalExpression{Op: api.LogicalAnd, Args: []api.Expression{&api.Literal{Kind: api.LiteralBool, Value: true}}},
	}
	for _, node := range nodes {
		if got := nodeFunctionTypes(node); len(got) == 0 {
			t.Fatalf("nodeFunctionTypes(%T) returned no types", node)
		}
	}

	propertyTypes := []api.PropertyType{
		api.PropertyTypeAny,
		api.PropertyTypeString,
		api.PropertyTypeNumber,
		api.PropertyTypeInteger,
		api.PropertyTypeBoolean,
		api.PropertyTypeDate,
		api.PropertyTypeTimestamp,
		api.PropertyTypeInterval,
		api.PropertyTypePoint,
		api.PropertyTypeMultiPoint,
		api.PropertyTypeLineString,
		api.PropertyTypeMultiLineString,
		api.PropertyTypePolygon,
		api.PropertyTypeMultiPolygon,
		api.PropertyTypeGeometry,
		api.PropertyTypeGeometryCollection,
		api.PropertyTypeArray,
	}
	for _, typ := range propertyTypes {
		if !isKnownFunctionType(propertyFunctionType(typ)) {
			t.Fatalf("propertyFunctionType(%q) returned unknown function type", typ)
		}
	}
	if got := propertyFunctionType(api.PropertyType("custom")); got != functionTypeUnsupported {
		t.Fatalf("custom property function type = %q, want unsupported", got)
	}

	for _, ret := range []api.FunctionType{api.FunctionTypeString, api.FunctionTypeNumber, api.FunctionTypeInteger, api.FunctionTypeBoolean, api.FunctionTypeDateTime, api.FunctionTypeGeometry, api.FunctionTypeArray, api.FunctionTypeAny} {
		_ = functionReturnPropertyType(&api.FunctionCall{ReturnTypes: []api.FunctionType{ret}})
	}
	if got := functionReturnPropertyType(&api.FunctionCall{ReturnTypes: []api.FunctionType{api.FunctionTypeString, api.FunctionTypeNumber}}); got != api.PropertyTypeAny {
		t.Fatalf("multi-return function property type = %q, want any", got)
	}
	if got := functionReturnPropertyType(&api.FunctionCall{}); got != propertyTypeUnsupported {
		t.Fatalf("empty function return type = %q, want unsupported", got)
	}
	if got := functionReturnPropertyType(&api.FunctionCall{ReturnTypes: []api.FunctionType{api.FunctionType("custom")}}); got != propertyTypeUnsupported {
		t.Fatalf("custom function return type = %q, want unsupported", got)
	}
	if functionTypesOverlap([]api.FunctionType{functionTypeUnsupported}, []api.FunctionType{api.FunctionTypeAny}) {
		t.Fatal("unsupported function type matched any")
	}
	if functionTypesOverlap([]api.FunctionType{api.FunctionType("custom")}, []api.FunctionType{api.FunctionTypeAny}) {
		t.Fatal("custom function type matched any")
	}
	if functionTypesOverlap(nodeFunctionTypes(&unknownScalarExpression{}), []api.FunctionType{api.FunctionTypeAny}) {
		t.Fatal("unknown scalar type matched any function argument")
	}
	if functionTypesOverlap(nodeFunctionTypes(&api.Literal{Kind: api.LiteralNull}), []api.FunctionType{api.FunctionTypeAny}) {
		t.Fatal("NULL literal type matched any function argument")
	}
	assertParseErrorContains(t, validateComparisonOperands(api.OpEqual, nil, &api.Literal{Kind: api.LiteralString}, api.LanguageText), "unsupported")
	assertParseErrorContains(t, validateComparisonOperands(api.OpEqual, &api.FunctionCall{ReturnTypes: []api.FunctionType{api.FunctionType("custom")}}, &api.Literal{Kind: api.LiteralString}, api.LanguageText), "unsupported")

	if got := describeFunctionTypes([]api.FunctionType{api.FunctionTypeAny, api.FunctionTypeString}); got != "any or string" {
		t.Fatalf("describeFunctionTypes = %q", got)
	}
	if cloneFunctionDefinitions(nil) != nil || cloneFunctionTypes(nil) != nil || functionNames(nil) != nil {
		t.Fatal("nil helper inputs should clone to nil")
	}
}

func TestFunctionReturnContexts(t *testing.T) {
	defs := []api.FunctionDefinition{
		{Name: "str_fn", Returns: []api.FunctionType{api.FunctionTypeString}},
		{Name: "num_fn", Returns: []api.FunctionType{api.FunctionTypeNumber}},
		{Name: "bool_fn", Returns: []api.FunctionType{api.FunctionTypeBoolean}},
	}

	if _, err := ParseJSON([]byte(`{"op":"like","args":[{"op":"str_fn","args":[]},"x"]}`), WithConformance(api.ConformanceAdvancedComparisonOperators, api.ConformancePropertyProperty), WithAllowedFunctions(defs...)); err != nil {
		t.Fatalf("string-returning JSON function in character context: %v", err)
	}
	if _, err := ParseJSON([]byte(`{"op":">","args":[{"op":"num_fn","args":[]},1]}`), WithConformance(api.ConformancePropertyProperty), WithAllowedFunctions(defs...)); err != nil {
		t.Fatalf("number-returning JSON function in numeric comparison: %v", err)
	}
	if _, err := ParseText(`num_fn() > 1`, WithConformance(api.ConformancePropertyProperty), WithAllowedFunctions(defs...)); err != nil {
		t.Fatalf("number-returning text function in numeric comparison: %v", err)
	}
	if _, err := ParseText(`bool_fn()`, WithAllowedFunctions(defs...)); err != nil {
		t.Fatalf("boolean-returning text function as expression: %v", err)
	}
}

func TestParseConfigDefaultsForDirectParsers(t *testing.T) {
	if _, err := parseText(`name = 'x'`, ParseConfig{}); err != nil {
		t.Fatalf("parseText with zero ParseConfig: %v", err)
	}
	if _, err := parseJSON([]byte(`{"op":"=","args":[{"property":"name"},"x"]}`), ParseConfig{}); err != nil {
		t.Fatalf("parseJSON with zero ParseConfig: %v", err)
	}
}
