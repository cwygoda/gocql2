package parser

import (
	"reflect"
	"testing"

	"github.com/cwygoda/cql2/api"
)

func TestTemporalPredicatesTextAndJSON(t *testing.T) {
	cases := []struct {
		name string
		text string
		json string
		op   api.TemporalPredicateOp
	}{
		{name: "after timestamp", text: `T_AFTER(event_time,TIMESTAMP('2022-04-24T07:59:57Z'))`, json: `{"op":"t_after","args":[{"property":"event_time"},{"timestamp":"2022-04-24T07:59:57Z"}]}`, op: api.TemporalOpAfter},
		{name: "before date", text: `T_BEFORE(event_date,DATE('2022-04-24'))`, json: `{"op":"t_before","args":[{"property":"event_date"},{"date":"2022-04-24"}]}`, op: api.TemporalOpBefore},
		{name: "disjoint interval", text: `T_DISJOINT(event_time,INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))`, json: `{"op":"t_disjoint","args":[{"property":"event_time"},{"interval":["2021-01-01T00:00:00Z","2021-12-31T23:59:59Z"]}]}`, op: api.TemporalOpDisjoint},
		{name: "equals interval", text: `T_EQUALS(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`, json: `{"op":"t_equals","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`, op: api.TemporalOpEquals},
		{name: "intersects timestamp", text: `T_INTERSECTS(event_time,TIMESTAMP('2022-04-24T07:59:57Z'))`, json: `{"op":"t_intersects","args":[{"property":"event_time"},{"timestamp":"2022-04-24T07:59:57Z"}]}`, op: api.TemporalOpIntersects},
		{name: "contains", text: `T_CONTAINS(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`, json: `{"op":"t_contains","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`, op: api.TemporalOpContains},
		{name: "during", text: `T_DURING(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`, json: `{"op":"t_during","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`, op: api.TemporalOpDuring},
		{name: "finishedBy", text: `T_FINISHEDBY(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`, json: `{"op":"t_finishedBy","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`, op: api.TemporalOpFinishedBy},
		{name: "finishes", text: `T_FINISHES(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`, json: `{"op":"t_finishes","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`, op: api.TemporalOpFinishes},
		{name: "meets", text: `T_MEETS(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`, json: `{"op":"t_meets","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`, op: api.TemporalOpMeets},
		{name: "metBy", text: `T_METBY(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`, json: `{"op":"t_metBy","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`, op: api.TemporalOpMetBy},
		{name: "overlappedBy", text: `T_OVERLAPPEDBY(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`, json: `{"op":"t_overlappedBy","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`, op: api.TemporalOpOverlappedBy},
		{name: "overlaps", text: `T_OVERLAPS(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`, json: `{"op":"t_overlaps","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`, op: api.TemporalOpOverlaps},
		{name: "startedBy", text: `T_STARTEDBY(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`, json: `{"op":"t_startedBy","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`, op: api.TemporalOpStartedBy},
		{name: "starts", text: `T_STARTS(INTERVAL(start_time,end_time),INTERVAL('2021-01-01','2021-12-31'))`, json: `{"op":"t_starts","args":[{"interval":[{"property":"start_time"},{"property":"end_time"}]},{"interval":["2021-01-01","2021-12-31"]}]}`, op: api.TemporalOpStarts},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			textExpr, err := NewParser().WithConformance(api.ConformanceTemporalFunctions).ParseText(tt.text)
			if err != nil {
				t.Fatalf("ParseText: %v", err)
			}
			textTemporal, ok := textExpr.(*api.TemporalPredicateExpression)
			if !ok {
				t.Fatalf("ParseText type = %T, want *api.TemporalPredicateExpression", textExpr)
			}
			if textTemporal.Op != tt.op {
				t.Fatalf("text op = %q, want %q", textTemporal.Op, tt.op)
			}

			jsonExpr, err := NewParser().WithConformance(api.ConformanceTemporalFunctions).ParseJSON([]byte(tt.json))
			if err != nil {
				t.Fatalf("ParseJSON: %v", err)
			}
			jsonTemporal, ok := jsonExpr.(*api.TemporalPredicateExpression)
			if !ok {
				t.Fatalf("ParseJSON type = %T, want *api.TemporalPredicateExpression", jsonExpr)
			}
			if jsonTemporal.Op != tt.op {
				t.Fatalf("json op = %q, want %q", jsonTemporal.Op, tt.op)
			}
			if !reflect.DeepEqual(semantic(textExpr), semantic(jsonExpr)) {
				t.Fatalf("semantic mismatch\ntext: %#v\njson: %#v", semantic(textExpr), semantic(jsonExpr))
			}
		})
	}
}

