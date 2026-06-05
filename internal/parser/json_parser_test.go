package parser

import (
	"testing"

	"github.com/cwygoda/cql2/api"
)

func TestParseJSONSemanticPathErrors(t *testing.T) {
	_, err := NewParser().ParseJSON([]byte(`{"op":"and","args":[{"op":"=","args":[{"property":"name"},"a"]},{"op":"=","args":[{"property":123},"b"]}]}`))
	assertParseErrorContains(t, err, "$.args[1].args[0].property")
}

func TestParseJSONSyntaxErrorLocation(t *testing.T) {
	_, err := NewParser().ParseJSON([]byte("{\n  ]"))
	assertParseErrorContains(t, err, "line 2, column 3")
}

func TestParseJSONNumericAlignment(t *testing.T) {
	textExpr, err := NewParser().ParseText(`value = 1.2300E2`)
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	jsonExpr, err := NewParser().ParseJSON([]byte(`{"op":"=","args":[{"property":"value"},123.00]}`))
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	textCmp, ok := textExpr.(*api.ComparisonExpression)
	if !ok {
		t.Fatalf("text expression = %T, want api.ComparisonExpression", textExpr)
	}
	jsonCmp, ok := jsonExpr.(*api.ComparisonExpression)
	if !ok {
		t.Fatalf("json expression = %T, want api.ComparisonExpression", jsonExpr)
	}
	textNum, ok := textCmp.Right.(*api.Literal)
	if !ok {
		t.Fatalf("text right = %T, want api.Literal", textCmp.Right)
	}
	jsonNum, ok := jsonCmp.Right.(*api.Literal)
	if !ok {
		t.Fatalf("json right = %T, want api.Literal", jsonCmp.Right)
	}
	if textNum.Value != jsonNum.Value || textNum.Value != "123" {
		t.Fatalf("numeric values text=%#v json=%#v, want both 123", textNum.Value, jsonNum.Value)
	}
}

func TestParseJSONInLists(t *testing.T) {
	cases := []string{
		`{"op":"in","args":[{"property":"status"},["new","done"]]}`,
		`{"op":"in","args":[{"property":"status"},[{"property":"old_status"},{"property":"current_status"}]]}`,
		`{"op":"in","args":[{"property":"flag"},[true,false]]}`,
		`{"op":"in","args":[{"property":"height"},[{"op":"+","args":[{"property":"min_height"},1]},{"op":"*","args":[{"property":"max_height"},2]}]]}`,
		`{"op":"in","args":[{"property":"status"},[]]}`,
	}
	for _, input := range cases {
		expr, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators, api.ConformancePropertyProperty, api.ConformanceArithmetic).ParseJSON([]byte(input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
		if _, ok := expr.(*api.InExpression); !ok {
			t.Fatalf("%s parsed as %T, want api.InExpression", input, expr)
		}
	}

	_, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).ParseJSON([]byte(`{"op":"in","args":[{"property":"status"},["ok",null]]}`))
	assertParseErrorContains(t, err, "NULL is only allowed")
}

