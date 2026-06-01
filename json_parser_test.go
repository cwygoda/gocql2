package gocql2

import "testing"

func TestParseJSONSemanticPathErrors(t *testing.T) {
	_, err := ParseJSON([]byte(`{"op":"and","args":[{"op":"=","args":[{"property":"name"},"a"]},{"op":"=","args":[{"property":123},"b"]}]}`))
	assertParseErrorContains(t, err, "$.args[1].args[0].property")
}

func TestParseJSONSyntaxErrorLocation(t *testing.T) {
	_, err := ParseJSON([]byte("{\n  ]"))
	assertParseErrorContains(t, err, "line 2, column 3")
}

func TestParseJSONNumericAlignment(t *testing.T) {
	textExpr, err := ParseText(`value = 1.2300E2`)
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	jsonExpr, err := ParseJSON([]byte(`{"op":"=","args":[{"property":"value"},123.00]}`))
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	textCmp, ok := textExpr.(*ComparisonExpression)
	if !ok {
		t.Fatalf("text expression = %T, want ComparisonExpression", textExpr)
	}
	jsonCmp, ok := jsonExpr.(*ComparisonExpression)
	if !ok {
		t.Fatalf("json expression = %T, want ComparisonExpression", jsonExpr)
	}
	textNum, ok := textCmp.Right.(*Literal)
	if !ok {
		t.Fatalf("text right = %T, want Literal", textCmp.Right)
	}
	jsonNum, ok := jsonCmp.Right.(*Literal)
	if !ok {
		t.Fatalf("json right = %T, want Literal", jsonCmp.Right)
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
		expr, err := ParseJSON([]byte(input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
		if _, ok := expr.(*InExpression); !ok {
			t.Fatalf("%s parsed as %T, want InExpression", input, expr)
		}
	}

	_, err := ParseJSON([]byte(`{"op":"in","args":[{"property":"status"},["ok",null]]}`))
	assertParseErrorContains(t, err, "NULL is only allowed")
}

func TestParseJSONBetweenNumericExpressions(t *testing.T) {
	cases := []string{
		`{"op":"between","args":[{"property":"height"},1,2]}`,
		`{"op":"between","args":[{"property":"height"},{"property":"min_height"},{"property":"max_height"}]}`,
		`{"op":"between","args":[{"op":"+","args":[{"property":"height"},1]},{"property":"min_height"},{"op":"*","args":[{"property":"max_height"},2]}]}`,
	}
	for _, input := range cases {
		expr, err := ParseJSON([]byte(input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
		if _, ok := expr.(*BetweenExpression); !ok {
			t.Fatalf("%s parsed as %T, want BetweenExpression", input, expr)
		}
	}

	for _, input := range []string{
		`{"op":"between","args":["x",1,2]}`,
		`{"op":"between","args":[{"property":"height"},"a",2]}`,
	} {
		_, err := ParseJSON([]byte(input))
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
		expr, err := ParseJSON([]byte(input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
		if _, ok := expr.(*LikeExpression); !ok {
			t.Fatalf("%s parsed as %T, want LikeExpression", input, expr)
		}
	}

	for _, input := range []string{
		`{"op":"like","args":[{"property":"name"},{"property":"other"}]}`,
		`{"op":"like","args":[{"property":"name"},{"op":"custom","args":["x"]}]}`,
	} {
		_, err := ParseJSON([]byte(input))
		assertParseErrorContains(t, err, "LIKE")
	}
	_, err := ParseJSON([]byte(`{"op":"like","args":[1,"x"]}`))
	assertParseErrorContains(t, err, "expected character expression")
}

func TestParseJSONIsNullOperands(t *testing.T) {
	cases := []string{
		`{"op":"isNull","args":[{"property":"deleted_at"}]}`,
		`{"op":"isNull","args":[true]}`,
		`{"op":"isNull","args":[{"op":"=","args":[{"property":"a"},1]}]}`,
		`{"op":"isNull","args":[{"op":"+","args":[{"property":"height"},1]}]}`,
	}
	for _, input := range cases {
		expr, err := ParseJSON([]byte(input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
		if _, ok := expr.(*IsNullExpression); !ok {
			t.Fatalf("%s parsed as %T, want IsNullExpression", input, expr)
		}
	}

	_, err := ParseJSON([]byte(`{"op":"isNull","args":[[]]}`))
	assertParseErrorContains(t, err, "expected IS NULL operand")
}

func TestParseJSONBooleanComparisons(t *testing.T) {
	cases := []string{
		`{"op":"=","args":[{"property":"active"},true]}`,
		`{"op":"<>","args":[false,{"property":"archived"}]}`,
		`{"op":"=","args":[true,false]}`,
	}
	for _, input := range cases {
		expr, err := ParseJSON([]byte(input))
		if err != nil {
			t.Fatalf("ParseJSON(%s): %v", input, err)
		}
		if _, ok := expr.(*ComparisonExpression); !ok {
			t.Fatalf("%s parsed as %T, want ComparisonExpression", input, expr)
		}
	}
}

func TestParseJSONFunctionsAndArrays(t *testing.T) {
	expr, err := ParseJSON([]byte(`{"op":"my_func","args":[{"property":"name"},[1,"x"],true]}`), WithAllowedFunctions(FunctionDefinition{
		Name:    "my_func",
		Args:    []FunctionArgument{{Types: []FunctionType{FunctionTypeAny}, Variadic: true}},
		Returns: []FunctionType{FunctionTypeBoolean},
	}))
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	fn, ok := expr.(*FunctionCall)
	if !ok {
		t.Fatalf("expr = %T, want FunctionCall", expr)
	}
	if fn.Name != "my_func" || len(fn.Args) != 3 {
		t.Fatalf("function = %#v", fn)
	}
	if _, ok := fn.Args[1].(*ArrayLiteral); !ok {
		t.Fatalf("arg[1] = %T, want ArrayLiteral", fn.Args[1])
	}

	expr, err = ParseJSON([]byte(`{"op":"=","args":[{"op":"casei","args":[{"property":"name"}]},{"op":"casei","args":["foo"]}]}`))
	if err != nil {
		t.Fatalf("ParseJSON casei comparison: %v", err)
	}
	if _, ok := expr.(*ComparisonExpression); !ok {
		t.Fatalf("expr = %T, want ComparisonExpression", expr)
	}
}

func TestParseJSONFunctionRegistry(t *testing.T) {
	_, err := ParseJSON([]byte(`{"op":"my_func","args":[]}`))
	assertParseErrorContains(t, err, `function "my_func" is not allowed`)

	_, err = ParseJSON([]byte(`{"op":"=","args":[{"property":"x"},{"op":"casei","args":[1]}]}`))
	assertParseErrorContains(t, err, `expected character expression`)

	expr, err := ParseJSON([]byte(`{"op":"is_named","args":[{"property":"name"}]}`), WithAllowedFunctions(FunctionDefinition{
		Name:    "is_named",
		Args:    []FunctionArgument{{Name: "value", Types: []FunctionType{FunctionTypeString}}},
		Returns: []FunctionType{FunctionTypeBoolean},
	}))
	if err != nil {
		t.Fatalf("ParseJSON registered function: %v", err)
	}
	fn, ok := expr.(*FunctionCall)
	if !ok || fn.Name != "is_named" || !functionCallReturns(fn, FunctionTypeBoolean) {
		t.Fatalf("expr = %#v, want registered boolean function", expr)
	}

	_, err = ParseJSON([]byte(`{"op":"str_fn","args":[]}`), WithAllowedFunctions(FunctionDefinition{
		Name:    "str_fn",
		Returns: []FunctionType{FunctionTypeString},
	}))
	assertParseErrorContains(t, err, `does not return boolean`)
}

func TestParseJSONArithmetic(t *testing.T) {
	expr, err := ParseJSON([]byte(`{"op":">=","args":[{"op":"*","args":[{"op":"+","args":[{"property":"a"},1]},2]},{"op":"div","args":[{"property":"b"},3]}]}`))
	if err != nil {
		t.Fatalf("ParseJSON arithmetic: %v", err)
	}
	cmp, ok := expr.(*ComparisonExpression)
	if !ok {
		t.Fatalf("expr = %T, want ComparisonExpression", expr)
	}
	left, ok := cmp.Left.(*ArithmeticExpression)
	if !ok || left.Op != ArithmeticMul {
		t.Fatalf("left = %#v, want multiplication", cmp.Left)
	}
	right, ok := cmp.Right.(*ArithmeticExpression)
	if !ok || right.Op != ArithmeticIntDiv {
		t.Fatalf("right = %#v, want integer division", cmp.Right)
	}

	_, err = ParseJSON([]byte(`{"op":"+","args":[1]}`))
	assertParseErrorContains(t, err, `unsupported reserved operation "+"`)

	_, err = ParseJSON([]byte(`{"op":"=","args":[{"property":"x"},{"op":"+","args":["a",1]}]}`))
	assertParseErrorContains(t, err, "expected numeric expression")
}

func TestParseJSONDepthLimit(t *testing.T) {
	_, err := ParseJSON([]byte(`{"op":"not","args":[{"op":"not","args":[true]}]}`), WithMaxDepth(1))
	assertParseErrorContains(t, err, "maximum parse depth exceeded")

	_, err = ParseJSON([]byte(`{"op":"like","args":[{"property":"name"},{"op":"casei","args":[{"op":"casei","args":["x"]}]}]}`), WithMaxDepth(1))
	assertParseErrorContains(t, err, "maximum parse depth exceeded")
}

func TestParseJSONRejectsInvalidExpressionLiterals(t *testing.T) {
	_, err := ParseJSON([]byte(`"not a filter"`))
	assertParseErrorContains(t, err, "expected CQL2 expression object or boolean")

	_, err = ParseJSON([]byte(`1`))
	assertParseErrorContains(t, err, "expected CQL2 expression object or boolean")
}

func TestParseJSONRejectsReservedScalarFunctions(t *testing.T) {
	_, err := ParseJSON([]byte(`{"op":"=","args":[{"property":"x"},{"op":"=","args":[1,2]}]}`))
	assertParseErrorContains(t, err, `reserved operation "="`)
}

func TestParseJSONArgumentCounts(t *testing.T) {
	_, err := ParseJSON([]byte(`{"op":"=","args":[{"property":"name"}]}`))
	assertParseErrorContains(t, err, "expected exactly 2 arguments")

	_, err = ParseJSON([]byte(`{"args":[]}`))
	assertParseErrorContains(t, err, "$.op")

	_, err = ParseJSON([]byte(`{"op":"my_func"}`))
	assertParseErrorContains(t, err, "$.args")

	_, err = ParseJSON([]byte(`{"op":"my_func","args":null}`))
	assertParseErrorContains(t, err, "expected array")
}

func TestParseJSONRejectsInvalidOperandShapes(t *testing.T) {
	cases := map[string]string{
		`{"op":"=","args":[{"property":"x"},null]}`:                      "NULL is only allowed",
		`{"op":"like","args":[{"property":"name"},1]}`:                   "LIKE pattern must be a string",
		`{"op":"like","args":[1,"x"]}`:                                   "expected character expression",
		`{"op":"between","args":[{"property":"height"},"a",2]}`:          "expected numeric expression",
		`{"op":"casei","args":["a","b"]}`:                                "is not a boolean expression",
		`{"op":"=","args":[{"property":"x"},{"op":"casei","args":[1]}]}`: "expected character expression",
	}
	for input, want := range cases {
		_, err := ParseJSON([]byte(input))
		assertParseErrorContains(t, err, want)
	}
}