func TestParseJSONTemporalOpNamesAreCaseSensitive(t *testing.T) {
	cases := []string{
		`{"op":"T_AFTER","args":[{"property":"event_time"},{"timestamp":"2022-04-24T07:59:57Z"}]}`,
		`{"op":"t_finishedby","args":[{"interval":["2021-01-01","2021-12-31"]},{"interval":["2021-01-01","2021-12-31"]}]}`,
		`{"op":"t_metby","args":[{"interval":["2021-01-01","2021-12-31"]},{"interval":["2021-01-01","2021-12-31"]}]}`,
		`{"op":"t_overlappedby","args":[{"interval":["2021-01-01","2021-12-31"]},{"interval":["2021-01-01","2021-12-31"]}]}`,
		`{"op":"t_startedby","args":[{"interval":["2021-01-01","2021-12-31"]},{"interval":["2021-01-01","2021-12-31"]}]}`,
	}
	parser := NewParser().WithConformance(api.ConformanceTemporalFunctions)
	for _, input := range cases {
		_, err := parser.ParseJSON([]byte(input))
		assertParseErrorContains(t, err, "unsupported reserved operation")
	}
}

func TestTemporalLiteralsInScalarAndValueContexts(t *testing.T) {
	okCases := []struct {
		name string
		lang api.Language
		in   string
	}{
		{name: "text date comparison", lang: api.LanguageText, in: `event_date = DATE('2022-04-24')`},
		{name: "text timestamp comparison", lang: api.LanguageText, in: `event_time >= TIMESTAMP('2022-04-24T07:59:57Z')`},
		{name: "text interval is null", lang: api.LanguageText, in: `INTERVAL('2022-01-01','2022-01-31') IS NOT NULL`},
		{name: "text interval function arg", lang: api.LanguageText, in: `has_interval(INTERVAL('2022-01-01','..'))`},
		{name: "text interval array element", lang: api.LanguageText, in: `A_CONTAINS(values,(INTERVAL('2022-01-01','..'))) `},
		{name: "json date comparison", lang: api.LanguageJSON, in: `{"op":"=","args":[{"property":"event_date"},{"date":"2022-04-24"}]}`},
		{name: "json timestamp comparison", lang: api.LanguageJSON, in: `{"op":">=","args":[{"property":"event_time"},{"timestamp":"2022-04-24T07:59:57Z"}]}`},
		{name: "json interval is null", lang: api.LanguageJSON, in: `{"op":"isNull","args":[{"interval":["2022-01-01","2022-01-31"]}]}`},
	}
	parser := NewParser().WithConformance(api.ConformanceArrayFunctions).WithAllowedFunctions(api.FunctionDefinition{
		Name:    "has_interval",
		Args:    []api.FunctionArgument{{Name: "value", Types: []api.FunctionType{api.FunctionTypeInterval}}},
		Returns: []api.FunctionType{api.FunctionTypeBoolean},
	})

	for _, tt := range okCases {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.lang == api.LanguageText {
				_, err = parser.ParseText(tt.in)
			} else {
				_, err = parser.ParseJSON([]byte(tt.in))
			}
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
		})
	}
}

