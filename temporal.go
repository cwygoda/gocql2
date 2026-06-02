package gocql2

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var temporalPredicateOps = map[string]TemporalPredicateOp{
	"t_after":        TemporalOpAfter,
	"t_before":       TemporalOpBefore,
	"t_contains":     TemporalOpContains,
	"t_disjoint":     TemporalOpDisjoint,
	"t_during":       TemporalOpDuring,
	"t_equals":       TemporalOpEquals,
	"t_finishedby":   TemporalOpFinishedBy,
	"t_finishes":     TemporalOpFinishes,
	"t_intersects":   TemporalOpIntersects,
	"t_meets":        TemporalOpMeets,
	"t_metby":        TemporalOpMetBy,
	"t_overlappedby": TemporalOpOverlappedBy,
	"t_overlaps":     TemporalOpOverlaps,
	"t_startedby":    TemporalOpStartedBy,
	"t_starts":       TemporalOpStarts,
}

var (
	dateLiteralPattern             = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	strictTimestampLiteralPattern  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$`)
	relaxedTimestampLiteralPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$`)
)

func isTemporalPredicateOp(name string) (TemporalPredicateOp, bool) {
	op, ok := temporalPredicateOps[strings.ToLower(name)]
	return op, ok
}

func isIntervalOnlyTemporalPredicate(op TemporalPredicateOp) bool {
	switch op {
	case TemporalOpContains, TemporalOpDuring, TemporalOpFinishedBy, TemporalOpFinishes,
		TemporalOpMeets, TemporalOpMetBy, TemporalOpOverlappedBy, TemporalOpOverlaps,
		TemporalOpStartedBy, TemporalOpStarts:
		return true
	default:
		return false
	}
}

func temporalInstantKindFromString(value string, strictUTC bool) (TemporalInstantKind, error) {
	if value == ".." {
		return "", fmt.Errorf("unbounded marker is only allowed as an interval endpoint")
	}
	if err := validateDateLiteral(value); err == nil {
		return TemporalInstantDate, nil
	}
	if err := validateTimestampLiteral(value, strictUTC); err == nil {
		return TemporalInstantTimestamp, nil
	}
	if strings.Contains(value, "T") {
		return "", validateTimestampLiteral(value, strictUTC)
	}
	return "", validateDateLiteral(value)
}

func validateDateLiteral(value string) error {
	if !dateLiteralPattern.MatchString(value) {
		return fmt.Errorf("date must match YYYY-MM-DD")
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("invalid date %q", value)
	}
	return nil
}

func validateTimestampLiteral(value string, strictUTC bool) error {
	pattern := relaxedTimestampLiteralPattern
	message := "timestamp must be an RFC3339 timestamp"
	if strictUTC {
		pattern = strictTimestampLiteralPattern
		message = "timestamp must be an RFC3339 UTC timestamp ending in Z"
	}
	if !pattern.MatchString(value) {
		return errors.New(message)
	}
	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		return fmt.Errorf("invalid timestamp %q", value)
	}
	return nil
}

func validateTemporalPredicateOperands(op TemporalPredicateOp, left, right Node, source Language) error {
	if err := validateTemporalOperand(left, source); err != nil {
		return err
	}
	if err := validateTemporalOperand(right, source); err != nil {
		return err
	}
	if !isIntervalOnlyTemporalPredicate(op) {
		return nil
	}
	if !isTemporalIntervalOperand(left) {
		return parseError(source, left.Span().Start, fmt.Sprintf("%s operands must be intervals", strings.ToUpper(string(op))))
	}
	if !isTemporalIntervalOperand(right) {
		return parseError(source, right.Span().Start, fmt.Sprintf("%s operands must be intervals", strings.ToUpper(string(op))))
	}
	return nil
}

func validateTemporalOperand(node Node, source Language) error {
	switch value := node.(type) {
	case *TemporalInstant, *TemporalInterval:
		return nil
	case *PropertyRef:
		if value.Type == PropertyTypeAny || isTemporalPropertyType(value.Type) {
			return nil
		}
		return parseError(source, value.Span().Start, fmt.Sprintf("property %q of type %q cannot be used as a temporal operand", value.Name, value.Type))
	case *FunctionCall:
		if functionReturnsTemporal(value) || functionCallReturnsExact(value, FunctionTypeAny) {
			return nil
		}
		return parseError(source, value.Span().Start, fmt.Sprintf("function %q does not return temporal", value.Name))
	default:
		return parseError(source, node.Span().Start, "expected temporal operand", "date", "timestamp", "interval", "temporal property", "temporal function")
	}
}

