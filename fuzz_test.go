package gocql2

import "testing"

func FuzzParseTextDoesNotPanic(f *testing.F) {
	for _, seed := range []string{
		`name = 'alice'`,
		`name = 'O\'Brien'`,
		`height BETWEEN 1 AND 2`,
		`status IN ('new','done')`,
		`NOT (a = 1 OR b <> 2)`,
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

func FuzzParseJSONDoesNotPanic(f *testing.F) {
	for _, seed := range []string{
		`{"op":"=","args":[{"property":"name"},"alice"]}`,
		`{"op":"and","args":[true,{"op":"not","args":[false]}]}`,
		`{"op":"in","args":[{"property":"status"},["new","done"]]}`,
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
