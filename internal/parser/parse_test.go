package parser

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cwygoda/cql2/api"
)

func TestParserCapabilities(t *testing.T) {
	toLower := api.FunctionDefinition{
		Name:    "tolower",
		Args:    []api.FunctionArgument{{Name: "value", Types: []api.FunctionType{api.FunctionTypeString}}},
		Returns: []api.FunctionType{api.FunctionTypeString},
	}
	parser := NewParser().WithAllowedProperties(api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString},
		api.PropertyDefinition{Name: "height", Type: api.PropertyTypeNumber}).WithAllowedFunctions(toLower).WithConformanceClasses("/conf/cql2-text/validate")

	if got, want := parser.SupportedProperties(), []string{"name", "height"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SupportedProperties() = %#v, want %#v", got, want)
	}
	if got, want := parser.SupportedFunctions(), []string{"tolower"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SupportedFunctions() = %#v, want %#v", got, want)
	}
	if got, want := parser.ConformanceClasses(), []string{"/conf/cql2-text/validate"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ConformanceClasses() = %#v, want %#v", got, want)
	}

	props := parser.SupportedProperties()
	props[0] = "mutated"
	if parser.SupportedProperties()[0] != "name" {
		t.Fatal("SupportedProperties exposed internal storage")
	}

	defs := parser.SupportedPropertyDefinitions()
	defs[0].Name = "mutated"
	for _, def := range parser.SupportedPropertyDefinitions() {
		if def.Name == "mutated" {
			t.Fatal("SupportedPropertyDefinitions exposed internal storage")
		}
	}

	functionParser := NewParser().WithAllowedFunctions(api.CaseIFunction(), api.AccentiFunction())
	fnDefs := functionParser.SupportedFunctionDefinitions()
	if got, want := len(fnDefs), 2; got != want {
		t.Fatalf("len(SupportedFunctionDefinitions()) = %d, want %d", got, want)
	}
	fnDefs[0].Name = "mutated"
	fnDefs[0].Args[0].Types[0] = api.FunctionTypeNumber
	for _, def := range functionParser.SupportedFunctionDefinitions() {
		if def.Name == "mutated" || def.Args[0].Types[0] != api.FunctionTypeString {
			t.Fatal("SupportedFunctionDefinitions exposed internal storage")
		}
	}
}

func TestFunctionOptionsMergeWithConformance(t *testing.T) {
	lower := api.FunctionDefinition{
		Name:    "tolower",
		Args:    []api.FunctionArgument{{Name: "value", Types: []api.FunctionType{api.FunctionTypeString}}},
		Returns: []api.FunctionType{api.FunctionTypeString},
	}

	parser := NewParser().WithConformance(api.ConformanceCaseInsensitiveComparison).WithAllowedFunctions(lower)

	if _, err := parser.ParseText(`tolower(CASEI(name)) = tolower(CASEI('ALICE'))`); err != nil {
		t.Fatalf("ParseText with conformance and custom functions: %v", err)
	}

	parser = NewParser().WithConformance(api.ConformanceCaseInsensitiveComparison, api.ConformancePropertyProperty).WithAllowedFunctions(api.FunctionDefinition{
		Name:    "casei",
		Args:    []api.FunctionArgument{{Name: "value", Types: []api.FunctionType{api.FunctionTypeNumber}}},
		Returns: []api.FunctionType{api.FunctionTypeNumber},
	})

	if _, err := parser.ParseText(`CASEI(1) = 1`); err != nil {
		t.Fatalf("ParseText with later custom function override: %v", err)
	}

	contains := api.FunctionDefinition{
		Name:    "contains",
		Args:    []api.FunctionArgument{{Types: []api.FunctionType{api.FunctionTypeAny}, Variadic: true}},
		Returns: []api.FunctionType{api.FunctionTypeBoolean},
	}
	parser = NewParser().WithConformance(api.ConformanceCaseInsensitiveComparison).WithAllowedFunctions(contains)

	if _, err := parser.ParseText(`contains(CASEI(name), CASEI('alice'))`); err != nil {
		t.Fatalf("ParseText with conformance and explicit custom function: %v", err)
	}
}

