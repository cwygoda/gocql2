package parser

import (
	"testing"

	"github.com/cwygoda/cql2/api"
)

func TestParseTextComparisonsAndLogicalPrecedence(t *testing.T) {
	expr, err := NewParser().ParseText(`a = 1 OR b = 2 AND NOT c = 3`)
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	root, ok := expr.(*api.LogicalExpression)
	if !ok || root.Op != api.LogicalOr || len(root.Args) != 2 {
		t.Fatalf("root = %#v, want OR with 2 args", expr)
	}
	right, ok := root.Args[1].(*api.LogicalExpression)
	if !ok || right.Op != api.LogicalAnd || len(right.Args) != 2 {
		t.Fatalf("right = %#v, want AND with 2 args", root.Args[1])
	}
	not, ok := right.Args[1].(*api.LogicalExpression)
	if !ok || not.Op != api.LogicalNot || len(not.Args) != 1 {
		t.Fatalf("right second = %#v, want NOT", right.Args[1])
	}
}

func TestParseTextEscapedQuotes(t *testing.T) {
	cases := map[string]string{
		`name = 'O''Brien'`: "O'Brien",
		`name = 'O\'Brien'`: "O'Brien",
	}
	for input, want := range cases {
		expr, err := NewParser().ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		cmp, ok := expr.(*api.ComparisonExpression)
		if !ok {
			t.Fatalf("expr = %T, want api.ComparisonExpression", expr)
		}
		lit, ok := cmp.Right.(*api.Literal)
		if !ok {
			t.Fatalf("right = %T, want api.Literal", cmp.Right)
		}
		if lit.Value != want {
			t.Fatalf("literal = %#v, want %q", lit.Value, want)
		}
	}
}

func TestParseTextReservedKeywords(t *testing.T) {
	_, err := NewParser().ParseText(`AND = 1`)
	assertParseErrorContains(t, err, `reserved keyword "AND"`)

	_, err = NewParser().ParseText(`AND()`)
	assertParseErrorContains(t, err, `reserved keyword "AND"`)

	expr, err := NewParser().ParseText(`"AND" = 1`)
	if err != nil {
		t.Fatalf("quoted reserved property failed: %v", err)
	}
	cmp, ok := expr.(*api.ComparisonExpression)
	if !ok {
		t.Fatalf("expr = %T, want api.ComparisonExpression", expr)
	}
	prop, ok := cmp.Left.(*api.PropertyRef)
	if !ok {
		t.Fatalf("left = %T, want api.PropertyRef", cmp.Left)
	}
	if prop.Name != "AND" {
		t.Fatalf("property name = %q, want AND", prop.Name)
	}
}

func TestParseTextIdentifierGrammar(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "a\u0301 = 1", want: "a\u0301"},               // combining mark as identifier part
		{input: "a\u203fb = 1", want: "a\u203fb"},             // undertie connector punctuation
		{input: "\u200cname = 1", want: "\u200cname"},         // zero-width non-joiner as start
		{input: "\u3001name = 1", want: "\u3001name"},         // CJK punctuation range as start
		{input: "\U00010000name = 1", want: "\U00010000name"}, // supplementary-plane range as start
		{input: "\U0001f600name = 1", want: "\U0001f600name"}, // symbol in supplementary-plane range
		{input: "\"a.b\u0301\u2040c\" = 1", want: "a.b\u0301\u2040c"},
	}
	for _, tc := range cases {
		expr, err := NewParser().ParseText(tc.input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", tc.input, err)
		}
		cmp, ok := expr.(*api.ComparisonExpression)
		if !ok {
			t.Fatalf("%q parsed as %T, want api.ComparisonExpression", tc.input, expr)
		}
		prop, ok := cmp.Left.(*api.PropertyRef)
		if !ok {
			t.Fatalf("%q left = %T, want api.PropertyRef", tc.input, cmp.Left)
		}
		if prop.Name != tc.want {
			t.Fatalf("%q property name = %q, want %q", tc.input, prop.Name, tc.want)
		}
	}
}

