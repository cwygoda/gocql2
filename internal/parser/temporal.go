package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cwygoda/cql2/api"
)

var temporalPredicateOps = map[string]api.TemporalPredicateOp{
	"t_after":        api.TemporalOpAfter,
	"t_before":       api.TemporalOpBefore,
	"t_contains":     api.TemporalOpContains,
	"t_disjoint":     api.TemporalOpDisjoint,
	"t_during":       api.TemporalOpDuring,
	"t_equals":       api.TemporalOpEquals,
	"t_finishedby":   api.TemporalOpFinishedBy,
	"t_finishes":     api.TemporalOpFinishes,
	"t_intersects":   api.TemporalOpIntersects,
	"t_meets":        api.TemporalOpMeets,
	"t_metby":        api.TemporalOpMetBy,
	"t_overlappedby": api.TemporalOpOverlappedBy,
	"t_overlaps":     api.TemporalOpOverlaps,
	"t_startedby":    api.TemporalOpStartedBy,
	"t_starts":       api.TemporalOpStarts,
}

var jsonTemporalPredicateOps = map[string]api.TemporalPredicateOp{
	"t_after":        api.TemporalOpAfter,
	"t_before":       api.TemporalOpBefore,
	"t_contains":     api.TemporalOpContains,
	"t_disjoint":     api.TemporalOpDisjoint,
	"t_during":       api.TemporalOpDuring,
	"t_equals":       api.TemporalOpEquals,
	"t_finishedBy":   api.TemporalOpFinishedBy,
	"t_finishes":     api.TemporalOpFinishes,
	"t_intersects":   api.TemporalOpIntersects,
	"t_meets":        api.TemporalOpMeets,
	"t_metBy":        api.TemporalOpMetBy,
	"t_overlappedBy": api.TemporalOpOverlappedBy,
	"t_overlaps":     api.TemporalOpOverlaps,
	"t_startedBy":    api.TemporalOpStartedBy,
	"t_starts":       api.TemporalOpStarts,
}