func TestParseJSONTemporalInstanceRequiresExactlyOneKind(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{name: "date and timestamp", in: `{"op":"=","args":[{"property":"event_time"},{"date":"2024-01-01","timestamp":"2024-01-01T00:00:00Z"}]}`},
		{name: "date and interval", in: `{"op":"t_after","args":[{"date":"2024-01-01","interval":["2024-01-01","2024-01-02"]},{"property":"event_time"}]}`},
		{name: "timestamp and interval is null", in: `{"op":"isNull","args":[{"timestamp":"2024-01-01T00:00:00Z","interval":["2024-01-01T00:00:00Z","2024-01-02T00:00:00Z"]}]}`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewParser().WithConformance(api.ConformanceTemporalFunctions).ParseJSON([]byte(tt.in))
			assertParseErrorContains(t, err, "exactly one of date, timestamp, or interval")
		})
	}
}

func TestTimestampRequiresUTC(t *testing.T) {
	_, err := ParseText(`event_time = TIMESTAMP('2022-04-24T09:59:57+02:00')`)
	assertParseErrorContains(t, err, "ending in Z")

	_, err = ParseJSON([]byte(`{"op":"=","args":[{"property":"event_time"},{"timestamp":"2022-04-24T09:59:57+02:00"}]}`))
	assertParseErrorContains(t, err, "ending in Z")

	if _, err := ParseJSON([]byte(`{"op":"=","args":[{"property":"event_time"},{"timestamp":"2022-04-24T07:59:57Z"}]}`)); err != nil {
		t.Fatalf("UTC Z timestamp should parse: %v", err)
	}
}

