package gocql2

import "testing"

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