func TestAllowedPropertyRegistry(t *testing.T) {
	typedProperties := []api.PropertyDefinition{
		{Name: "name", Type: api.PropertyTypeString},
		{Name: "height", Type: api.PropertyTypeNumber},
		{Name: "count", Type: api.PropertyTypeInteger},
		{Name: "active", Type: api.PropertyTypeBoolean},
		{Name: "geometry", Type: api.PropertyTypeGeometry},
	}
	newTypedParser := func() *Parser {
		return NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators, api.ConformancePropertyProperty, api.ConformanceArithmetic).WithAllowedProperties(typedProperties...)
	}

	okCases := []struct {
		name string
		lang api.Language
		in   string
	}{
		{name: "text string", lang: api.LanguageText, in: `name = 'alice'`},
		{name: "text numeric", lang: api.LanguageText, in: `height BETWEEN 1 AND 2`},
		{name: "text integer numeric", lang: api.LanguageText, in: `count + 1 > 2`},
		{name: "text boolean", lang: api.LanguageText, in: `active = TRUE`},
		{name: "json string", lang: api.LanguageJSON, in: `{"op":"like","args":[{"property":"name"},"a%"]}`},
		{name: "json numeric", lang: api.LanguageJSON, in: `{"op":"between","args":[{"property":"height"},1,2]}`},
	}
	for _, tt := range okCases {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := newTypedParser().Parse([]byte(tt.in), tt.lang); err != nil {
				t.Fatalf("Parse: %v", err)
			}
		})
	}

	errorCases := []struct {
		name    string
		lang    api.Language
		in      string
		message string
	}{
		{name: "text unknown", lang: api.LanguageText, in: `missing = 1`, message: `property "missing" is not allowed`},
		{name: "json unknown", lang: api.LanguageJSON, in: `{"op":"=","args":[{"property":"missing"},1]}`, message: `property "missing" is not allowed`},
		{name: "text string as numeric", lang: api.LanguageText, in: `name BETWEEN 1 AND 2`, message: `BETWEEN operands must be numeric expressions`},
		{name: "json string as numeric", lang: api.LanguageJSON, in: `{"op":"between","args":[{"property":"name"},1,2]}`, message: `expected numeric expression`},
		{name: "text number as character", lang: api.LanguageText, in: `height LIKE '1%'`, message: `LIKE left operand must be a character expression`},
		{name: "json number as character", lang: api.LanguageJSON, in: `{"op":"like","args":[{"property":"height"},"1%"]}`, message: `expected character expression`},
		{name: "text comparison mismatch", lang: api.LanguageText, in: `height = 'tall'`, message: `cannot compare number expression to string expression`},
		{name: "json comparison mismatch", lang: api.LanguageJSON, in: `{"op":"=","args":[{"property":"active"},1]}`, message: `cannot compare boolean expression to number expression`},
		{name: "text non scalar", lang: api.LanguageText, in: `geometry = 'POINT (0 0)'`, message: `cannot be used as a scalar expression`},
		{name: "json in mismatch", lang: api.LanguageJSON, in: `{"op":"in","args":[{"property":"name"},["a",1]]}`, message: `IN list value has type number, expected string`},
	}
	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newTypedParser().Parse([]byte(tt.in), tt.lang)
			assertParseErrorContains(t, err, tt.message)
		})
	}

	if _, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).WithAllowedProperties(api.PropertyDefinition{Name: "name", Type: api.PropertyTypeAny}).ParseText(`name BETWEEN 1 AND 2`); err != nil {
		t.Fatalf("explicit any-typed property should remain usable in numeric context: %v", err)
	}
	_, err := NewParser().WithAllowedProperties(api.PropertyDefinition{Name: "name", Type: api.PropertyTypeAny}).ParseText(`other = 1`)
	assertParseErrorContains(t, err, `property "other" is not allowed`)
}