func TestParseJSONArrayPredicates(t *testing.T) {
	cases := []struct {
		input string
		op    api.ArrayPredicateOp
	}{
		{input: `{"op":"a_contains","args":[{"property":"tags"},["foo","bar"]]}`, op: api.ArrayOpContains},
		{input: `{"op":"a_containedBy","args":[[],{"property":"tags"}]}`, op: api.ArrayOpContainedBy},
		{input: `{"op":"a_equals","args":[{"property":"tags"},[1,{"op":"+","args":[2,3]},true]]}`, op: api.ArrayOpEquals},
		{input: `{"op":"a_overlaps","args":[{"op":"get_tags","args":[]},[["nested"],{"op":"=","args":[{"property":"status"},"new"]}]]}`, op: api.ArrayOpOverlaps},
	}
	parser := NewParser().WithConformance(api.ConformanceArrayFunctions, api.ConformanceArithmetic).WithAllowedFunctions(api.FunctionDefinition{
		Name:    "get_tags",
		Returns: []api.FunctionType{api.FunctionTypeArray},
	})

	for _, tc := range cases {
		expr, err := parser.ParseJSON([]byte(tc.input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", tc.input, err)
		}
		array, ok := expr.(*api.ArrayPredicateExpression)
		if !ok || array.Op != tc.op {
			t.Fatalf("%s parsed as %#v, want array predicate %s", tc.input, expr, tc.op)
		}
	}

	_, err := NewParser().ParseJSON([]byte(`{"op":"a_containedby","args":[[],{"property":"tags"}]}`))
	assertParseErrorContains(t, err, `function "a_containedby" is not allowed`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).WithAllowedProperties(api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString}).ParseJSON([]byte(`{"op":"a_contains","args":[{"property":"name"},["foo"]]}`))

	assertParseErrorContains(t, err, `cannot be used as an array operand`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).ParseJSON([]byte(`{"op":"a_contains","args":[{"property":"tags"},"foo"]}`))
	assertParseErrorContains(t, err, `expected array operand`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).ParseJSON([]byte(`{"op":"a_contains","args":[[]]}`))
	assertParseErrorContains(t, err, `expected exactly 2 arguments`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).ParseJSON([]byte(`{"op":"a_contains","args":[{},[]]}`))
	assertParseErrorContains(t, err, `expected scalar expression`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).WithAllowedFunctions(api.FunctionDefinition{
		Name:    "bad_fn",
		Returns: []api.FunctionType{api.FunctionTypeString},
	}).ParseJSON([]byte(`{"op":"a_contains","args":[[],{"op":"bad_fn","args":[]}]}`))

	assertParseErrorContains(t, err, `does not return array`)

	_, err = NewParser().WithConformance(api.ConformanceArrayFunctions).WithMaxDepth(1).ParseJSON([]byte(`{"op":"a_contains","args":[[["foo"]],[]]}`))
	assertParseErrorContains(t, err, `maximum parse depth exceeded`)
}

func TestParseJSONBetweenNumericExpressions(t *testing.T) {
	cases := []string{
		`{"op":"between","args":[{"property":"height"},1,2]}`,
		`{"op":"between","args":[{"property":"height"},{"property":"min_height"},{"property":"max_height"}]}`,
		`{"op":"between","args":[{"op":"+","args":[{"property":"height"},1]},{"property":"min_height"},{"op":"*","args":[{"property":"max_height"},2]}]}`,
	}
	for _, input := range cases {
		expr, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators, api.ConformancePropertyProperty, api.ConformanceArithmetic).ParseJSON([]byte(input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
		if _, ok := expr.(*api.BetweenExpression); !ok {
			t.Fatalf("%s parsed as %T, want api.BetweenExpression", input, expr)
		}
	}

	for _, input := range []string{
		`{"op":"between","args":["x",1,2]}`,
		`{"op":"between","args":[{"property":"height"},"a",2]}`,
	} {
		_, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).ParseJSON([]byte(input))
		assertParseErrorContains(t, err, "numeric")
	}
}

func TestParseJSONLikeLiteralPatterns(t *testing.T) {
	cases := []string{
		`{"op":"like","args":[{"property":"name"},"%"]}`,
		`{"op":"like","args":[{"op":"casei","args":[{"property":"name"}]},{"op":"casei","args":["foo%"]}]}`,
		`{"op":"like","args":[{"op":"accenti","args":[{"property":"name"}]},{"op":"accenti","args":["é%"]}]}`,
	}
	for _, input := range cases {
		expr, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).WithAllowedFunctions(api.StandardTextFunctions()...).ParseJSON([]byte(input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
		if _, ok := expr.(*api.LikeExpression); !ok {
			t.Fatalf("%s parsed as %T, want api.LikeExpression", input, expr)
		}
	}

	for _, input := range []string{
		`{"op":"like","args":[{"property":"name"},{"property":"other"}]}`,
		`{"op":"like","args":[{"property":"name"},{"op":"custom","args":["x"]}]}`,
	} {
		_, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).ParseJSON([]byte(input))
		assertParseErrorContains(t, err, "LIKE")
	}
	_, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).ParseJSON([]byte(`{"op":"like","args":[1,"x"]}`))
	assertParseErrorContains(t, err, "expected character expression")
}

