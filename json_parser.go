package gocql2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

var comparisonOps = map[string]ComparisonOp{
	"=":  OpEqual,
	"<>": OpNotEqual,
	"<":  OpLessThan,
	">":  OpGreaterThan,
	"<=": OpLessThanOrEqual,
	">=": OpGreaterThanOrEqual,
}

var reservedJSONOps = map[string]struct{}{
	"and": {}, "or": {}, "not": {}, "=": {}, "<>": {}, "<": {}, ">": {}, "<=": {}, ">=": {},
	"like": {}, "between": {}, "in": {}, "isNull": {}, "casei": {}, "accenti": {},
	"+": {}, "-": {}, "*": {}, "/": {}, "^": {}, "%": {}, "div": {},
	"s_contains": {}, "s_crosses": {}, "s_disjoint": {}, "s_equals": {}, "s_intersects": {}, "s_overlaps": {}, "s_touches": {}, "s_within": {},
	"t_after": {}, "t_before": {}, "t_contains": {}, "t_disjoint": {}, "t_during": {}, "t_equals": {}, "t_finishedBy": {}, "t_finishes": {}, "t_intersects": {}, "t_meets": {}, "t_metBy": {}, "t_overlappedBy": {}, "t_overlaps": {}, "t_startedBy": {}, "t_starts": {},
	"a_containedBy": {}, "a_contains": {}, "a_equals": {}, "a_overlaps": {},
}

func parseJSON(input []byte, cfg ParseConfig) (Expression, error) {
	cfg = applyParseConfigDefaults(cfg)
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.UseNumber()

	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return nil, jsonSyntaxError(input, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, jsonSyntaxError(input, err)
	}

	return parseJSONExpression(raw, JSONPathRoot(), 0, cfg)
}

type rawObject map[string]json.RawMessage

type rawOpObject struct {
	Op   string
	Args []json.RawMessage
}

func parseJSONOpObject(raw json.RawMessage, path JSONPath) (rawOpObject, error) {
	var obj rawObject
	if err := unmarshalAt(raw, path, &obj); err != nil {
		return rawOpObject{}, err
	}

	opRaw, hasOp := obj["op"]
	if !hasOp {
		return rawOpObject{}, nil
	}

	var op string
	if err := unmarshalAt(opRaw, path.Key("op"), &op); err != nil {
		return rawOpObject{}, jsonPathError(path.Key("op"), "expected string operation")
	}

	argsRaw, hasArgs := obj["args"]
	if !hasArgs {
		return rawOpObject{}, jsonPathError(path.Key("args"), "missing arguments")
	}
	if trimmed := bytes.TrimSpace(argsRaw); len(trimmed) == 0 || trimmed[0] != '[' {
		return rawOpObject{}, jsonPathError(path.Key("args"), "expected array")
	}

	var args []json.RawMessage
	if err := unmarshalAt(argsRaw, path.Key("args"), &args); err != nil {
		return rawOpObject{}, jsonPathError(path.Key("args"), "expected array")
	}
	return rawOpObject{Op: op, Args: args}, nil
}

