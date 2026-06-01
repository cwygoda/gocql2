package gocql2

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func FuzzParseTextDoesNotPanic(f *testing.F) {
	for _, seed := range []string{
		`name = 'alice'`,
		`name = 'O\'Brien'`,
		`height BETWEEN 1 AND 2`,
		`status IN ('new','done')`,
		`NOT (a = 1 OR b <> 2)`,
		`CASEI(name) = casei('alice')`,
		`"AND" = 1`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if _, err := ParseText(input, WithMaxDepth(32)); err != nil {
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
	parser := NewParser(WithAllowedFunctions(FunctionDefinition{
		Name: "has_text",
		Args: []FunctionArgument{
			{Name: "value", Types: []FunctionType{FunctionTypeString}},
			{Name: "needle", Types: []FunctionType{FunctionTypeString}, Variadic: true},
		},
		Returns: []FunctionType{FunctionTypeBoolean},
	}))
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
		{FunctionNameCaseI, "Alice%"},
		{FunctionNameAccenti, "Äé%"},
		{FunctionNameCaseI, "O'Brien"},
		{FunctionNameAccenti, `slash\\percent%_`},
	} {
		f.Add(seed.fn, seed.value)
	}
	f.Fuzz(func(t *testing.T, fn, value string) {
		fn = strings.ToLower(fn)
		if fn != FunctionNameCaseI && fn != FunctionNameAccenti {
			fn = FunctionNameCaseI
		}

		textLiteral := cqlTextString(value)
		textInputs := []string{
			fmt.Sprintf("%s(name) = %s(%s)", strings.ToUpper(fn), fn, textLiteral),
			fmt.Sprintf("name LIKE %s(%s)", fn, textLiteral),
			fmt.Sprintf("ACCENTI(CASEI(name)) LIKE accenti(casei(%s))", textLiteral),
		}
		for _, input := range textInputs {
			if _, err := ParseText(input, WithMaxDepth(32)); err != nil {
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
			if _, err := ParseJSON([]byte(input), WithMaxDepth(32)); err != nil {
				t.Fatalf("ParseJSON(%s): %v", input, err)
			}
		}
	})
}

func cqlTextString(value string) string {
	return "'" + strings.NewReplacer(`\\`, `\\\\`, `'`, `\'`).Replace(value) + "'"
}

func FuzzParseJSONDoesNotPanic(f *testing.F) {
	for _, seed := range []string{
		`{"op":"=","args":[{"property":"name"},"alice"]}`,
		`{"op":"and","args":[true,{"op":"not","args":[false]}]}`,
		`{"op":"in","args":[{"property":"status"},["new","done"]]}`,
		`{"op":"=","args":[{"op":"casei","args":[{"property":"name"}]},{"op":"casei","args":["alice"]}]}`,
		`null`,
		`{`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if _, err := ParseJSON([]byte(input), WithMaxDepth(32)); err != nil {
			return
		}
	})
}