func TestParseTextRejectsInvalidQuotedIdentifiers(t *testing.T) {
	cases := map[string]string{
		`"" = 1`:          "quoted identifier must not be empty",
		`"1abc" = 1`:      "invalid quoted identifier start character",
		`".abc" = 1`:      "invalid quoted identifier start character",
		`"has space" = 1`: "invalid quoted identifier character",
		`"a-b" = 1`:       "invalid quoted identifier character",
		`"a/b" = 1`:       "invalid quoted identifier character",
	}
	for input, want := range cases {
		_, err := NewParser().ParseText(input)
		assertParseErrorContains(t, err, want)
	}
}

func TestParseTextLocations(t *testing.T) {
	_, err := NewParser().ParseText("name =\n  AND")
	assertParseErrorContains(t, err, "line 2, column 3")
}

func TestParseTextRejectsNonBooleanPrimary(t *testing.T) {
	_, err := NewParser().ParseText(`'not a filter'`)
	assertParseErrorContains(t, err, "expected predicate operator")

	_, err = NewParser().ParseText(`1`)
	assertParseErrorContains(t, err, "expected predicate operator")
}

func TestParseTextParsesAdjacentArithmeticOperators(t *testing.T) {
	cases := []struct {
		input string
		op    api.ArithmeticOp
	}{
		{input: `foo-bar = 1`, op: api.ArithmeticSub},
		{input: `a-1=2`, op: api.ArithmeticSub},
		{input: `a - 1 = 2`, op: api.ArithmeticSub},
		{input: `1-1=0`, op: api.ArithmeticSub},
		{input: `1 - 1 = 0`, op: api.ArithmeticSub},
		{input: `a+1=2`, op: api.ArithmeticAdd},
		{input: `a + 1 = 2`, op: api.ArithmeticAdd},
		{input: `1+1=2`, op: api.ArithmeticAdd},
		{input: `1 + 1 = 2`, op: api.ArithmeticAdd},
	}
	for _, tc := range cases {
		expr, err := NewParser().WithConformance(api.ConformanceArithmetic, api.ConformancePropertyProperty).ParseText(tc.input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", tc.input, err)
		}
		cmp, ok := expr.(*api.ComparisonExpression)
		if !ok {
			t.Fatalf("%q parsed as %T, want api.ComparisonExpression", tc.input, expr)
		}
		arith, ok := cmp.Left.(*api.ArithmeticExpression)
		if !ok || arith.Op != tc.op {
			t.Fatalf("%q left = %#v, want %s", tc.input, cmp.Left, tc.op)
		}
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
		expr, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators, api.ConformancePropertyProperty, api.ConformanceArithmetic).ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		if _, ok := expr.(*api.InExpression); !ok {
			t.Fatalf("%q parsed as %T, want api.InExpression", input, expr)
		}
	}

	_, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).ParseText(`status IN ()`)
	assertParseErrorContains(t, err, "IN list must not be empty")

	_, err = NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).ParseText(`status IN ('ok', NULL)`)
	assertParseErrorContains(t, err, "NULL is only allowed")
}