func parseJSONExpression(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (Expression, error) {
	if depth > cfg.MaxDepth {
		return nil, jsonPathError(path, "maximum parse depth exceeded")
	}

	if lit, ok, err := parseJSONLiteral(raw, path); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if lit.Kind == LiteralBool {
			return lit, nil
		}
		return nil, jsonPathError(path, "expected CQL2 expression object or boolean")
	}

	op, err := parseJSONOpObject(raw, path)
	if err != nil {
		return nil, err
	}
	if op.Op == "" {
		return nil, jsonPathError(path.Key("op"), "missing operation")
	}

	src := jsonSpan(path)
	if spatialOp, ok := isSpatialPredicateOp(op.Op); ok {
		return parseJSONSpatialPredicate(spatialOp, op.Args, path, depth, cfg)
	}
	if temporalOp, ok := isTemporalPredicateOp(op.Op); ok {
		return parseJSONTemporalPredicate(temporalOp, op.Args, path, depth, cfg)
	}
	switch op.Op {
	case "and", "or":
		if len(op.Args) < 2 {
			return nil, jsonPathError(path.Key("args"), "expected at least two arguments")
		}
		args := make([]Expression, 0, len(op.Args))
		for i, arg := range op.Args {
			expr, err := parseJSONExpression(arg, path.Key("args").Index(i), depth+1, cfg)
			if err != nil {
				return nil, err
			}
			args = append(args, expr)
		}
		return &LogicalExpression{Op: LogicalOp(op.Op), Args: args, Src: src}, nil
	case "not":
		if len(op.Args) != 1 {
			return nil, jsonPathError(path.Key("args"), "expected exactly one argument")
		}
		expr, err := parseJSONExpression(op.Args[0], path.Key("args").Index(0), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		return &LogicalExpression{Op: LogicalNot, Args: []Expression{expr}, Src: src}, nil
	case "=", "<>", "<", ">", "<=", ">=":
		args, err := parseJSONScalarArgs(op.Args, path.Key("args"), depth, cfg, 2, 2)
		if err != nil {
			return nil, err
		}
		cmpOp := comparisonOps[op.Op]
		if err := validateComparisonOperands(cmpOp, args[0], args[1], LanguageJSON); err != nil {
			return nil, err
		}
		return &ComparisonExpression{Op: cmpOp, Left: args[0], Right: args[1], Src: src}, nil
	case "like":
		if len(op.Args) != 2 {
			return nil, jsonPathError(path.Key("args"), "expected exactly 2 arguments")
		}
		expr, err := parseJSONCharacterExpression(op.Args[0], path.Key("args").Index(0), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		pattern, err := parseJSONPatternExpression(op.Args[1], path.Key("args").Index(1), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		return &LikeExpression{Expr: expr, Pattern: pattern, Src: src}, nil
	case "between":
		if len(op.Args) != 3 {
			return nil, jsonPathError(path.Key("args"), "expected exactly 3 arguments")
		}
		args := make([]ScalarExpression, 0, 3)
		for i, rawArg := range op.Args {
			arg, err := parseJSONNumericExpression(rawArg, path.Key("args").Index(i), depth+1, cfg)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
		return &BetweenExpression{Expr: args[0], Lower: args[1], Upper: args[2], Src: src}, nil
	case "in":
		if len(op.Args) != 2 {
			return nil, jsonPathError(path.Key("args"), "expected exactly two arguments")
		}
		expr, err := parseJSONScalar(op.Args[0], path.Key("args").Index(0), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		values, err := parseJSONScalarArray(op.Args[1], path.Key("args").Index(1), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		if err := validateInOperands(expr, values, LanguageJSON); err != nil {
			return nil, err
		}
		return &InExpression{Expr: expr, Values: values, Src: src}, nil
	case "isNull":
		if len(op.Args) != 1 {
			return nil, jsonPathError(path.Key("args"), "expected exactly 1 arguments")
		}
		operand, err := parseJSONIsNullOperand(op.Args[0], path.Key("args").Index(0), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		return &IsNullExpression{Expr: operand, Src: src}, nil
	case "a_contains", "a_containedby", "a_equals", "a_overlaps":
		if len(op.Args) != 2 {
			return nil, jsonPathError(path.Key("args"), "expected exactly 2 arguments")
		}
		left, err := parseJSONArrayOperand(op.Args[0], path.Key("args").Index(0), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		right, err := parseJSONArrayOperand(op.Args[1], path.Key("args").Index(1), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		if err := validateArrayPredicateOperands(left, right, LanguageJSON); err != nil {
			return nil, err
		}
		return &ArrayPredicateExpression{Op: arrayPredicateOps[op.Op], Left: left, Right: right, Src: src}, nil
	case "casei", "accenti":
		return nil, jsonPathError(path.Key("op"), fmt.Sprintf("%q is not a boolean expression", op.Op))
	default:
		if _, reserved := reservedJSONOps[op.Op]; reserved {
			return nil, jsonPathError(path.Key("op"), fmt.Sprintf("unsupported reserved operation %q", op.Op))
		}
		fn, err := parseJSONFunction(op.Op, op.Args, path, depth, cfg)
		if err != nil {
			return nil, err
		}
		if !functionCallReturns(fn, FunctionTypeBoolean) {
			return nil, jsonPathError(path.Key("op"), fmt.Sprintf("function %q does not return boolean", fn.Name))
		}
		return fn, nil
	}
}

func parseJSONTemporalPredicate(op TemporalPredicateOp, rawArgs []json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (*TemporalPredicateExpression, error) {
	if len(rawArgs) != 2 {
		return nil, jsonPathError(path.Key("args"), "expected exactly 2 arguments")
	}
	left, err := parseJSONTemporalOperand(rawArgs[0], path.Key("args").Index(0), depth+1, cfg)
	if err != nil {
		return nil, err
	}
	right, err := parseJSONTemporalOperand(rawArgs[1], path.Key("args").Index(1), depth+1, cfg)
	if err != nil {
		return nil, err
	}
	if err := validateTemporalPredicateOperands(op, left, right, LanguageJSON); err != nil {
		return nil, err
	}
	return &TemporalPredicateExpression{Op: op, Left: left, Right: right, Src: jsonSpan(path)}, nil
}

func parseJSONTemporalOperand(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (Node, error) {
	temporal, temporalErr := parseJSONTemporalInstance(raw, path, depth+1, cfg)
	if temporalErr == nil {
		return temporal, nil
	}
	if hasJSONTemporalInstanceKey(raw, path) {
		return nil, temporalErr
	}
	node, err := parseJSONScalar(raw, path, depth+1, cfg)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func hasJSONTemporalInstanceKey(raw json.RawMessage, path JSONPath) bool {
	var obj rawObject
	if err := unmarshalAt(raw, path, &obj); err != nil {
		return false
	}
	_, hasDate := obj["date"]
	_, hasTimestamp := obj["timestamp"]
	_, hasInterval := obj["interval"]
	return hasDate || hasTimestamp || hasInterval
}

func parseJSONTemporalInstance(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (Node, error) {
	if depth > cfg.MaxDepth {
		return nil, jsonPathError(path, "maximum parse depth exceeded")
	}
	var obj rawObject
	if err := unmarshalAt(raw, path, &obj); err != nil {
		return nil, err
	}
	if _, ok := obj["date"]; ok {
		return parseJSONTemporalInstant(raw, path, cfg)
	}
	if _, ok := obj["timestamp"]; ok {
		return parseJSONTemporalInstant(raw, path, cfg)
	}
	if _, ok := obj["interval"]; ok {
		return parseJSONTemporalInterval(raw, path, depth+1, cfg)
	}
	return nil, jsonPathError(path, "expected temporal instance")
}

func parseJSONTemporalInstant(raw json.RawMessage, path JSONPath, cfg ParseConfig) (*TemporalInstant, error) {
	var obj rawObject
	if err := unmarshalAt(raw, path, &obj); err != nil {
		return nil, err
	}
	if rawDate, ok := obj["date"]; ok {
		var value string
		if err := unmarshalAt(rawDate, path.Key("date"), &value); err != nil {
			return nil, jsonPathError(path.Key("date"), "expected date string")
		}
		if err := validateDateLiteral(value); err != nil {
			return nil, jsonPathError(path.Key("date"), err.Error())
		}
		return &TemporalInstant{Kind: TemporalInstantDate, Value: value, Src: jsonSpan(path)}, nil
	}
	if rawTimestamp, ok := obj["timestamp"]; ok {
		var value string
		if err := unmarshalAt(rawTimestamp, path.Key("timestamp"), &value); err != nil {
			return nil, jsonPathError(path.Key("timestamp"), "expected timestamp string")
		}
		if err := validateTimestampLiteral(value, cfg.StrictTimestampUTC); err != nil {
			return nil, jsonPathError(path.Key("timestamp"), err.Error())
		}
		return &TemporalInstant{Kind: TemporalInstantTimestamp, Value: value, Src: jsonSpan(path)}, nil
	}
	return nil, jsonPathError(path, "expected temporal instant")
}

func parseJSONTemporalInterval(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (*TemporalInterval, error) {
	if depth > cfg.MaxDepth {
		return nil, jsonPathError(path, "maximum parse depth exceeded")
	}
	var obj rawObject
	if err := unmarshalAt(raw, path, &obj); err != nil {
		return nil, err
	}
	rawInterval, ok := obj["interval"]
	if !ok {
		return nil, jsonPathError(path.Key("interval"), "missing interval")
	}
	var items []json.RawMessage
	if err := unmarshalAt(rawInterval, path.Key("interval"), &items); err != nil {
		return nil, jsonPathError(path.Key("interval"), "expected array")
	}
	if len(items) != 2 {
		return nil, jsonPathError(path.Key("interval"), "expected exactly 2 interval endpoints")
	}
	start, err := parseJSONTemporalIntervalEndpoint(items[0], path.Key("interval").Index(0), depth+1, cfg)
	if err != nil {
		return nil, err
	}
	end, err := parseJSONTemporalIntervalEndpoint(items[1], path.Key("interval").Index(1), depth+1, cfg)
	if err != nil {
		return nil, err
	}
	if err := validateTemporalIntervalOperands(start, end, LanguageJSON); err != nil {
		return nil, err
	}
	return &TemporalInterval{Start: start, End: end, Src: jsonSpan(path)}, nil
}

func parseJSONTemporalIntervalEndpoint(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (Node, error) {
	if depth > cfg.MaxDepth {
		return nil, jsonPathError(path, "maximum parse depth exceeded")
	}
	if lit, ok, err := parseJSONLiteral(raw, path); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if lit.Kind != LiteralString {
			return nil, jsonPathError(path, "expected interval endpoint")
		}
		value, ok := lit.Value.(string)
		if !ok {
			return nil, jsonPathError(path, "expected interval endpoint")
		}
		if value == ".." {
			return &TemporalUnbounded{Src: lit.Src}, nil
		}
		kind, err := temporalInstantKindFromString(value, cfg.StrictTimestampUTC)
		if err != nil {
			return nil, jsonPathError(path, err.Error())
		}
		return &TemporalInstant{Kind: kind, Value: value, Src: lit.Src}, nil
	}
	node, err := parseJSONScalar(raw, path, depth+1, cfg)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func parseJSONIsNullOperand(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (Node, error) {
	if expr, err := parseJSONExpression(raw, path, depth, cfg); err == nil {
		return expr, nil
	}
	if scalar, err := parseJSONScalar(raw, path, depth, cfg); err == nil {
		return scalar, nil
	}
	if temporal, err := parseJSONTemporalInstance(raw, path, depth, cfg); err == nil {
		return temporal, nil
	}
	return nil, jsonPathError(path, "expected IS NULL operand")
}

func parseJSONArrayOperand(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (Node, error) {
	if array, err := parseJSONArrayLiteral(raw, path, depth+1, cfg); err == nil {
		return array, nil
	}
	node, err := parseJSONScalar(raw, path, depth+1, cfg)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func parseJSONScalar(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (ScalarExpression, error) {
	if depth > cfg.MaxDepth {
		return nil, jsonPathError(path, "maximum parse depth exceeded")
	}
	if lit, ok, err := parseJSONLiteral(raw, path); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if lit.Kind == LiteralNull {
			return nil, jsonPathError(path, "NULL is only allowed in isNull predicates")
		}
		return lit, nil
	}

	var obj rawObject
	if err := unmarshalAt(raw, path, &obj); err != nil {
		return nil, err
	}
	if propRaw, ok := obj["property"]; ok {
		var name string
		if err := unmarshalAt(propRaw, path.Key("property"), &name); err != nil {
			return nil, jsonPathError(path.Key("property"), "expected string property name")
		}
		if name == "" {
			return nil, jsonPathError(path.Key("property"), "property name must not be empty")
		}
		return propertyRef(name, jsonSpan(path), cfg, LanguageJSON, Location{ByteOffset: -1, CharOffset: -1, JSONPath: path.Key("property")})
	}
	if _, ok := obj["date"]; ok {
		return parseJSONTemporalInstant(raw, path, cfg)
	}
	if _, ok := obj["timestamp"]; ok {
		return parseJSONTemporalInstant(raw, path, cfg)
	}

	op, err := parseJSONOpObject(raw, path)
	if err != nil {
		return nil, err
	}
	if op.Op == "" {
		return nil, jsonPathError(path, "expected scalar expression")
	}
	if op.Op == "casei" || op.Op == "accenti" {
		return parseJSONCharacterFunction(op.Op, op.Args, path, depth, cfg)
	}
	if isJSONArithmeticOp(op.Op) {
		return parseJSONArithmeticExpression(op.Op, op.Args, path, depth, cfg)
	}
	if _, reserved := reservedJSONOps[op.Op]; reserved {
		return nil, jsonPathError(path.Key("op"), fmt.Sprintf("reserved operation %q cannot be used as a scalar function", op.Op))
	}
	return parseJSONFunction(op.Op, op.Args, path, depth, cfg)
}

func parseJSONCharacterExpression(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (ScalarExpression, error) {
	if lit, ok, err := parseJSONLiteral(raw, path); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if lit.Kind != LiteralString {
			return nil, jsonPathError(path, "expected character expression")
		}
		return lit, nil
	}

	var obj rawObject
	if err := unmarshalAt(raw, path, &obj); err != nil {
		return nil, err
	}
	if _, ok := obj["property"]; ok {
		scalar, err := parseJSONScalar(raw, path, depth+1, cfg)
		if err != nil {
			return nil, err
		}
		if !isCharacterExpression(scalar) {
			return nil, jsonPathError(path, "expected character expression")
		}
		return scalar, nil
	}

	op, err := parseJSONOpObject(raw, path)
	if err != nil {
		return nil, err
	}
	if op.Op == "" {
		return nil, jsonPathError(path, "expected character expression")
	}
	if op.Op == "casei" || op.Op == "accenti" {
		return parseJSONCharacterFunction(op.Op, op.Args, path, depth+1, cfg)
	}
	if _, reserved := reservedJSONOps[op.Op]; reserved {
		return nil, jsonPathError(path.Key("op"), fmt.Sprintf("reserved operation %q cannot be used as a character function", op.Op))
	}
	fn, err := parseJSONFunction(op.Op, op.Args, path, depth+1, cfg)
	if err != nil {
		return nil, err
	}
	if !isCharacterExpression(fn) {
		return nil, jsonPathError(path, "expected character expression")
	}
	return fn, nil
}

func parseJSONPatternExpression(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (ScalarExpression, error) {
	if depth > cfg.MaxDepth {
		return nil, jsonPathError(path, "maximum parse depth exceeded")
	}
	if lit, ok, err := parseJSONLiteral(raw, path); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if lit.Kind != LiteralString {
			return nil, jsonPathError(path, "LIKE pattern must be a string or casei/accenti pattern")
		}
		return lit, nil
	}

	op, err := parseJSONOpObject(raw, path)
	if err != nil {
		return nil, err
	}
	if op.Op != "casei" && op.Op != "accenti" {
		return nil, jsonPathError(path, "LIKE pattern must be a string or casei/accenti pattern")
	}
	if len(op.Args) != 1 {
		return nil, jsonPathError(path.Key("args"), "expected exactly one argument")
	}
	pattern, err := parseJSONPatternExpression(op.Args[0], path.Key("args").Index(0), depth+1, cfg)
	if err != nil {
		return nil, err
	}
	def, err := validateFunctionCall(op.Op, []Node{pattern}, cfg, LanguageJSON, Location{ByteOffset: -1, CharOffset: -1, JSONPath: path.Key("op")})
	if err != nil {
		return nil, err
	}
	return &FunctionCall{Name: normalizeFunctionName(op.Op), Args: []Node{pattern}, ReturnTypes: cloneFunctionTypes(def.Returns), Src: jsonSpan(path)}, nil
}

func parseJSONNumericExpression(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (ScalarExpression, error) {
	if lit, ok, err := parseJSONLiteral(raw, path); ok || err != nil {
		if err != nil {
			return nil, err
		}
		if lit.Kind != LiteralNumber {
			return nil, jsonPathError(path, "expected numeric expression")
		}
		return lit, nil
	}

	var obj rawObject
	if err := unmarshalAt(raw, path, &obj); err != nil {
		return nil, err
	}
	if _, ok := obj["property"]; ok {
		scalar, err := parseJSONScalar(raw, path, depth+1, cfg)
		if err != nil {
			return nil, err
		}
		if !isNumericExpression(scalar) {
			return nil, jsonPathError(path, "expected numeric expression")
		}
		return scalar, nil
	}

	op, err := parseJSONOpObject(raw, path)
	if err != nil {
		return nil, err
	}
	if op.Op == "" {
		return nil, jsonPathError(path, "expected numeric expression")
	}
	if isJSONArithmeticOp(op.Op) {
		return parseJSONArithmeticExpression(op.Op, op.Args, path, depth+1, cfg)
	}
	if _, reserved := reservedJSONOps[op.Op]; reserved {
		return nil, jsonPathError(path.Key("op"), fmt.Sprintf("reserved operation %q cannot be used as a numeric function", op.Op))
	}
	fn, err := parseJSONFunction(op.Op, op.Args, path, depth+1, cfg)
	if err != nil {
		return nil, err
	}
	if !isNumericExpression(fn) {
		return nil, jsonPathError(path, "expected numeric expression")
	}
	return fn, nil
}

func parseJSONArithmeticExpression(name string, rawArgs []json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (*ArithmeticExpression, error) {
	if len(rawArgs) != 2 {
		return nil, jsonPathError(path.Key("args"), "expected exactly 2 arguments")
	}
	left, err := parseJSONNumericExpression(rawArgs[0], path.Key("args").Index(0), depth+1, cfg)
	if err != nil {
		return nil, err
	}
	right, err := parseJSONNumericExpression(rawArgs[1], path.Key("args").Index(1), depth+1, cfg)
	if err != nil {
		return nil, err
	}
	return &ArithmeticExpression{Op: ArithmeticOp(name), Left: left, Right: right, Src: jsonSpan(path)}, nil
}

func isJSONArithmeticOp(op string) bool {
	switch op {
	case "+", "-", "*", "/", "^", "%", "div":
		return true
	default:
		return false
	}
}

func parseJSONCharacterFunction(name string, rawArgs []json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (*FunctionCall, error) {
	if len(rawArgs) != 1 {
		return nil, jsonPathError(path.Key("args"), "expected exactly one argument")
	}
	arg, err := parseJSONCharacterExpression(rawArgs[0], path.Key("args").Index(0), depth+1, cfg)
	if err != nil {
		return nil, err
	}
	def, err := validateFunctionCall(name, []Node{arg}, cfg, LanguageJSON, Location{ByteOffset: -1, CharOffset: -1, JSONPath: path.Key("op")})
	if err != nil {
		return nil, err
	}
	return &FunctionCall{Name: normalizeFunctionName(name), Args: []Node{arg}, ReturnTypes: cloneFunctionTypes(def.Returns), Src: jsonSpan(path)}, nil
}

func parseJSONFunction(name string, rawArgs []json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (*FunctionCall, error) {
	args := make([]Node, 0, len(rawArgs))
	for i, raw := range rawArgs {
		node, err := parseJSONNode(raw, path.Key("args").Index(i), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		args = append(args, node)
	}
	def, err := validateFunctionCall(name, args, cfg, LanguageJSON, Location{ByteOffset: -1, CharOffset: -1, JSONPath: path.Key("op")})
	if err != nil {
		return nil, err
	}
	return &FunctionCall{Name: normalizeFunctionName(name), Args: args, ReturnTypes: cloneFunctionTypes(def.Returns), Src: jsonSpan(path)}, nil
}

func parseJSONNode(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (Node, error) {
	if expr, err := parseJSONExpression(raw, path, depth, cfg); err == nil {
		return expr, nil
	}
	if scalar, err := parseJSONScalar(raw, path, depth, cfg); err == nil {
		return scalar, nil
	}
	if temporal, err := parseJSONTemporalInstance(raw, path, depth, cfg); err == nil {
		return temporal, nil
	} else if hasJSONTemporalInstanceKey(raw, path) {
		return nil, err
	}
	if geom, err := parseJSONGeometryLiteral(raw, path, depth, cfg); err == nil {
		return geom, nil
	} else if hasJSONGeometryLiteralKey(raw, path) {
		return nil, err
	}
	if array, err := parseJSONArrayLiteral(raw, path, depth, cfg); err == nil {
		return array, nil
	}
	return nil, jsonPathError(path, "expected CQL2 value")
}

func parseJSONScalarArgs(rawArgs []json.RawMessage, path JSONPath, depth int, cfg ParseConfig, minArgs, maxArgs int) ([]ScalarExpression, error) {
	if len(rawArgs) < minArgs || len(rawArgs) > maxArgs {
		if minArgs == maxArgs {
			return nil, jsonPathError(path, fmt.Sprintf("expected exactly %d arguments", minArgs))
		}
		return nil, jsonPathError(path, fmt.Sprintf("expected %d to %d arguments", minArgs, maxArgs))
	}
	args := make([]ScalarExpression, 0, len(rawArgs))
	for i, raw := range rawArgs {
		arg, err := parseJSONScalar(raw, path.Index(i), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}
	return args, nil
}

func parseJSONScalarArray(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) ([]ScalarExpression, error) {
	var items []json.RawMessage
	if err := unmarshalAt(raw, path, &items); err != nil {
		return nil, jsonPathError(path, "expected array")
	}
	values := make([]ScalarExpression, 0, len(items))
	for i, item := range items {
		value, err := parseJSONScalar(item, path.Index(i), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func parseJSONArrayLiteral(raw json.RawMessage, path JSONPath, depth int, cfg ParseConfig) (*ArrayLiteral, error) {
	if depth > cfg.MaxDepth {
		return nil, jsonPathError(path, "maximum parse depth exceeded")
	}
	var items []json.RawMessage
	if err := unmarshalAt(raw, path, &items); err != nil {
		return nil, err
	}
	values := make([]Node, 0, len(items))
	for i, item := range items {
		value, err := parseJSONNode(item, path.Index(i), depth+1, cfg)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return &ArrayLiteral{Values: values, Src: jsonSpan(path)}, nil
}

func parseJSONLiteral(raw json.RawMessage, path JSONPath) (*Literal, bool, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		return nil, false, jsonPathError(path, "invalid JSON value")
	}
	src := jsonSpan(path)
	switch v := value.(type) {
	case string:
		return &Literal{Kind: LiteralString, Value: v, Src: src}, true, nil
	case json.Number:
		canonical, err := canonicalNumber(v.String())
		if err != nil {
			return nil, true, jsonPathError(path, err.Error())
		}
		return &Literal{Kind: LiteralNumber, Value: canonical, Src: src}, true, nil
	case bool:
		return &Literal{Kind: LiteralBool, Value: v, Src: src}, true, nil
	case nil:
		return &Literal{Kind: LiteralNull, Value: nil, Src: src}, true, nil
	default:
		return nil, false, nil
	}
}

func unmarshalAt(raw json.RawMessage, path JSONPath, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(out); err != nil {
		return jsonPathError(path, err.Error())
	}
	return nil
}

func jsonSpan(path JSONPath) Span {
	return Span{Start: Location{ByteOffset: -1, CharOffset: -1, JSONPath: path}, End: Location{ByteOffset: -1, CharOffset: -1, JSONPath: path}}
}

func jsonPathError(path JSONPath, message string) *ParseError {
	return parseError(LanguageJSON, Location{ByteOffset: -1, CharOffset: -1, JSONPath: path}, message)
}

func jsonSyntaxError(input []byte, err error) *ParseError {
	loc := NoLocation()
	loc.ByteOffset = 0
	loc.CharOffset = 0
	if syntaxErr, ok := err.(*json.SyntaxError); ok {
		loc = locationForByteOffset(input, int(syntaxErr.Offset)-1)
	} else if typeErr, ok := err.(*json.UnmarshalTypeError); ok {
		loc = locationForByteOffset(input, int(typeErr.Offset)-1)
	}
	return &ParseError{Source: LanguageJSON, Location: loc, Message: err.Error(), Cause: err}
}

func locationForByteOffset(input []byte, byteOffset int) Location {
	if byteOffset < 0 {
		byteOffset = 0
	}
	if byteOffset > len(input) {
		byteOffset = len(input)
	}
	line, col, chars := 1, 1, 0
	for i, r := range string(input[:byteOffset]) {
		_ = i
		chars++
		if r == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return Location{ByteOffset: byteOffset, CharOffset: chars, Line: line, Column: col}
}
