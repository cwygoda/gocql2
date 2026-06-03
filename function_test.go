package gocql2

import "testing"

func TestFunctionRegistryValidationEdges(t *testing.T) {
	_, err := ParseText(`casei() = 'x'`, WithAllowedFunctions(CaseIFunction()))
	assertParseErrorContains(t, err, `function "casei" expects exactly 1 arguments`)

	variadic := FunctionDefinition{
		Name: "any_of",
		Args: []FunctionArgument{
			{Name: "value", Types: []FunctionType{FunctionTypeString}},
			{Name: "candidate", Types: []FunctionType{FunctionTypeString}, Variadic: true},
		},
		Returns: []FunctionType{FunctionTypeBoolean},
	}
	_, err = ParseText(`any_of()`, WithAllowedFunctions(variadic))
	assertParseErrorContains(t, err, `function "any_of" expects at least 1 arguments`)

	badDefs := []FunctionDefinition{
		{Name: "", Returns: []FunctionType{FunctionTypeString}},
		{Name: "bad_return", Returns: []FunctionType{"object"}},
		{Name: "bad_arg", Args: []FunctionArgument{{Types: []FunctionType{"object"}}}, Returns: []FunctionType{FunctionTypeString}},
		{Name: "bad_variadic", Args: []FunctionArgument{{Types: []FunctionType{FunctionTypeString}, Variadic: true}, {Types: []FunctionType{FunctionTypeString}}}, Returns: []FunctionType{FunctionTypeString}},
	}
	for _, def := range badDefs {
		if _, err := validateFunctionCall(def.Name, nil, ParseConfig{functions: newFunctionRegistry([]FunctionDefinition{def}), MaxDepth: defaultMaxDepth}, LanguageText, NoLocation()); err == nil {
			t.Fatalf("validateFunctionCall(%q) succeeded, want error", def.Name)
		}
	}

	if _, ok := (functionRegistry{}).lookup("casei"); ok {
		t.Fatal("zero-value function registry allowed standard functions by default")
	}
	if _, ok := newFunctionRegistry([]FunctionDefinition{{Name: "", Returns: []FunctionType{FunctionTypeString}}}).lookup(""); ok {
		t.Fatal("empty function name was registered")
	}
}

func TestFunctionTypeHelpers(t *testing.T) {
	nodes := []Node{
		&Literal{Kind: LiteralNull},
		&PropertyRef{Name: "geom", Type: PropertyTypeGeometry},
		&PropertyRef{Name: "when", Type: PropertyTypeTimestamp},
		&FunctionCall{Name: "unknown"},
		&ArrayLiteral{},
		&LogicalExpression{Op: LogicalAnd, Args: []Expression{&Literal{Kind: LiteralBool, Value: true}}},
	}
	for _, node := range nodes {
		if got := nodeFunctionTypes(node); len(got) == 0 {
			t.Fatalf("nodeFunctionTypes(%T) returned no types", node)
		}
	}

	propertyTypes := []PropertyType{
		PropertyTypeAny,
		PropertyTypeString,
		PropertyTypeNumber,
		PropertyTypeInteger,
		PropertyTypeBoolean,
		PropertyTypeDate,
		PropertyTypeTimestamp,
		PropertyTypeInterval,
		PropertyTypePoint,
		PropertyTypeMultiPoint,
		PropertyTypeLineString,
		PropertyTypeMultiLineString,
		PropertyTypePolygon,
		PropertyTypeMultiPolygon,
		PropertyTypeGeometry,
		PropertyTypeGeometryCollection,
		PropertyTypeArray,
		PropertyType("custom"),
	}
	for _, typ := range propertyTypes {
		if !isKnownFunctionType(propertyFunctionType(typ)) {
			t.Fatalf("propertyFunctionType(%q) returned unknown function type", typ)
		}
	}

	for _, ret := range []FunctionType{FunctionTypeString, FunctionTypeNumber, FunctionTypeInteger, FunctionTypeBoolean, FunctionTypeDateTime, FunctionTypeGeometry, FunctionTypeArray, FunctionTypeAny} {
		_ = functionReturnPropertyType(&FunctionCall{ReturnTypes: []FunctionType{ret}})
	}
	if got := functionReturnPropertyType(&FunctionCall{ReturnTypes: []FunctionType{FunctionTypeString, FunctionTypeNumber}}); got != PropertyTypeAny {
		t.Fatalf("multi-return function property type = %q, want any", got)
	}

	if got := describeFunctionTypes([]FunctionType{FunctionTypeAny, FunctionTypeString}); got != "any or string" {
		t.Fatalf("describeFunctionTypes = %q", got)
	}
	if cloneFunctionDefinitions(nil) != nil || cloneFunctionTypes(nil) != nil || functionNames(nil) != nil {
		t.Fatal("nil helper inputs should clone to nil")
	}
}

func TestFunctionReturnContexts(t *testing.T) {
	defs := []FunctionDefinition{
		{Name: "str_fn", Returns: []FunctionType{FunctionTypeString}},
		{Name: "num_fn", Returns: []FunctionType{FunctionTypeNumber}},
		{Name: "bool_fn", Returns: []FunctionType{FunctionTypeBoolean}},
	}

	if _, err := ParseJSON([]byte(`{"op":"like","args":[{"op":"str_fn","args":[]},"x"]}`), WithAllowedFunctions(defs...)); err != nil {
		t.Fatalf("string-returning JSON function in character context: %v", err)
	}
	if _, err := ParseJSON([]byte(`{"op":">","args":[{"op":"num_fn","args":[]},1]}`), WithAllowedFunctions(defs...)); err != nil {
		t.Fatalf("number-returning JSON function in numeric comparison: %v", err)
	}
	if _, err := ParseText(`num_fn() > 1`, WithAllowedFunctions(defs...)); err != nil {
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