func TestTemporalOperandValidation(t *testing.T) {
	typedProperties := []api.PropertyDefinition{
		{Name: "name", Type: api.PropertyTypeString},
		{Name: "event_date", Type: api.PropertyTypeDate},
		{Name: "event_time", Type: api.PropertyTypeTimestamp},
		{Name: "event_interval", Type: api.PropertyTypeInterval},
		{Name: "start_date", Type: api.PropertyTypeDate},
		{Name: "end_time", Type: api.PropertyTypeTimestamp},
	}
	errorCases := []struct {
		name    string
		lang    api.Language
		in      string
		message string
	}{
		{name: "text non temporal property", lang: api.LanguageText, in: `T_AFTER(name,DATE('2022-01-01'))`, message: `cannot be used as a temporal operand`},
		{name: "json non temporal property", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"property":"name"},{"date":"2022-01-01"}]}`, message: `cannot be used as a temporal operand`},
		{name: "text non temporal literal right", lang: api.LanguageText, in: `T_AFTER(event_date,'x')`, message: `expected temporal operand`},
		{name: "json non temporal literal right", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"property":"event_date"},"x"]}`, message: `expected temporal operand`},
		{name: "text interval-only instant", lang: api.LanguageText, in: `T_DURING(event_date,DATE('2022-01-01'))`, message: `operands must be intervals`},
		{name: "json interval-only instant", lang: api.LanguageJSON, in: `{"op":"t_during","args":[{"property":"event_date"},{"date":"2022-01-01"}]}`, message: `operands must be intervals`},
		{name: "text interval-only right instant", lang: api.LanguageText, in: `T_DURING(event_interval,event_date)`, message: `operands must be intervals`},
		{name: "json interval-only right instant", lang: api.LanguageJSON, in: `{"op":"t_during","args":[{"property":"event_interval"},{"property":"event_date"}]}`, message: `operands must be intervals`},
		{name: "text bad interval endpoint", lang: api.LanguageText, in: `T_AFTER(INTERVAL(name,event_date),event_date)`, message: `cannot be used as an interval endpoint`},
		{name: "text bad interval end endpoint", lang: api.LanguageText, in: `T_AFTER(INTERVAL(event_date,name),event_date)`, message: `cannot be used as an interval endpoint`},
		{name: "json bad interval endpoint", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"interval":[{"property":"name"},{"property":"event_date"}]},{"property":"event_date"}]}`, message: `cannot be used as an interval endpoint`},
		{name: "text mismatched interval endpoint", lang: api.LanguageText, in: `T_AFTER(INTERVAL(start_date,end_time),event_date)`, message: `matching temporal granularity`},
		{name: "json mismatched interval endpoint", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"interval":["2022-01-01","2022-01-02T00:00:00Z"]},{"property":"event_date"}]}`, message: `matching temporal granularity`},
		{name: "text interval start after end", lang: api.LanguageText, in: `T_AFTER(INTERVAL('2022-02-01','2022-01-01'),event_interval)`, message: `start must not be after`},
		{name: "json interval start after end", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"interval":["2022-02-01","2022-01-01"]},{"property":"event_interval"}]}`, message: `start must not be after`},
	}
	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewParser().WithConformance(api.ConformanceTemporalFunctions).WithAllowedProperties(typedProperties...).Parse([]byte(tt.in), tt.lang)
			assertParseErrorContains(t, err, tt.message)
		})
	}
}

func TestTemporalLiteralValidation(t *testing.T) {
	errorCases := []struct {
		name    string
		lang    api.Language
		in      string
		message string
	}{
		{name: "text invalid date", lang: api.LanguageText, in: `event_date = DATE('2022-02-30')`, message: `invalid date`},
		{name: "json invalid date", lang: api.LanguageJSON, in: `{"op":"=","args":[{"property":"event_date"},{"date":"2022-02-30"}]}`, message: `invalid date`},
		{name: "text invalid timestamp", lang: api.LanguageText, in: `event_time = TIMESTAMP('2022-04-24T25:59:57Z')`, message: `invalid timestamp`},
		{name: "json invalid timestamp", lang: api.LanguageJSON, in: `{"op":"=","args":[{"property":"event_time"},{"timestamp":"2022-04-24T25:59:57Z"}]}`, message: `invalid timestamp`},
		{name: "text malformed timestamp", lang: api.LanguageText, in: `event_time = TIMESTAMP('2022-04-24 07:59:57')`, message: `timestamp must be`},
		{name: "json malformed interval", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"interval":["bad","2022-01-01"]},{"property":"event_date"}]}`, message: `date must match`},
		{name: "json object instant interval endpoint", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"interval":[{"date":"2022-01-01"},{"date":"2022-01-02"}]},{"property":"event_date"}]}`, message: `expected interval endpoint`},
	}
	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewParser().WithConformance(api.ConformanceTemporalFunctions).Parse([]byte(tt.in), tt.lang)
			assertParseErrorContains(t, err, tt.message)
		})
	}
}

func TestTemporalSyntaxAndJSONValidationErrors(t *testing.T) {
	errorCases := []struct {
		name    string
		lang    api.Language
		in      string
		message string
	}{
		{name: "text predicate missing left", lang: api.LanguageText, in: `T_AFTER(,DATE('2022-01-02'))`, message: `expected scalar expression`},
		{name: "text predicate missing comma", lang: api.LanguageText, in: `T_AFTER(DATE('2022-01-01') DATE('2022-01-02'))`, message: `expected comma`},
		{name: "text predicate missing right", lang: api.LanguageText, in: `T_AFTER(DATE('2022-01-01'),)`, message: `expected scalar expression`},
		{name: "text predicate missing open", lang: api.LanguageText, in: `T_AFTER DATE('2022-01-01'),DATE('2022-01-02')`, message: `expected opening parenthesis`},
		{name: "text predicate missing close", lang: api.LanguageText, in: `T_AFTER(DATE('2022-01-01'),DATE('2022-01-02')`, message: `expected closing parenthesis`},
		{name: "text date missing open", lang: api.LanguageText, in: `T_AFTER(DATE '2022-01-01',DATE('2022-01-02'))`, message: `expected opening parenthesis`},
		{name: "text date missing string", lang: api.LanguageText, in: `T_AFTER(DATE(),DATE('2022-01-02'))`, message: `expected temporal literal string`},
		{name: "text date missing close", lang: api.LanguageText, in: `T_AFTER(DATE('2022-01-01',DATE('2022-01-02'))`, message: `expected closing parenthesis`},
		{name: "text interval missing open", lang: api.LanguageText, in: `INTERVAL '2022-01-01','2022-01-02') IS NULL`, message: `expected opening parenthesis`},
		{name: "text interval missing comma", lang: api.LanguageText, in: `T_AFTER(INTERVAL('2022-01-01' '2022-01-02'),DATE('2022-01-03'))`, message: `expected comma`},
		{name: "text interval missing close", lang: api.LanguageText, in: `T_AFTER(INTERVAL('2022-01-01','2022-01-02',DATE('2022-01-03'))`, message: `expected closing parenthesis`},
		{name: "text interval bad start endpoint token", lang: api.LanguageText, in: `T_AFTER(INTERVAL(1,'2022-01-02'),DATE('2022-01-03'))`, message: `expected interval endpoint`},
		{name: "text interval bad end endpoint token", lang: api.LanguageText, in: `T_AFTER(INTERVAL('2022-01-01',1),DATE('2022-01-03'))`, message: `expected interval endpoint`},
		{name: "text interval malformed timestamp endpoint", lang: api.LanguageText, in: `T_AFTER(INTERVAL('2022-01-01T00:00:00','2022-01-02T00:00:00Z'),DATE('2022-01-03'))`, message: `timestamp must be`},
		{name: "json temporal predicate arity", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"date":"2022-01-01"}]}`, message: `expected exactly 2 arguments`},
		{name: "json date wrong type", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"date":1},{"date":"2022-01-02"}]}`, message: `expected date string`},
		{name: "json right date wrong type", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"date":"2022-01-01"},{"date":1}]}`, message: `expected date string`},
		{name: "json timestamp wrong type", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"timestamp":1},{"date":"2022-01-02"}]}`, message: `expected timestamp string`},
		{name: "json interval not array", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"interval":"2022-01-01/2022-01-02"},{"date":"2022-01-03"}]}`, message: `expected array`},
		{name: "json interval arity", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"interval":["2022-01-01"]},{"date":"2022-01-03"}]}`, message: `expected exactly 2 interval endpoints`},
		{name: "json interval endpoint wrong type", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"interval":[1,"2022-01-02"]},{"date":"2022-01-03"}]}`, message: `expected interval endpoint`},
		{name: "json interval endpoint bad scalar", lang: api.LanguageJSON, in: `{"op":"t_after","args":[{"interval":[{"foo":"bar"},"2022-01-02"]},{"date":"2022-01-03"}]}`, message: `expected interval endpoint`},
	}
	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewParser().WithConformance(api.ConformanceTemporalFunctions).Parse([]byte(tt.in), tt.lang)
			assertParseErrorContains(t, err, tt.message)
		})
	}
}

