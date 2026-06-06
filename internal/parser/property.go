package parser

import (
	"fmt"

	"github.com/cwygoda/gocql2/api"
)

const propertyTypeUnsupported api.PropertyType = "\x00unsupported"

type propertyRegistry struct {
	defs       map[string]api.PropertyDefinition
	restricted bool
}

func newPropertyRegistry(defs []api.PropertyDefinition, restricted bool) propertyRegistry {
	registry := propertyRegistry{restricted: restricted}
	if len(defs) == 0 {
		return registry
	}
	registry.defs = make(map[string]api.PropertyDefinition, len(defs))
	for _, def := range defs {
		if def.Name == "" {
			continue
		}
		registry.defs[def.Name] = def
	}
	return registry
}

func (r propertyRegistry) lookup(name string) (api.PropertyDefinition, bool) {
	if !r.restricted {
		return api.PropertyDefinition{Name: name, Type: api.PropertyTypeAny}, true
	}
	def, ok := r.defs[name]
	return def, ok
}

func validatePropertyName(name string, typ api.PropertyType, loc api.Location, source api.Language) error {
	if typ == api.PropertyTypeAny {
		return nil
	}
	if !isKnownPropertyType(typ) {
		return parseError(source, loc, fmt.Sprintf("property %q has unsupported type %q", name, typ))
	}
	return nil
}

func isKnownPropertyType(typ api.PropertyType) bool {
	switch typ {
	case api.PropertyTypeString, api.PropertyTypeNumber, api.PropertyTypeInteger, api.PropertyTypeBoolean,
		api.PropertyTypeDate, api.PropertyTypeTimestamp, api.PropertyTypeInterval,
		api.PropertyTypePoint, api.PropertyTypeMultiPoint, api.PropertyTypeLineString, api.PropertyTypeMultiLineString,
		api.PropertyTypePolygon, api.PropertyTypeMultiPolygon, api.PropertyTypeGeometry, api.PropertyTypeGeometryCollection,
		api.PropertyTypeArray:
		return true
	default:
		return false
	}
}

func propertyRef(name string, src api.Span, cfg ParseConfig, source api.Language, errLoc api.Location) (*api.PropertyRef, error) {
	def, ok := cfg.properties.lookup(name)
	if !ok {
		return nil, parseError(source, errLoc, fmt.Sprintf("property %q is not allowed", name))
	}
	if err := validatePropertyName(name, def.Type, errLoc, source); err != nil {
		return nil, err
	}
	return &api.PropertyRef{Name: name, Type: def.Type, Src: src}, nil
}

func isCharacterExpression(scalar api.ScalarExpression) bool {
	switch value := scalar.(type) {
	case *api.Literal:
		return value.Kind == api.LiteralString
	case *api.PropertyRef:
		return value.Type == api.PropertyTypeAny || value.Type == api.PropertyTypeString
	case *api.FunctionCall:
		return functionCallReturns(value, api.FunctionTypeString)
	default:
		return false
	}
}

func isNumericExpression(scalar api.ScalarExpression) bool {
	switch value := scalar.(type) {
	case *api.Literal:
		return value.Kind == api.LiteralNumber
	case *api.PropertyRef:
		return value.Type == api.PropertyTypeAny || isNumericPropertyType(value.Type)
	case *api.FunctionCall:
		return functionCallReturns(value, api.FunctionTypeNumber) || functionCallReturns(value, api.FunctionTypeInteger)
	case *api.ArithmeticExpression:
		return true
	default:
		return false
	}
}

func validateScalarExpression(scalar api.ScalarExpression, source api.Language) error {
	prop, ok := scalar.(*api.PropertyRef)
	if !ok || prop.Type == api.PropertyTypeAny || isScalarPropertyType(prop.Type) {
		return nil
	}
	return parseError(source, prop.Span().Start, fmt.Sprintf("property %q of type %q cannot be used as a scalar expression", prop.Name, prop.Type))
}

