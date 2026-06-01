package gocql2

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParserCapabilities(t *testing.T) {
	parser := NewParser(
		WithSupportedProperties("name", "height"),
		WithSupportedFunctions("tolower"),
		WithConformanceClasses("/conf/cql2-text/validate"),
	)
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
}

func TestParseDispatchAndJSONPathString(t *testing.T) {
	path := JSONPathRoot().Key("args").Index(1).Key("property")
	if got, want := path.String(), "$.args[1].property"; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}

	expr, err := Parse([]byte(`{"op":"=","args":[{"property":"name"},"alice"]}`), LanguageJSON)
	if err != nil {
		t.Fatalf("Parse JSON: %v", err)
	}
	if _, ok := expr.(*ComparisonExpression); !ok {
		t.Fatalf("Parse JSON type = %T, want *ComparisonExpression", expr)
	}

	expr, err = Parse([]byte(`name = 'alice'`), LanguageText)
	if err != nil {
		t.Fatalf("Parse text: %v", err)
	}
	if _, ok := expr.(*ComparisonExpression); !ok {
		t.Fatalf("Parse text type = %T, want *ComparisonExpression", expr)
	}

	if _, err := Parse([]byte(`true`), Language("cql2-yaml")); err == nil {
		t.Fatal("Parse unsupported language succeeded")
	}
}

func TestParseErrorFormattingAndUnwrap(t *testing.T) {
	cause := errors.New("boom")
	err := &ParseError{Source: LanguageText, Location: Location{Line: 2, Column: 3}, Message: "bad", Expected: []string{"identifier"}, Cause: cause}
	if got, want := err.Error(), "cql2-text parse error at line 2, column 3: bad; expected identifier"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Fatal("ParseError did not unwrap cause")
	}

	jsonErr := &ParseError{Source: LanguageJSON, Location: Location{ByteOffset: -1, CharOffset: -1, JSONPath: JSONPathRoot().Key("op")}, Message: "missing operation"}
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
	}
	for _, tt := range pairs {
		t.Run(tt.name, func(t *testing.T) {
			textExpr, err := ParseText(tt.text)
			if err != nil {
				t.Fatalf("ParseText: %v", err)
			}
			jsonExpr, err := ParseJSON([]byte(tt.json))
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

func semantic(node Node) any {
	switch n := node.(type) {
	case *LogicalExpression:
		args := make([]any, len(n.Args))
		for i, arg := range n.Args {
			args[i] = semantic(arg)
		}
		return map[string]any{"type": "logical", "op": string(n.Op), "args": args}
	case *ComparisonExpression:
		return map[string]any{"type": "comparison", "op": string(n.Op), "left": semantic(n.Left), "right": semantic(n.Right)}
	case *ArithmeticExpression:
		return map[string]any{"type": "arithmetic", "op": string(n.Op), "left": semantic(n.Left), "right": semantic(n.Right)}
	case *LikeExpression:
		return map[string]any{"type": "like", "expr": semantic(n.Expr), "pattern": semantic(n.Pattern), "not": n.Not, "modifier": n.Modifier}
	case *BetweenExpression:
		return map[string]any{"type": "between", "expr": semantic(n.Expr), "lower": semantic(n.Lower), "upper": semantic(n.Upper), "not": n.Not}
	case *InExpression:
		values := make([]any, len(n.Values))
		for i, value := range n.Values {
			values[i] = semantic(value)
		}
		return map[string]any{"type": "in", "expr": semantic(n.Expr), "values": values, "not": n.Not}
	case *IsNullExpression:
		return map[string]any{"type": "isNull", "expr": semantic(n.Expr), "not": n.Not}
	case *PropertyRef:
		return map[string]any{"type": "property", "name": n.Name}
	case *Literal:
		return map[string]any{"type": "literal", "kind": string(n.Kind), "value": n.Value}
	case *FunctionCall:
		args := make([]any, len(n.Args))
		for i, arg := range n.Args {
			args[i] = semantic(arg)
		}
		return map[string]any{"type": "function", "name": n.Name, "args": args}
	case *ArrayLiteral:
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