func TestTemporalUnboundedIntervalsAndASTSpans(t *testing.T) {
	textExpr, err := NewParser().WithConformance(api.ConformanceTemporalFunctions).ParseText(`T_AFTER(INTERVAL('..','2022-01-02'),INTERVAL('2022-01-03','..'))`)
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	pred, ok := textExpr.(*api.TemporalPredicateExpression)
	if !ok {
		t.Fatalf("ParseText type = %T, want *api.TemporalPredicateExpression", textExpr)
	}
	if pred.Span().End.ByteOffset <= pred.Span().Start.ByteOffset {
		t.Fatalf("predicate span = %#v, want non-empty text span", pred.Span())
	}
	left, ok := pred.Left.(*api.TemporalInterval)
	if !ok {
		t.Fatalf("left type = %T, want *api.TemporalInterval", pred.Left)
	}
	if left.Span().End.ByteOffset <= left.Span().Start.ByteOffset {
		t.Fatalf("interval span = %#v, want non-empty text span", left.Span())
	}
	if _, ok := left.Start.(*api.TemporalUnbounded); !ok {
		t.Fatalf("left start type = %T, want *api.TemporalUnbounded", left.Start)
	}
	if left.Start.Span().End.ByteOffset <= left.Start.Span().Start.ByteOffset {
		t.Fatalf("unbounded span = %#v, want non-empty text span", left.Start.Span())
	}

	jsonExpr, err := NewParser().WithConformance(api.ConformanceTemporalFunctions).ParseJSON([]byte(`{"op":"t_after","args":[{"interval":["..","2022-01-02"]},{"interval":["2022-01-03",".."]}]}`))
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if !reflect.DeepEqual(semantic(textExpr), semantic(jsonExpr)) {
		t.Fatalf("semantic mismatch\ntext: %#v\njson: %#v", semantic(textExpr), semantic(jsonExpr))
	}
}

