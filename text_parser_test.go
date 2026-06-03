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

func TestParseTextParsesHyphenAsArithmetic(t *testing.T) {
	expr, err := ParseText(`foo-bar = 1`)
	if err != nil {
		t.Fatalf("ParseText hyphen arithmetic: %v", err)
	}
	cmp, ok := expr.(*ComparisonExpression)
	if !ok {
		t.Fatalf("expr = %T, want ComparisonExpression", expr)
	}
	arith, ok := cmp.Left.(*ArithmeticExpression)
	if !ok || arith.Op != ArithmeticSub {
		t.Fatalf("left = %#v, want subtraction", cmp.Left)
	}
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

func TestParseTextArrayPredicates(t *testing.T) {
	cases := []struct {
		input string
		op    ArrayPredicateOp
	}{
		{input: `A_CONTAINS(tags, ('foo', 'bar'))`, op: ArrayOpContains},
		{input: `A_CONTAINEDBY((), tags)`, op: ArrayOpContainedBy},
		{input: `A_EQUALS(tags, (1, 2 + count, TRUE))`, op: ArrayOpEquals},
		{input: `A_OVERLAPS(get_tags(), (('nested'), status = 'new'))`, op: ArrayOpOverlaps},
	}
	parser := NewParser(WithAllowedFunctions(FunctionDefinition{
		Name:    "get_tags",
		Returns: []FunctionType{FunctionTypeArray},
	}))
	for _, tc := range cases {
		expr, err := parser.ParseText(tc.input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", tc.input, err)
		}
		array, ok := expr.(*ArrayPredicateExpression)
		if !ok || array.Op != tc.op {
			t.Fatalf("%q parsed as %#v, want array predicate %s", tc.input, expr, tc.op)
		}
		if array.Span().Start.Line != 1 {
			t.Fatalf("Span().Start.Line = %d, want 1", array.Span().Start.Line)
		}
	}

	_, err := ParseText(`A_CONTAINS(name, ('foo'))`, WithAllowedProperties(
		PropertyDefinition{Name: "name", Type: PropertyTypeString},
	))
	assertParseErrorContains(t, err, `cannot be used as an array operand`)

	_, err = ParseText(`A_CONTAINS(tags, 'foo')`)
	assertParseErrorContains(t, err, `expected array operand`)

	_, err = ParseText(`A_CONTAINS(tags, name)`, WithAllowedProperties(
		PropertyDefinition{Name: "tags", Type: PropertyTypeArray},
		PropertyDefinition{Name: "name", Type: PropertyTypeString},
	))
	assertParseErrorContains(t, err, `cannot be used as an array operand`)

	_, err = ParseText(`A_CONTAINS`)
	assertParseErrorContains(t, err, `opening parenthesis`)

	_, err = ParseText(`A_CONTAINS(tags ('foo'))`)
	assertParseErrorContains(t, err, `function "tags" is not allowed`)

	_, err = ParseText(`A_CONTAINS(tags, ('foo')`)
	assertParseErrorContains(t, err, `closing parenthesis`)

	_, err = ParseText(`A_CONTAINS(tags, )`)
	assertParseErrorContains(t, err, `expected scalar expression`)

	_, err = ParseText(`A_CONTAINS(tags, bad_fn())`, WithAllowedFunctions(FunctionDefinition{
		Name:    "bad_fn",
		Returns: []FunctionType{FunctionTypeString},
	}))
	assertParseErrorContains(t, err, `does not return array`)

	_, err = ParseText(`A_CONTAINS(tags, ((('foo'))))`, WithMaxDepth(3))
	assertParseErrorContains(t, err, `maximum parse depth exceeded`)
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
		expr, err := ParseText(input, WithAllowedFunctions(StandardTextFunctions()...))
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

func TestParseTextIsNullOperands(t *testing.T) {
	cases := []string{
		`deleted_at IS NULL`,
		`TRUE IS NOT NULL`,
		`(a = 1) IS NULL`,
		`(height + 1) IS NOT NULL`,
	}
	for _, input := range cases {
		expr, err := ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		if _, ok := expr.(*IsNullExpression); !ok {
			t.Fatalf("%q parsed as %T, want IsNullExpression", input, expr)
		}
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
		{input: `CASEI(name) = CASEI('foo')`, want: &ComparisonExpression{}},
	}
	for _, tc := range cases {
		expr, err := ParseText(tc.input, WithAllowedFunctions(StandardTextFunctions()...))
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

func TestParseTextFunctionRegistry(t *testing.T) {
	expr, err := ParseText(`ACCENTI(CASEI(name)) = accenti(casei('ÄÉ'))`, WithAllowedFunctions(CaseIFunction(), AccentiFunction()))
	if err != nil {
		t.Fatalf("ParseText registered standard text functions: %v", err)
	}
	if _, ok := expr.(*ComparisonExpression); !ok {
		t.Fatalf("expr = %T, want ComparisonExpression", expr)
	}

	_, err = ParseText(`custom(name, 1)`)
	assertParseErrorContains(t, err, `function "custom" is not allowed`)

	_, err = ParseText(`CASEI(1) = '1'`, WithAllowedFunctions(CaseIFunction()))
	assertParseErrorContains(t, err, `expected string`)

	_, err = ParseText(`CASEI(name)`, WithAllowedFunctions(CaseIFunction()))
	assertParseErrorContains(t, err, `expected predicate operator`)

	boolFn := FunctionDefinition{
		Name: "contains_any",
		Args: []FunctionArgument{
			{Name: "value", Types: []FunctionType{FunctionTypeString}},
			{Name: "needle", Types: []FunctionType{FunctionTypeString}, Variadic: true},
		},
		Returns: []FunctionType{FunctionTypeBoolean},
	}
	expr, err = ParseText(`contains_any(name, 'a', 'b')`, WithAllowedFunctions(boolFn))
	if err != nil {
		t.Fatalf("ParseText registered variadic function: %v", err)
	}
	fn, ok := expr.(*FunctionCall)
	if !ok || fn.Name != "contains_any" || len(fn.Args) != 3 || !functionCallReturns(fn, FunctionTypeBoolean) {
		t.Fatalf("expr = %#v, want registered boolean function", expr)
	}

	_, err = ParseText(`contains_any(name, 1)`, WithAllowedFunctions(boolFn))
	assertParseErrorContains(t, err, `expected string`)

	groupedBoolFn := FunctionDefinition{
		Name:    "accept_bool",
		Args:    []FunctionArgument{{Name: "value", Types: []FunctionType{FunctionTypeBoolean}}},
		Returns: []FunctionType{FunctionTypeBoolean},
	}
	expr, err = ParseText(`accept_bool((a = 1))`, WithAllowedFunctions(groupedBoolFn))
	if err != nil {
		t.Fatalf("ParseText grouped boolean function argument: %v", err)
	}
	fn, ok = expr.(*FunctionCall)
	if !ok || len(fn.Args) != 1 {
		t.Fatalf("expr = %#v, want function with one argument", expr)
	}
	if _, comparisonOK := fn.Args[0].(*ComparisonExpression); !comparisonOK {
		t.Fatalf("arg = %T, want grouped boolean expression", fn.Args[0])
	}

	arrayFn := FunctionDefinition{
		Name:    "accept_array",
		Args:    []FunctionArgument{{Name: "value", Types: []FunctionType{FunctionTypeArray}}},
		Returns: []FunctionType{FunctionTypeBoolean},
	}
	expr, err = ParseText(`accept_array(('a', 'b'))`, WithAllowedFunctions(arrayFn))
	if err != nil {
		t.Fatalf("ParseText array function argument: %v", err)
	}
	fn, ok = expr.(*FunctionCall)
	if !ok || len(fn.Args) != 1 {
		t.Fatalf("expr = %#v, want function with one argument", expr)
	}
	if _, ok := fn.Args[0].(*ArrayLiteral); !ok {
		t.Fatalf("arg = %T, want array literal", fn.Args[0])
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

func TestParseTextArithmetic(t *testing.T) {
	expr, err := ParseText(`(a + 1) * 2 >= b DIV -c`)
	if err != nil {
		t.Fatalf("ParseText arithmetic: %v", err)
	}
	cmp, ok := expr.(*ComparisonExpression)
	if !ok {
		t.Fatalf("expr = %T, want ComparisonExpression", expr)
	}
	left, ok := cmp.Left.(*ArithmeticExpression)
	if !ok || left.Op != ArithmeticMul {
		t.Fatalf("left = %#v, want multiplication", cmp.Left)
	}
	add, ok := left.Left.(*ArithmeticExpression)
	if !ok || add.Op != ArithmeticAdd {
		t.Fatalf("left.Left = %#v, want addition", left.Left)
	}
	right, ok := cmp.Right.(*ArithmeticExpression)
	if !ok || right.Op != ArithmeticIntDiv {
		t.Fatalf("right = %#v, want integer division", cmp.Right)
	}
	unary, ok := right.Right.(*ArithmeticExpression)
	if !ok || unary.Op != ArithmeticSub {
		t.Fatalf("right.Right = %#v, want unary minus", right.Right)
	}

	_, err = ParseText(`'x' + 1 = 2`)
	assertParseErrorContains(t, err, "arithmetic operands must be numeric expressions")
}

func TestParseTextDepthLimit(t *testing.T) {
	_, err := ParseText(`NOT NOT NOT a = 1`, WithMaxDepth(2))
	assertParseErrorContains(t, err, "maximum parse depth exceeded")
}