func TestParseDispatchAndJSONPathString(t *testing.T) {
	path := api.JSONPathRoot().Key("args").Index(1).Key("property")
	if got, want := path.String(), "$.args[1].property"; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}

	expr, err := NewParser().Parse([]byte(`{"op":"=","args":[{"property":"name"},"alice"]}`), api.LanguageJSON)
	if err != nil {
		t.Fatalf("Parse JSON: %v", err)
	}
	if _, ok := expr.(*api.ComparisonExpression); !ok {
		t.Fatalf("Parse JSON type = %T, want *api.ComparisonExpression", expr)
	}

	expr, err = NewParser().Parse([]byte(`name = 'alice'`), api.LanguageText)
	if err != nil {
		t.Fatalf("Parse text: %v", err)
	}
	if _, ok := expr.(*api.ComparisonExpression); !ok {
		t.Fatalf("Parse text type = %T, want *api.ComparisonExpression", expr)
	}

	if _, err := NewParser().Parse([]byte(`true`), api.Language("cql2-yaml")); err == nil {
		t.Fatal("Parse unsupported language succeeded")
	}
}

func TestParseErrorFormattingAndUnwrap(t *testing.T) {
	cause := errors.New("boom")
	err := &api.ParseError{Source: api.LanguageText, Location: api.Location{Line: 2, Column: 3}, Message: "bad", Expected: []string{"identifier"}, Cause: cause}
	if got, want := err.Error(), "cql2-text parse error at line 2, column 3: bad; expected identifier"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Fatal("api.ParseError did not unwrap cause")
	}

	jsonErr := &api.ParseError{Source: api.LanguageJSON, Location: api.Location{ByteOffset: -1, CharOffset: -1, JSONPath: api.JSONPathRoot().Key("op")}, Message: "missing operation"}
	if got, want := jsonErr.Error(), "cql2-json parse error at $.op: missing operation"; got != want {
		t.Fatalf("JSON Error() = %q, want %q", got, want)
	}
}