func validateTemporalIntervalOperands(start, end Node, source Language) error {
	startType, err := validateTemporalIntervalEndpoint(start, source)
	if err != nil {
		return err
	}
	endType, err := validateTemporalIntervalEndpoint(end, source)
	if err != nil {
		return err
	}
	if isKnownInstantType(startType) && isKnownInstantType(endType) && startType != endType {
		return parseError(source, end.Span().Start, "interval endpoints must have matching temporal granularity")
	}
	if startInstant, ok := start.(*TemporalInstant); ok {
		if endInstant, ok := end.(*TemporalInstant); ok && startInstant.Kind == endInstant.Kind && temporalInstantAfter(startInstant, endInstant) {
			return parseError(source, end.Span().Start, "interval start must not be after interval end")
		}
	}
	return nil
}

func validateTemporalIntervalEndpoint(node Node, source Language) (PropertyType, error) {
	switch value := node.(type) {
	case *TemporalUnbounded:
		return PropertyTypeAny, nil
	case *TemporalInstant:
		if value.Kind == TemporalInstantDate {
			return PropertyTypeDate, nil
		}
		return PropertyTypeTimestamp, nil
	case *PropertyRef:
		if value.Type == PropertyTypeAny || value.Type == PropertyTypeDate || value.Type == PropertyTypeTimestamp {
			return value.Type, nil
		}
		return PropertyTypeAny, parseError(source, value.Span().Start, fmt.Sprintf("property %q of type %q cannot be used as an interval endpoint", value.Name, value.Type))
	case *FunctionCall:
		endpointType, ok := temporalFunctionEndpointType(value)
		if !ok {
			return PropertyTypeAny, parseError(source, value.Span().Start, fmt.Sprintf("function %q does not return instant", value.Name))
		}
		if endpointType != PropertyTypeInterval {
			return endpointType, nil
		}
		return PropertyTypeAny, parseError(source, value.Span().Start, fmt.Sprintf("function %q returns interval and cannot be used as an interval endpoint", value.Name))
	default:
		return PropertyTypeAny, parseError(source, node.Span().Start, "expected interval endpoint", "date string", "timestamp string", "..", "instant property", "instant function")
	}
}

func temporalInstantAfter(left, right *TemporalInstant) bool {
	if left.Kind == TemporalInstantDate {
		leftTime, leftErr := time.Parse("2006-01-02", left.Value)
		rightTime, rightErr := time.Parse("2006-01-02", right.Value)
		return leftErr == nil && rightErr == nil && leftTime.After(rightTime)
	}
	leftTime, leftErr := time.Parse(time.RFC3339Nano, left.Value)
	rightTime, rightErr := time.Parse(time.RFC3339Nano, right.Value)
	return leftErr == nil && rightErr == nil && leftTime.After(rightTime)
}

func isTemporalIntervalOperand(node Node) bool {
	switch value := node.(type) {
	case *TemporalInterval:
		return true
	case *PropertyRef:
		return value.Type == PropertyTypeAny || value.Type == PropertyTypeInterval
	case *FunctionCall:
		return functionCallReturnsExact(value, FunctionTypeAny) || functionCallReturnsExact(value, FunctionTypeInterval)
	default:
		return false
	}
}

func isTemporalPropertyType(typ PropertyType) bool {
	return typ == PropertyTypeDate || typ == PropertyTypeTimestamp || typ == PropertyTypeInterval
}

func isKnownInstantType(typ PropertyType) bool {
	return typ == PropertyTypeDate || typ == PropertyTypeTimestamp
}

func functionReturnsTemporal(call *FunctionCall) bool {
	return functionCallReturns(call, FunctionTypeDate) || functionCallReturns(call, FunctionTypeTimestamp) ||
		functionCallReturns(call, FunctionTypeDateTime) || functionCallReturns(call, FunctionTypeInterval)
}

func temporalFunctionEndpointType(call *FunctionCall) (PropertyType, bool) {
	if functionCallReturnsExact(call, FunctionTypeAny) {
		return PropertyTypeAny, true
	}
	if functionCallReturnsExact(call, FunctionTypeInterval) {
		return PropertyTypeInterval, true
	}
	if functionCallReturnsExact(call, FunctionTypeDate) {
		return PropertyTypeDate, true
	}
	if functionCallReturnsExact(call, FunctionTypeTimestamp) || functionCallReturnsExact(call, FunctionTypeDateTime) {
		return PropertyTypeTimestamp, true
	}
	return PropertyTypeAny, false
}
