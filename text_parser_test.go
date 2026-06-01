package gocql2

import "testing"

func TestParseTextComparisonsAndLogicalPrecedence(t *testing.T) {
	expr, err := ParseText(`a = 1 OR b = 2 AND NOT c = 3`)
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	root, ok := expr.(*LogicalExpression)
	if !ok || root.Op != LogicalOr || len(root.Args) != 2 {
		t.Fatalf("root = %#v, want OR with 2 args", expr)
	}
	right, ok := root.Args[1].(*LogicalExpression)
	if !ok || right.Op != LogicalAnd || len(right.Args) != 2 {
		t.Fatalf("right = %#v, want AND with 2 args", root.Args[1])
	}
	not, ok := right.Args[1].(*LogicalExpression)
	if !ok || not.Op != LogicalNot || len(not.Args) != 1 {
		t.Fatalf("right second = %#v, want NOT", right.Args[1])
	}
}

func TestParseTextEscapedQuotes(t *testing.T) {
	cases := map[string]string{
		`name = 'O''Brien'`: "O'Brien",
		`name = 'O\'Brien'`: "O'Brien",
	}
	for input, want := range cases {
		expr, err := ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		cmp, ok := expr.(*ComparisonExpression)
		if !ok {
			t.Fatalf("expr = %T, want ComparisonExpression", expr)
		}
		lit, ok := cmp.Right.(*Literal)
		if !ok {
			t.Fatalf("right = %T, want Literal", cmp.Right)
		}
		if lit.Value != want {
			t.Fatalf("literal = %#v, want %q", lit.Value, want)
		}
	}
}

func TestParseTextReservedKeywords(t *testing.T) {
	_, err := ParseText(`AND = 1`)
	assertParseErrorContains(t, err, `reserved keyword "AND"`)

	_, err = ParseText(`AND()`)
	assertParseErrorContains(t, err, `reserved keyword "AND"`)

	expr, err := ParseText(`"AND" = 1`)
	if err != nil {
		t.Fatalf("quoted reserved property failed: %v", err)
	}
	cmp, ok := expr.(*ComparisonExpression)
	if !ok {
		t.Fatalf("expr = %T, want ComparisonExpression", expr)
	}
	prop, ok := cmp.Left.(*PropertyRef)
	if !ok {
		t.Fatalf("left = %T, want PropertyRef", cmp.Left)
	}
	if prop.Name != "AND" {
		t.Fatalf("property name = %q, want AND", prop.Name)
	}
}

func TestParseTextLocations(t *testing.T) {
	_, err := ParseText("name =\n  AND")
	assertParseErrorContains(t, err, "line 2, column 3")
}

func TestParseTextRejectsNonBooleanPrimary(t *testing.T) {
	_, err := ParseText(`'not a filter'`)
	assertParseErrorContains(t, err, "expected predicate operator")

	_, err = ParseText(`1`)
	assertParseErrorContains(t, err, "expected predicate operator")
}

func TestParseTextRejectsHyphenatedIdentifier(t *testing.T) {
	_, err := ParseText(`foo-bar = 1`)
	assertParseErrorContains(t, err, "expected predicate operator")
}

func TestParseTextInLists(t *testing.T) {
	cases := []string{
		`status IN ('new', 'done')`,
		`status NOT IN (old_status, current_status)`,
		`flag IN (TRUE, FALSE)`,
		`height IN (min_height + 1, max_height * 2)`,
	}
	for _, input := range cases {
		expr, err := ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		if _, ok := expr.(*InExpression); !ok {
			t.Fatalf("%q parsed as %T, want InExpression", input, expr)
		}
	}

	_, err := ParseText(`status IN ()`)
	assertParseErrorContains(t, err, "IN list must not be empty")

	_, err = ParseText(`status IN ('ok', NULL)`)
	assertParseErrorContains(t, err, "NULL is only allowed")
}

func TestParseTextBetweenNumericExpressions(t *testing.T) {
	cases := []string{
		`height BETWEEN 1 AND 2`,
		`height BETWEEN min_height AND max_height`,
		`height + 1 BETWEEN min_height AND max_height * 2`,
		`height NOT BETWEEN 1 AND limit`,
	}
	for _, input := range cases {
		expr, err := ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		if _, ok := expr.(*BetweenExpression); !ok {
			t.Fatalf("%q parsed as %T, want BetweenExpression", input, expr)
		}
	}

	for _, input := range []string{`'x' BETWEEN 1 AND 2`, `height BETWEEN 'a' AND 2`} {
		_, err := ParseText(input)
		assertParseErrorContains(t, err, "numeric")
	}
}