func TestTemporalFunctionTypes(t *testing.T) {
	defs := []api.FunctionDefinition{
		{Name: "any_fn", Returns: []api.FunctionType{api.FunctionTypeAny}},
		{Name: "instant_fn", Returns: []api.FunctionType{api.FunctionTypeTimestamp}},
		{Name: "date_fn", Returns: []api.FunctionType{api.FunctionTypeDate}},
		{Name: "interval_fn", Returns: []api.FunctionType{api.FunctionTypeInterval}},
		{Name: "legacy_datetime_fn", Returns: []api.FunctionType{api.FunctionTypeDateTime}},
		{Name: "string_fn", Returns: []api.FunctionType{api.FunctionTypeString}},
	}
	okCases := []string{
		`T_AFTER(instant_fn(),TIMESTAMP('2022-01-01T00:00:00Z'))`,
		`T_AFTER(date_fn(),DATE('2022-01-01'))`,
		`T_DURING(interval_fn(),INTERVAL('2022-01-01','2022-01-31'))`,
		`T_DURING(any_fn(),interval_fn())`,
		`T_AFTER(INTERVAL(date_fn(),'2022-01-02'),interval_fn())`,
		`T_AFTER(INTERVAL(any_fn(),'2022-01-02'),interval_fn())`,
		`T_AFTER(INTERVAL(legacy_datetime_fn(),instant_fn()),interval_fn())`,
	}
	for _, input := range okCases {
		if _, err := NewParser().WithConformance(api.ConformanceTemporalFunctions).WithAllowedFunctions(defs...).ParseText(input); err != nil {
			t.Fatalf("ParseText(%q): %v", input, err)
		}
	}
	jsonFunctionEndpoint := `{"op":"t_after","args":[{"interval":[{"op":"legacy_datetime_fn","args":[]},{"op":"instant_fn","args":[]}]},{"timestamp":"2022-01-02T00:00:00Z"}]}`
	if _, err := NewParser().WithConformance(api.ConformanceTemporalFunctions).WithAllowedFunctions(defs...).ParseJSON([]byte(jsonFunctionEndpoint)); err != nil {
		t.Fatalf("ParseJSON interval function endpoints: %v", err)
	}
	_, err := NewParser().WithConformance(api.ConformanceTemporalFunctions).WithAllowedFunctions(defs...).ParseText(`T_AFTER(string_fn(),DATE('2022-01-01'))`)
	assertParseErrorContains(t, err, `does not return temporal`)
	_, err = NewParser().WithConformance(api.ConformanceTemporalFunctions).WithAllowedFunctions(defs...).ParseText(`T_AFTER(INTERVAL(interval_fn(),instant_fn()),DATE('2022-01-01'))`)
	assertParseErrorContains(t, err, `returns interval and cannot be used as an interval endpoint`)
	_, err = NewParser().WithConformance(api.ConformanceTemporalFunctions).WithAllowedFunctions(defs...).ParseText(`T_AFTER(INTERVAL(string_fn(),'2022-01-01'),DATE('2022-01-02'))`)
	assertParseErrorContains(t, err, `does not return instant`)

	acceptInterval := api.FunctionDefinition{
		Name:    "accept_interval",
		Args:    []api.FunctionArgument{{Name: "value", Types: []api.FunctionType{api.FunctionTypeInterval}}},
		Returns: []api.FunctionType{api.FunctionTypeBoolean},
	}
	_, err = NewParser().WithAllowedFunctions(append(defs, acceptInterval)...).ParseText(`accept_interval(legacy_datetime_fn())`)
	assertParseErrorContains(t, err, `expected interval`)
}