func validatePropertyPropertyConformance(cfg ParseConfig, source api.Language, operands ...api.ScalarExpression) error {
	if cfg.conformance.propertyProperty || len(operands) == 0 {
		return nil
	}
	if !scalarExpressionReferencesProperty(operands[0]) {
		return parseError(source, operands[0].Span().Start, "comparison requires a property-valued left operand unless property-property conformance is enabled")
	}
	for _, operand := range operands[1:] {
		if scalarExpressionReferencesProperty(operand) {
			return parseError(source, operand.Span().Start, "property-valued comparison operands require property-property conformance")
		}
	}
	return nil
}

func scalarExpressionReferencesProperty(scalar api.ScalarExpression) bool {
	switch value := scalar.(type) {
	case *api.PropertyRef:
		return true
	case *api.FunctionCall:
		for _, arg := range value.Args {
			if nodeReferencesProperty(arg) {
				return true
			}
		}
	case *api.ArithmeticExpression:
		return scalarExpressionReferencesProperty(value.Left) || scalarExpressionReferencesProperty(value.Right)
	}
	return false
}

func nodeReferencesProperty(node api.Node) bool {
	switch value := node.(type) {
	case *api.ArrayLiteral:
		for _, item := range value.Values {
			if nodeReferencesProperty(item) {
				return true
			}
		}
	case api.ScalarExpression:
		return scalarExpressionReferencesProperty(value)
	case *api.LogicalExpression:
		for _, arg := range value.Args {
			if nodeReferencesProperty(arg) {
				return true
			}
		}
	case *api.ComparisonExpression:
		return scalarExpressionReferencesProperty(value.Left) || scalarExpressionReferencesProperty(value.Right)
	case *api.LikeExpression:
		return scalarExpressionReferencesProperty(value.Expr) || scalarExpressionReferencesProperty(value.Pattern)
	case *api.BetweenExpression:
		return scalarExpressionReferencesProperty(value.Expr) || scalarExpressionReferencesProperty(value.Lower) || scalarExpressionReferencesProperty(value.Upper)
	case *api.InExpression:
		if scalarExpressionReferencesProperty(value.Expr) {
			return true
		}
		for _, item := range value.Values {
			if scalarExpressionReferencesProperty(item) {
				return true
			}
		}
	case *api.IsNullExpression:
		return nodeReferencesProperty(value.Expr)
	case *api.TemporalInterval:
		return nodeReferencesProperty(value.Start) || nodeReferencesProperty(value.End)
	case *api.SpatialPredicateExpression:
		return nodeReferencesProperty(value.Left) || nodeReferencesProperty(value.Right)
	case *api.TemporalPredicateExpression:
		return nodeReferencesProperty(value.Left) || nodeReferencesProperty(value.Right)
	case *api.ArrayPredicateExpression:
		return nodeReferencesProperty(value.Left) || nodeReferencesProperty(value.Right)
	}
	return false
}

func validateComparisonOperands(op api.ComparisonOp, left, right api.ScalarExpression, source api.Language) error {
	if err := validateScalarExpression(left, source); err != nil {
		return err
	}
	if err := validateScalarExpression(right, source); err != nil {
		return err
	}
	leftType := scalarExpressionType(left)
	rightType := scalarExpressionType(right)
	if !areComparableTypes(leftType, rightType) {
		return parseError(source, right.Span().Start, fmt.Sprintf("cannot compare %s expression to %s expression", describePropertyType(leftType), describePropertyType(rightType)))
	}
	if op != api.OpEqual && op != api.OpNotEqual && !isOrderedComparisonType(leftType, rightType) {
		return parseError(source, left.Span().Start, fmt.Sprintf("operator %q is not supported for %s expressions", op, describePropertyType(leftType)))
	}
	return nil
}

func validateInOperands(expr api.ScalarExpression, values []api.ScalarExpression, source api.Language) error {
	if err := validateScalarExpression(expr, source); err != nil {
		return err
	}
	exprType := scalarExpressionType(expr)
	for _, value := range values {
		if err := validateScalarExpression(value, source); err != nil {
			return err
		}
		valueType := scalarExpressionType(value)
		if !areComparableTypes(exprType, valueType) {
			return parseError(source, value.Span().Start, fmt.Sprintf("IN list value has type %s, expected %s", describePropertyType(valueType), describePropertyType(exprType)))
		}
	}
	return nil
}