func TestParseTextLikeLiteralPatterns(t *testing.T) {
	cases := []string{
		`name LIKE '%'`,
		`name NOT LIKE ''`,
		`CASEI(name) LIKE CASEI('foo%')`,
		`ACCENTI(name) LIKE ACCENTI('é%')`,
	}
	for _, input := range cases {
		expr, err := ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		if _, ok := expr.(*LikeExpression); !ok {
			t.Fatalf("%q parsed as %T, want LikeExpression", input, expr)
		}
	}

	for _, input := range []string{`name LIKE other`, `name LIKE custom('x')`, `1 LIKE 'x'`} {
		_, err := ParseText(input)
		assertParseErrorContains(t, err, "LIKE")
	}
}

func TestParseTextBooleanComparisons(t *testing.T) {
	cases := []string{
		`active = TRUE`,
		`FALSE <> archived`,
		`TRUE = FALSE`,
	}
	for _, input := range cases {
		expr, err := ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		if _, ok := expr.(*ComparisonExpression); !ok {
			t.Fatalf("%q parsed as %T, want ComparisonExpression", input, expr)
		}
	}
}

func TestParseTextPredicates(t *testing.T) {
	cases := []struct {
		want  any
		input string
	}{
		{input: `name NOT LIKE 'A%'`, want: &LikeExpression{}},
		{input: `height NOT BETWEEN 1 AND 2`, want: &BetweenExpression{}},
		{input: `status NOT IN ('old')`, want: &InExpression{}},
		{input: `deleted_at IS NULL`, want: &IsNullExpression{}},
		{input: `custom(name, 1)`, want: &FunctionCall{}},
		{input: `CASEI(name) = CASEI('foo')`, want: &ComparisonExpression{}},
		{input: `DATE('2022-04-14') = DATE('2022-04-14')`, want: &ComparisonExpression{}},
		{input: `S_INTERSECTS(geom,BBOX(-180,-90,180,90))`, want: &FunctionCall{}},
	}
	for _, tc := range cases {
		expr, err := ParseText(tc.input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", tc.input, err)
		}
		switch tc.want.(type) {
		case *LikeExpression:
			if _, ok := expr.(*LikeExpression); !ok {
				t.Fatalf("%q parsed as %T, want LikeExpression", tc.input, expr)
			}
		case *BetweenExpression:
			if _, ok := expr.(*BetweenExpression); !ok {
				t.Fatalf("%q parsed as %T, want BetweenExpression", tc.input, expr)
			}
		case *InExpression:
			if _, ok := expr.(*InExpression); !ok {
				t.Fatalf("%q parsed as %T, want InExpression", tc.input, expr)
			}
		case *IsNullExpression:
			if _, ok := expr.(*IsNullExpression); !ok {
				t.Fatalf("%q parsed as %T, want IsNullExpression", tc.input, expr)
			}
		case *FunctionCall:
			if _, ok := expr.(*FunctionCall); !ok {
				t.Fatalf("%q parsed as %T, want FunctionCall", tc.input, expr)
			}
		case *ComparisonExpression:
			if _, ok := expr.(*ComparisonExpression); !ok {
				t.Fatalf("%q parsed as %T, want ComparisonExpression", tc.input, expr)
			}
		}
	}
}

func TestParseTextRejectsInvalidOperandShapes(t *testing.T) {
	cases := map[string]string{
		`name LIKE 1`:                "LIKE pattern must be a character literal",
		`1 LIKE 'x'`:                 "LIKE left operand must be a character expression",
		`height BETWEEN 'a' AND 'z'`: "expected numeric expression",
		`x = NULL`:                   "NULL is only allowed in IS NULL predicates",
	}
	for input, want := range cases {
		_, err := ParseText(input)
		assertParseErrorContains(t, err, want)
	}
}

func TestParseTextDepthLimit(t *testing.T) {
	_, err := ParseText(`NOT NOT NOT a = 1`, WithMaxDepth(2))
	assertParseErrorContains(t, err, "maximum parse depth exceeded")
}