func TestNumericCanonicalization(t *testing.T) {
	cases := map[string]string{
		"1":       "1",
		"+1.0":    "1",
		"001.230": "1.23",
		"1E3":     "1000",
		"1e-3":    "0.001",
		"-.50":    "-0.5",
		"-0.0":    "0",
	}
	for raw, want := range cases {
		got, err := canonicalNumber(raw)
		if err != nil {
			t.Fatalf("canonicalNumber(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("canonicalNumber(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestTextAndJSONParity(t *testing.T) {
	pairs := []struct {
		name string
		text string
		json string
	}{
		{
			name: "logical comparison",
			text: `name = 'alice' AND height >= 1.0`,
			json: `{"op":"and","args":[{"op":"=","args":[{"property":"name"},"alice"]},{"op":">=","args":[{"property":"height"},1]}]}`,
		},
		{
			name: "like modifier",
			text: `name LIKE CASEI('Al%')`,
			json: `{"op":"like","args":[{"property":"name"},{"op":"casei","args":["Al%"]}]}`,
		},
		{
			name: "between",
			text: `height BETWEEN 1.0 AND 2E0`,
			json: `{"op":"between","args":[{"property":"height"},1,2]}`,
		},
		{
			name: "boolean comparison",
			text: `active = TRUE AND FALSE <> archived`,
			json: `{"op":"and","args":[{"op":"=","args":[{"property":"active"},true]},{"op":"<>","args":[false,{"property":"archived"}]}]}`,
		},
		{
			name: "in list",
			text: `status IN ('new', 'done')`,
			json: `{"op":"in","args":[{"property":"status"},["new","done"]]}`,
		},
		{
			name: "is null",
			text: `deleted_at IS NOT NULL`,
			json: `{"op":"isNull","args":[{"property":"deleted_at"}]}`,
		},
		{
			name: "arithmetic comparison",
			text: `(height + 1) * 2 >= other DIV 3`,
			json: `{"op":">=","args":[{"op":"*","args":[{"op":"+","args":[{"property":"height"},1]},2]},{"op":"div","args":[{"property":"other"},3]}]}`,
		},
		{
			name: "array predicate",
			text: `A_CONTAINS(tags, ('foo', 'bar'))`,
			json: `{"op":"a_contains","args":[{"property":"tags"},["foo","bar"]]}`,
		},
	}
	for _, tt := range pairs {
		t.Run(tt.name, func(t *testing.T) {
			textExpr, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators, api.ConformancePropertyProperty, api.ConformanceArithmetic, api.ConformanceArrayFunctions).WithAllowedFunctions(api.StandardTextFunctions()...).ParseText(tt.text)
			if err != nil {
				t.Fatalf("ParseText: %v", err)
			}
			jsonExpr, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators, api.ConformancePropertyProperty, api.ConformanceArithmetic, api.ConformanceArrayFunctions).WithAllowedFunctions(api.StandardTextFunctions()...).ParseJSON([]byte(tt.json))
			if err != nil {
				t.Fatalf("ParseJSON: %v", err)
			}
			textSem := semantic(textExpr)
			jsonSem := semantic(jsonExpr)
			if tt.name == "is null" {
				// CQL2 JSON has no negated isNull op; text keeps the NOT flag.
				textMap, ok := textSem.(map[string]any)
				if !ok {
					t.Fatalf("text semantic value = %T, want map", textSem)
				}
				textMap["not"] = false
			}
			if !reflect.DeepEqual(textSem, jsonSem) {
				t.Fatalf("semantic mismatch\ntext: %#v\njson: %#v", textSem, jsonSem)
			}
		})
	}
}

func semantic(node api.Node) any {
	switch n := node.(type) {
	case *api.LogicalExpression:
		args := make([]any, len(n.Args))
		for i, arg := range n.Args {
			args[i] = semantic(arg)
		}
		return map[string]any{"type": "logical", "op": string(n.Op), "args": args}
	case *api.ComparisonExpression:
		return map[string]any{"type": "comparison", "op": string(n.Op), "left": semantic(n.Left), "right": semantic(n.Right)}
	case *api.ArithmeticExpression:
		return map[string]any{"type": "arithmetic", "op": string(n.Op), "left": semantic(n.Left), "right": semantic(n.Right)}
	case *api.LikeExpression:
		return map[string]any{"type": "like", "expr": semantic(n.Expr), "pattern": semantic(n.Pattern), "not": n.Not}
	case *api.BetweenExpression:
		return map[string]any{"type": "between", "expr": semantic(n.Expr), "lower": semantic(n.Lower), "upper": semantic(n.Upper), "not": n.Not}
	case *api.InExpression:
		values := make([]any, len(n.Values))
		for i, value := range n.Values {
			values[i] = semantic(value)
		}
		return map[string]any{"type": "in", "expr": semantic(n.Expr), "values": values, "not": n.Not}
	case *api.IsNullExpression:
		return map[string]any{"type": "isNull", "expr": semantic(n.Expr), "not": n.Not}
	case *api.ArrayPredicateExpression:
		return map[string]any{"type": "arrayPredicate", "op": string(n.Op), "left": semantic(n.Left), "right": semantic(n.Right)}
	case *api.TemporalPredicateExpression:
		return map[string]any{"type": "temporalPredicate", "op": string(n.Op), "left": semantic(n.Left), "right": semantic(n.Right)}
	case *api.TemporalInstant:
		return map[string]any{"type": "temporalInstant", "kind": string(n.Kind), "value": n.Value}
	case *api.TemporalUnbounded:
		return map[string]any{"type": "temporalUnbounded"}
	case *api.TemporalInterval:
		return map[string]any{"type": "temporalInterval", "start": semantic(n.Start), "end": semantic(n.End)}
	case *api.PropertyRef:
		return map[string]any{"type": "property", "name": n.Name}
	case *api.Literal:
		return map[string]any{"type": "literal", "kind": string(n.Kind), "value": n.Value}
	case *api.FunctionCall:
		args := make([]any, len(n.Args))
		for i, arg := range n.Args {
			args[i] = semantic(arg)
		}
		return map[string]any{"type": "function", "name": n.Name, "args": args}
	case *api.ArrayLiteral:
		values := make([]any, len(n.Values))
		for i, value := range n.Values {
			values[i] = semantic(value)
		}
		return map[string]any{"type": "array", "values": values}
	default:
		return map[string]any{"type": "unknown"}
	}
}

func assertParseErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("error = %q, want substring %q", err.Error(), substr)
	}
}