func TestParseTextArrayPredicates(t *testing.T) {
	cases := []struct {
		input string
		op    api.ArrayPredicateOp
	}{
		{input: `A_CONTAINS(tags, ('foo', 'bar'))`, op: api.ArrayOpContains},
		{input: `A_CONTAINS(tags, (POINT(1 2)))`, op: api.ArrayOpContains},
		{input: `A_CONTAINEDBY((), tags)`, op: api.ArrayOpContainedBy},
		{input: `A_EQUALS(tags, (1, 2 + count, TRUE))`, op: api.ArrayOpEquals},
		{input: `A_OVERLAPS(get_tags(), (('nested'), status = 'new'))`, op: api.ArrayOpOverlaps},
	}
	parser := NewParser().WithConformance(api.ConformanceArrayFunctions, api.ConformanceArithmetic).WithAllowedFunctions(api.FunctionDefinition{
		Name:    "get_tags",
		Returns: []api.FunctionType{api.FunctionTypeArray},
	})

	for _, tc := range cases {
		expr, err := parser.ParseText(tc.input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", tc.input, err)
		}
		array, ok := expr.(*api.ArrayPredicateExpression)
		if !ok || array.Op != tc.op {
			t.Fatalf("%q parsed as %#v, want array predicate %s", tc.input, expr, tc.op)
		}
		if array.Span().Start.Line != 1 {
			t.Fatalf("api.Span().Start.Line = %d, want 1", array.Span().Start.Line)
		}
	}

	_, err := NewParser().WithConformance(api.ConformanceArrayFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString}).ParseText(`A_CONTAINS(name, ('foo'))`)

	assertParseErrorContains(t, err, `cannot be used as an array operand`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).ParseText(`A_CONTAINS(tags, 'foo')`)
	assertParseErrorContains(t, err, `expected array operand`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "tags", Type: api.PropertyTypeArray},
		api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString}).ParseText(`A_CONTAINS(tags, name)`)

	assertParseErrorContains(t, err, `cannot be used as an array operand`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).ParseText(`A_CONTAINS`)
	assertParseErrorContains(t, err, `opening parenthesis`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).ParseText(`A_CONTAINS(tags ('foo'))`)
	assertParseErrorContains(t, err, `function "tags" is not allowed`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).ParseText(`A_CONTAINS(tags, ('foo')`)
	assertParseErrorContains(t, err, `closing parenthesis`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).ParseText(`A_CONTAINS(tags, )`)
	assertParseErrorContains(t, err, `expected scalar expression`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).WithAllowedFunctions(api.FunctionDefinition{
		Name:    "bad_fn",
		Returns: []api.FunctionType{api.FunctionTypeString},
	}).ParseText(`A_CONTAINS(tags, bad_fn())`)

	assertParseErrorContains(t, err, `does not return array`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).WithMaxDepth(3).ParseText(`A_CONTAINS(tags, ((('foo'))))`)
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
		expr, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators, api.ConformancePropertyProperty, api.ConformanceArithmetic).ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		if _, ok := expr.(*api.BetweenExpression); !ok {
			t.Fatalf("%q parsed as %T, want api.BetweenExpression", input, expr)
		}
	}

	for _, input := range []string{`'x' BETWEEN 1 AND 2`, `height BETWEEN 'a' AND 2`} {
		_, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).ParseText(input)
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
		expr, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).WithAllowedFunctions(api.StandardTextFunctions()...).ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		if _, ok := expr.(*api.LikeExpression); !ok {
			t.Fatalf("%q parsed as %T, want api.LikeExpression", input, expr)
		}
	}

	for _, input := range []string{`name LIKE other`, `name LIKE custom('x')`, `1 LIKE 'x'`} {
		_, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).ParseText(input)
		assertParseErrorContains(t, err, "LIKE")
	}
}

func TestParseTextIsNullOperands(t *testing.T) {
	cases := []string{
		`deleted_at IS NULL`,
		`TRUE IS NOT NULL`,
		`(a = 1) IS NULL`,
		`(height + 1) IS NOT NULL`,
		`POINT(1 2) IS NULL`,
	}
	for _, input := range cases {
		expr, err := NewParser().WithConformance(api.ConformanceArithmetic).ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		if _, ok := expr.(*api.IsNullExpression); !ok {
			t.Fatalf("%q parsed as %T, want api.IsNullExpression", input, expr)
		}
	}
}