var (
	dateLiteralPattern      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timestampLiteralPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$`)
)

func isTemporalPredicateOp(name string) (api.TemporalPredicateOp, bool) {
	op, ok := temporalPredicateOps[strings.ToLower(name)]
	return op, ok
}

func isJSONTemporalPredicateOp(name string) (api.TemporalPredicateOp, bool) {
	op, ok := jsonTemporalPredicateOps[name]
	return op, ok
}

func isIntervalOnlyTemporalPredicate(op api.TemporalPredicateOp) bool {
	switch op {
	case api.TemporalOpContains, api.TemporalOpDuring, api.TemporalOpFinishedBy, api.TemporalOpFinishes,
		api.TemporalOpMeets, api.TemporalOpMetBy, api.TemporalOpOverlappedBy, api.TemporalOpOverlaps,
		api.TemporalOpStartedBy, api.TemporalOpStarts:
		return true
	default:
		return false
	}
}

func temporalInstantKindFromString(value string) (api.TemporalInstantKind, error) {
	if value == ".." {
		return "", fmt.Errorf("unbounded marker is only allowed as an interval endpoint")
	}
	if err := validateDateLiteral(value); err == nil {
		return api.TemporalInstantDate, nil
	}
	if err := validateTimestampLiteral(value); err == nil {
		return api.TemporalInstantTimestamp, nil
	}
	if strings.Contains(value, "T") {
		return "", validateTimestampLiteral(value)
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

func validateTimestampLiteral(value string) error {
	if !timestampLiteralPattern.MatchString(value) {
		return errors.New("timestamp must be an RFC3339 UTC timestamp ending in Z")
	}
	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		return fmt.Errorf("invalid timestamp %q", value)
	}
	return nil
}

func validateTemporalPredicateOperands(op api.TemporalPredicateOp, left, right api.Node, source api.Language) error {
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

func validateTemporalOperand(node api.Node, source api.Language) error {
	switch value := node.(type) {
	case *api.TemporalInstant, *api.TemporalInterval:
		return nil
	case *api.PropertyRef:
		if value.Type == api.PropertyTypeAny || isTemporalPropertyType(value.Type) {
			return nil
		}
		return parseError(source, value.Span().Start, fmt.Sprintf("property %q of type %q cannot be used as a temporal operand", value.Name, value.Type))
	case *api.FunctionCall:
		if functionReturnsTemporal(value) || functionCallReturnsExact(value, api.FunctionTypeAny) {
			return nil
		}
		return parseError(source, value.Span().Start, fmt.Sprintf("function %q does not return temporal", value.Name))
	default:
		return parseError(source, node.Span().Start, "expected temporal operand", "date", "timestamp", "interval", "temporal property", "temporal function")
	}
}

func validateTemporalIntervalOperands(start, end api.Node, source api.Language) error {
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
	if startInstant, ok := start.(*api.TemporalInstant); ok {
		if endInstant, ok := end.(*api.TemporalInstant); ok && startInstant.Kind == endInstant.Kind && temporalInstantAfter(startInstant, endInstant) {
			return parseError(source, end.Span().Start, "interval start must not be after interval end")
		}
	}
	return nil
}

func validateTemporalIntervalEndpoint(node api.Node, source api.Language) (api.PropertyType, error) {
	switch value := node.(type) {
	case *api.TemporalUnbounded:
		return api.PropertyTypeAny, nil
	case *api.TemporalInstant:
		if value.Kind == api.TemporalInstantDate {
			return api.PropertyTypeDate, nil
		}
		return api.PropertyTypeTimestamp, nil
	case *api.PropertyRef:
		if value.Type == api.PropertyTypeAny || value.Type == api.PropertyTypeDate || value.Type == api.PropertyTypeTimestamp {
			return value.Type, nil
		}
		return api.PropertyTypeAny, parseError(source, value.Span().Start, fmt.Sprintf("property %q of type %q cannot be used as an interval endpoint", value.Name, value.Type))
	case *api.FunctionCall:
		endpointType, ok := temporalFunctionEndpointType(value)
		if !ok {
			return api.PropertyTypeAny, parseError(source, value.Span().Start, fmt.Sprintf("function %q does not return instant", value.Name))
		}
		if endpointType != api.PropertyTypeInterval {
			return endpointType, nil
		}
		return api.PropertyTypeAny, parseError(source, value.Span().Start, fmt.Sprintf("function %q returns interval and cannot be used as an interval endpoint", value.Name))
	default:
		return api.PropertyTypeAny, parseError(source, node.Span().Start, "expected interval endpoint", "date string", "timestamp string", "..", "instant property", "instant function")
	}
}

func temporalInstantAfter(left, right *api.TemporalInstant) bool {
	if left.Kind == api.TemporalInstantDate {
		leftTime, leftErr := time.Parse("2006-01-02", left.Value)
		rightTime, rightErr := time.Parse("2006-01-02", right.Value)
		return leftErr == nil && rightErr == nil && leftTime.After(rightTime)
	}
	leftTime, leftErr := time.Parse(time.RFC3339Nano, left.Value)
	rightTime, rightErr := time.Parse(time.RFC3339Nano, right.Value)
	return leftErr == nil && rightErr == nil && leftTime.After(rightTime)
}

func isTemporalIntervalOperand(node api.Node) bool {
	switch value := node.(type) {
	case *api.TemporalInterval:
		return true
	case *api.PropertyRef:
		return value.Type == api.PropertyTypeAny || value.Type == api.PropertyTypeInterval
	case *api.FunctionCall:
		return functionCallReturnsExact(value, api.FunctionTypeAny) || functionCallReturnsExact(value, api.FunctionTypeInterval)
	default:
		return false
	}
}

func isTemporalPropertyType(typ api.PropertyType) bool {
	return typ == api.PropertyTypeDate || typ == api.PropertyTypeTimestamp || typ == api.PropertyTypeInterval
}

func isKnownInstantType(typ api.PropertyType) bool {
	return typ == api.PropertyTypeDate || typ == api.PropertyTypeTimestamp
}

func functionReturnsTemporal(call *api.FunctionCall) bool {
	return functionCallReturns(call, api.FunctionTypeDate) || functionCallReturns(call, api.FunctionTypeTimestamp) ||
		functionCallReturns(call, api.FunctionTypeDateTime) || functionCallReturns(call, api.FunctionTypeInterval)
}

func temporalFunctionEndpointType(call *api.FunctionCall) (api.PropertyType, bool) {
	if functionCallReturnsExact(call, api.FunctionTypeAny) {
		return api.PropertyTypeAny, true
	}
	if functionCallReturnsExact(call, api.FunctionTypeInterval) {
		return api.PropertyTypeInterval, true
	}
	if functionCallReturnsExact(call, api.FunctionTypeDate) {
		return api.PropertyTypeDate, true
	}
	if functionCallReturnsExact(call, api.FunctionTypeTimestamp) || functionCallReturnsExact(call, api.FunctionTypeDateTime) {
		return api.PropertyTypeTimestamp, true
	}
	return api.PropertyTypeAny, false
}