func validateArrayPredicateOperands(left, right api.Node, source api.Language) error {
	if err := validateArrayOperand(left, source); err != nil {
		return err
	}
	return validateArrayOperand(right, source)
}

func validateArrayOperand(node api.Node, source api.Language) error {
	switch value := node.(type) {
	case *api.ArrayLiteral:
		return nil
	case *api.PropertyRef:
		if value.Type == api.PropertyTypeAny || value.Type == api.PropertyTypeArray {
			return nil
		}
		return parseError(source, value.Span().Start, fmt.Sprintf("property %q of type %q cannot be used as an array operand", value.Name, value.Type))
	case *api.FunctionCall:
		if functionCallReturns(value, api.FunctionTypeArray) || functionCallReturnsExact(value, api.FunctionTypeAny) {
			return nil
		}
		return parseError(source, value.Span().Start, fmt.Sprintf("function %q does not return array", value.Name))
	default:
		return parseError(source, node.Span().Start, "expected array operand", "array", "array property", "array function")
	}
}

func functionCallReturnsExact(call *api.FunctionCall, typ api.FunctionType) bool {
	for _, ret := range call.ReturnTypes {
		if ret == typ {
			return true
		}
	}
	return false
}

func scalarExpressionType(scalar api.ScalarExpression) api.PropertyType {
	switch value := scalar.(type) {
	case *api.Literal:
		switch value.Kind {
		case api.LiteralString:
			return api.PropertyTypeString
		case api.LiteralNumber:
			return api.PropertyTypeNumber
		case api.LiteralBool:
			return api.PropertyTypeBoolean
		default:
			return api.PropertyTypeAny
		}
	case *api.PropertyRef:
		return value.Type
	case *api.ArithmeticExpression:
		return api.PropertyTypeNumber
	case *api.TemporalInstant:
		if value.Kind == api.TemporalInstantDate {
			return api.PropertyTypeDate
		}
		return api.PropertyTypeTimestamp
	case *api.FunctionCall:
		return functionReturnPropertyType(value)
	default:
		return propertyTypeUnsupported
	}
}

func areComparableTypes(left, right api.PropertyType) bool {
	if isUnsupportedPropertyType(left) || isUnsupportedPropertyType(right) {
		return false
	}
	if left == api.PropertyTypeAny || right == api.PropertyTypeAny {
		return true
	}
	if isNumericPropertyType(left) && isNumericPropertyType(right) {
		return true
	}
	if isInstantPropertyType(left) && isInstantPropertyType(right) {
		return true
	}
	return left == right
}

func isOrderedComparisonType(left, right api.PropertyType) bool {
	if isUnsupportedPropertyType(left) || isUnsupportedPropertyType(right) {
		return false
	}
	if left == api.PropertyTypeAny || right == api.PropertyTypeAny {
		return true
	}
	if isNumericPropertyType(left) && isNumericPropertyType(right) {
		return true
	}
	if isInstantPropertyType(left) && isInstantPropertyType(right) {
		return true
	}
	return left == right && (left == api.PropertyTypeString || left == api.PropertyTypeBoolean)
}

func isScalarPropertyType(typ api.PropertyType) bool {
	switch typ {
	case api.PropertyTypeString, api.PropertyTypeNumber, api.PropertyTypeInteger, api.PropertyTypeBoolean, api.PropertyTypeDate, api.PropertyTypeTimestamp:
		return true
	default:
		return false
	}
}

func isNumericPropertyType(typ api.PropertyType) bool {
	return typ == api.PropertyTypeNumber || typ == api.PropertyTypeInteger
}

func isInstantPropertyType(typ api.PropertyType) bool {
	return typ == api.PropertyTypeDate || typ == api.PropertyTypeTimestamp
}

func isUnsupportedPropertyType(typ api.PropertyType) bool {
	return typ == propertyTypeUnsupported || typ != api.PropertyTypeAny && !isKnownPropertyType(typ)
}

func describePropertyType(typ api.PropertyType) string {
	if typ == api.PropertyTypeAny {
		return "untyped"
	}
	if isUnsupportedPropertyType(typ) {
		return "unsupported"
	}
	return string(typ)
}