func TestParseTextBooleanComparisons(t *testing.T) {
	cases := []string{
		`active = TRUE`,
		`FALSE <> archived`,
		`TRUE = FALSE`,
		`active > FALSE`,
		`TRUE <= archived`,
	}
	for _, input := range cases {
		expr, err := NewParser().WithConformance(api.ConformancePropertyProperty).ParseText(input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
		if _, ok := expr.(*api.ComparisonExpression); !ok {
			t.Fatalf("%q parsed as %T, want api.ComparisonExpression", input, expr)
		}
	}
}

func TestParseTextPredicates(t *testing.T) {
	cases := []struct {
		want  any
		input string
	}{
		{input: `name NOT LIKE 'A%'`, want: &api.LikeExpression{}},
		{input: `height NOT BETWEEN 1 AND 2`, want: &api.BetweenExpression{}},
		{input: `status NOT IN ('old')`, want: &api.InExpression{}},
		{input: `deleted_at IS NULL`, want: &api.IsNullExpression{}},
		{input: `CASEI(name) = CASEI('foo')`, want: &api.ComparisonExpression{}},
	}
	for _, tc := range cases {
		expr, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).WithAllowedFunctions(api.StandardTextFunctions()...).ParseText(tc.input)
		if err != nil {
			t.Fatalf("ParseText(%q): %v", tc.input, err)
		}
		switch tc.want.(type) {
		case *api.LikeExpression:
			if _, ok := expr.(*api.LikeExpression); !ok {
				t.Fatalf("%q parsed as %T, want api.LikeExpression", tc.input, expr)
			}
		case *api.BetweenExpression:
			if _, ok := expr.(*api.BetweenExpression); !ok {
				t.Fatalf("%q parsed as %T, want api.BetweenExpression", tc.input, expr)
			}
		case *api.InExpression:
			if _, ok := expr.(*api.InExpression); !ok {
				t.Fatalf("%q parsed as %T, want api.InExpression", tc.input, expr)
			}
		case *api.IsNullExpression:
			if _, ok := expr.(*api.IsNullExpression); !ok {
				t.Fatalf("%q parsed as %T, want api.IsNullExpression", tc.input, expr)
			}
		case *api.FunctionCall:
			if _, ok := expr.(*api.FunctionCall); !ok {
				t.Fatalf("%q parsed as %T, want api.FunctionCall", tc.input, expr)
			}
		case *api.ComparisonExpression:
			if _, ok := expr.(*api.ComparisonExpression); !ok {
				t.Fatalf("%q parsed as %T, want api.ComparisonExpression", tc.input, expr)
			}
		}
	}
}