func TestParseJSONIsNullOperands(t *testing.T) {
	cases := []string{
		`{"op":"isNull","args":[{"property":"deleted_at"}]}`,
		`{"op":"isNull","args":[true]}`,
		`{"op":"isNull","args":[{"op":"=","args":[{"property":"a"},1]}]}`,
		`{"op":"isNull","args":[{"op":"+","args":[{"property":"height"},1]}]}`,
		`{"op":"isNull","args":[{"type":"Point","coordinates":[1,2]}]}`,
	}
	for _, input := range cases {
		expr, err := NewParser().WithConformance(api.ConformanceArithmetic).ParseJSON([]byte(input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
		if _, ok := expr.(*api.IsNullExpression); !ok {
			t.Fatalf("%s parsed as %T, want api.IsNullExpression", input, expr)
		}
	}

	_, err := NewParser().ParseJSON([]byte(`{"op":"isNull","args":[[]]}`))
	assertParseErrorContains(t, err, "expected IS NULL operand")
}

func TestParseJSONBooleanComparisons(t *testing.T) {
	cases := []string{
		`{"op":"=","args":[{"property":"active"},true]}`,
		`{"op":"<>","args":[false,{"property":"archived"}]}`,
		`{"op":"=","args":[true,false]}`,
		`{"op":">","args":[{"property":"active"},false]}`,
		`{"op":"<=","args":[true,{"property":"archived"}]}`,
	}
	for _, input := range cases {
		expr, err := NewParser().WithConformance(api.ConformancePropertyProperty).ParseJSON([]byte(input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
		if _, ok := expr.(*api.ComparisonExpression); !ok {
			t.Fatalf("%s parsed as %T, want api.ComparisonExpression", input, expr)
		}
	}
}

func TestParseJSONFunctionsAndArrays(t *testing.T) {
	expr, err := NewParser().WithAllowedFunctions(api.FunctionDefinition{
		Name:    "my_func",
		Args:    []api.FunctionArgument{{Types: []api.FunctionType{api.FunctionTypeAny}, Variadic: true}},
		Returns: []api.FunctionType{api.FunctionTypeBoolean},
	}).ParseJSON([]byte(`{"op":"my_func","args":[{"property":"name"},[1,"x"],true]}`))
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	fn, ok := expr.(*api.FunctionCall)
	if !ok {
		t.Fatalf("expr = %T, want api.FunctionCall", expr)
	}
	if fn.Name != "my_func" || len(fn.Args) != 3 {
		t.Fatalf("function = %#v", fn)
	}
	if _, ok := fn.Args[1].(*api.ArrayLiteral); !ok {
		t.Fatalf("arg[1] = %T, want api.ArrayLiteral", fn.Args[1])
	}

	expr, err = NewParser().WithAllowedFunctions(api.StandardTextFunctions()...).ParseJSON([]byte(`{"op":"=","args":[{"op":"casei","args":[{"property":"name"}]},{"op":"casei","args":["foo"]}]}`))
	if err != nil {
		t.Fatalf("ParseJSON casei comparison: %v", err)
	}
	if _, ok := expr.(*api.ComparisonExpression); !ok {
		t.Fatalf("expr = %T, want api.ComparisonExpression", expr)
	}
}

func TestParseJSONFunctionRegistry(t *testing.T) {
	_, err := NewParser().ParseJSON([]byte(`{"op":"my_func","args":[]}`))
	assertParseErrorContains(t, err, `function "my_func" is not allowed`)

	_, err = NewParser().WithAllowedFunctions(api.CaseIFunction()).ParseJSON([]byte(`{"op":"=","args":[{"property":"x"},{"op":"casei","args":[1]}]}`))
	assertParseErrorContains(t, err, `expected character expression`)

	expr, err := NewParser().WithAllowedFunctions(api.FunctionDefinition{
		Name:    "is_named",
		Args:    []api.FunctionArgument{{Name: "value", Types: []api.FunctionType{api.FunctionTypeString}}},
		Returns: []api.FunctionType{api.FunctionTypeBoolean},
	}).ParseJSON([]byte(`{"op":"is_named","args":[{"property":"name"}]}`))
	if err != nil {
		t.Fatalf("ParseJSON registered function: %v", err)
	}
	fn, ok := expr.(*api.FunctionCall)
	if !ok || fn.Name != "is_named" || !functionCallReturns(fn, api.FunctionTypeBoolean) {
		t.Fatalf("expr = %#v, want registered boolean function", expr)
	}

	_, err = NewParser().WithAllowedFunctions(api.FunctionDefinition{
		Name:    "str_fn",
		Returns: []api.FunctionType{api.FunctionTypeString},
	}).ParseJSON([]byte(`{"op":"str_fn","args":[]}`))

	assertParseErrorContains(t, err, `does not return boolean`)
}

func TestParseJSONArithmetic(t *testing.T) {
	expr, err := NewParser().WithConformance(api.ConformanceArithmetic, api.ConformancePropertyProperty).ParseJSON([]byte(`{"op":">=","args":[{"op":"*","args":[{"op":"+","args":[{"property":"a"},1]},2]},{"op":"div","args":[{"property":"b"},3]}]}`))
	if err != nil {
		t.Fatalf("ParseJSON arithmetic: %v", err)
	}
	cmp, ok := expr.(*api.ComparisonExpression)
	if !ok {
		t.Fatalf("expr = %T, want api.ComparisonExpression", expr)
	}
	left, ok := cmp.Left.(*api.ArithmeticExpression)
	if !ok || left.Op != api.ArithmeticMul {
		t.Fatalf("left = %#v, want multiplication", cmp.Left)
	}
	right, ok := cmp.Right.(*api.ArithmeticExpression)
	if !ok || right.Op != api.ArithmeticIntDiv {
		t.Fatalf("right = %#v, want integer division", cmp.Right)
	}

	_, err = NewParser().ParseJSON([]byte(`{"op":"+","args":[1]}`))
	assertParseErrorContains(t, err, `unsupported reserved operation "+"`)

	_, err = NewParser().WithConformance(api.ConformanceArithmetic).ParseJSON([]byte(`{"op":"=","args":[{"property":"x"},{"op":"+","args":["a",1]}]}`))
	assertParseErrorContains(t, err, "expected numeric expression")
}

func TestParseJSONDepthLimit(t *testing.T) {
	_, err := NewParser().WithMaxDepth(1).ParseJSON([]byte(`{"op":"not","args":[{"op":"not","args":[true]}]}`))
	assertParseErrorContains(t, err, "maximum parse depth exceeded")

	_, err = NewParser().WithMaxDepth(1).WithConformance(api.ConformanceAdvancedComparisonOperators).WithAllowedFunctions(api.CaseIFunction()).ParseJSON([]byte(`{"op":"like","args":[{"property":"name"},{"op":"casei","args":[{"op":"casei","args":["x"]}]}]}`))
	assertParseErrorContains(t, err, "maximum parse depth exceeded")
}

func TestParseJSONRejectsInvalidExpressionLiterals(t *testing.T) {
	_, err := NewParser().ParseJSON([]byte(`"not a filter"`))
	assertParseErrorContains(t, err, "expected CQL2 expression object or boolean")

	_, err = NewParser().ParseJSON([]byte(`1`))
	assertParseErrorContains(t, err, "expected CQL2 expression object or boolean")
}

func TestParseJSONRejectsReservedScalarFunctions(t *testing.T) {
	_, err := NewParser().ParseJSON([]byte(`{"op":"=","args":[{"property":"x"},{"op":"=","args":[1,2]}]}`))
	assertParseErrorContains(t, err, `reserved operation "="`)
}

func TestParseJSONArgumentCounts(t *testing.T) {
	_, err := NewParser().ParseJSON([]byte(`{"op":"=","args":[{"property":"name"}]}`))
	assertParseErrorContains(t, err, "expected exactly 2 arguments")

	_, err = NewParser().ParseJSON([]byte(`{"args":[]}`))
	assertParseErrorContains(t, err, "$.op")

	_, err = NewParser().ParseJSON([]byte(`{"op":"my_func"}`))
	assertParseErrorContains(t, err, "$.args")

	_, err = NewParser().ParseJSON([]byte(`{"op":"my_func","args":null}`))
	assertParseErrorContains(t, err, "expected array")
}

func TestParseJSONRejectsInvalidOperandShapes(t *testing.T) {
	cases := map[string]string{
		`{"op":"=","args":[{"property":"x"},null]}`:             "NULL is only allowed",
		`{"op":"like","args":[{"property":"name"},1]}`:          "LIKE pattern must be a string",
		`{"op":"like","args":[1,"x"]}`:                          "expected character expression",
		`{"op":"between","args":[{"property":"height"},"a",2]}`: "expected numeric expression",
		`{"op":"casei","args":["a","b"]}`:                       "is not a boolean expression",
	}
	for input, want := range cases {
		_, err := NewParser().WithConformance(api.ConformanceAdvancedComparisonOperators).ParseJSON([]byte(input))
		assertParseErrorContains(t, err, want)
	}

	_, err := NewParser().WithAllowedFunctions(api.CaseIFunction()).ParseJSON([]byte(`{"op":"=","args":[{"property":"x"},{"op":"casei","args":[1]}]}`))
	assertParseErrorContains(t, err, "expected character expression")
}