func TestParseTextFunctionRegistry(t *testing.T) {
	expr, err := NewParser().WithAllowedFunctions(api.CaseIFunction(), api.AccentiFunction()).ParseText(`ACCENTI(CASEI(name)) = accenti(casei('ÄÉ'))`)
	if err != nil {
		t.Fatalf("ParseText registered standard text functions: %v", err)
	}
	if _, ok := expr.(*api.ComparisonExpression); !ok {
		t.Fatalf("expr = %T, want api.ComparisonExpression", expr)
	}

	_, err = NewParser().ParseText(`custom(name, 1)`)
	assertParseErrorContains(t, err, `function "custom" is not allowed`)

	_, err = NewParser().WithAllowedFunctions(api.CaseIFunction()).ParseText(`CASEI(1) = '1'`)
	assertParseErrorContains(t, err, `expected string`)

	_, err = NewParser().WithAllowedFunctions(api.CaseIFunction()).ParseText(`CASEI(name)`)
	assertParseErrorContains(t, err, `expected predicate operator`)

	boolFn := api.FunctionDefinition{
		Name: "contains_any",
		Args: []api.FunctionArgument{
			{Name: "value", Types: []api.FunctionType{api.FunctionTypeString}},
			{Name: "needle", Types: []api.FunctionType{api.FunctionTypeString}, Variadic: true},
		},
		Returns: []api.FunctionType{api.FunctionTypeBoolean},
	}
	expr, err = NewParser().WithAllowedFunctions(boolFn).ParseText(`contains_any(name, 'a', 'b')`)
	if err != nil {
		t.Fatalf("ParseText registered variadic function: %v", err)
	}
	fn, ok := expr.(*api.FunctionCall)
	if !ok || fn.Name != "contains_any" || len(fn.Args) != 3 || !functionCallReturns(fn, api.FunctionTypeBoolean) {
		t.Fatalf("expr = %#v, want registered boolean function", expr)
	}

	_, err = NewParser().WithAllowedFunctions(boolFn).ParseText(`contains_any(name, 1)`)
	assertParseErrorContains(t, err, `expected string`)

	groupedBoolFn := api.FunctionDefinition{
		Name:    "accept_bool",
		Args:    []api.FunctionArgument{{Name: "value", Types: []api.FunctionType{api.FunctionTypeBoolean}}},
		Returns: []api.FunctionType{api.FunctionTypeBoolean},
	}
	expr, err = NewParser().WithAllowedFunctions(groupedBoolFn).ParseText(`accept_bool((a = 1))`)
	if err != nil {
		t.Fatalf("ParseText grouped boolean function argument: %v", err)
	}
	fn, ok = expr.(*api.FunctionCall)
	if !ok || len(fn.Args) != 1 {
		t.Fatalf("expr = %#v, want function with one argument", expr)
	}
	if _, comparisonOK := fn.Args[0].(*api.ComparisonExpression); !comparisonOK {
		t.Fatalf("arg = %T, want grouped boolean expression", fn.Args[0])
	}

	arrayFn := api.FunctionDefinition{
		Name:    "accept_array",
		Args:    []api.FunctionArgument{{Name: "value", Types: []api.FunctionType{api.FunctionTypeArray}}},
		Returns: []api.FunctionType{api.FunctionTypeBoolean},
	}
	expr, err = NewParser().WithAllowedFunctions(arrayFn).ParseText(`accept_array(('a', 'b'))`)
	if err != nil {
		t.Fatalf("ParseText array function argument: %v", err)
	}
	fn, ok = expr.(*api.FunctionCall)
	if !ok || len(fn.Args) != 1 {
		t.Fatalf("expr = %#v, want function with one argument", expr)
	}
	if _, ok := fn.Args[0].(*api.ArrayLiteral); !ok {
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
		_, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).ParseText(input)
		assertParseErrorContains(t, err, want)
	}
}

func TestParseTextArithmetic(t *testing.T) {
	expr, err := NewParser().WithConformance(api.ConformanceArithmetic, api.ConformancePropertyProperty).ParseText(`(a + 1) * 2 >= b DIV -c`)
	if err != nil {
		t.Fatalf("ParseText arithmetic: %v", err)
	}
	cmp, ok := expr.(*api.ComparisonExpression)
	if !ok {
		t.Fatalf("expr = %T, want api.ComparisonExpression", expr)
	}
	left, ok := cmp.Left.(*api.ArithmeticExpression)
	if !ok || left.Op != api.ArithmeticMul {
		t.Fatalf("left = %#v, want multiplication", cmp.Left)
	}
	add, ok := left.Left.(*api.ArithmeticExpression)
	if !ok || add.Op != api.ArithmeticAdd {
		t.Fatalf("left.Left = %#v, want addition", left.Left)
	}
	right, ok := cmp.Right.(*api.ArithmeticExpression)
	if !ok || right.Op != api.ArithmeticIntDiv {
		t.Fatalf("right = %#v, want integer division", cmp.Right)
	}
	unary, ok := right.Right.(*api.ArithmeticExpression)
	if !ok || unary.Op != api.ArithmeticSub {
		t.Fatalf("right.Right = %#v, want unary minus", right.Right)
	}

	_, err = NewParser().WithConformance(api.ConformanceArithmetic, api.ConformancePropertyProperty).ParseText(`'x' + 1 = 2`)
	assertParseErrorContains(t, err, "arithmetic operands must be numeric expressions")
}

func TestParseTextDepthLimit(t *testing.T) {
	_, err := NewParser().WithMaxDepth(2).ParseText(`NOT NOT NOT a = 1`)
	assertParseErrorContains(t, err, "maximum parse depth exceeded")
}
