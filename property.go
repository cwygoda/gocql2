package gocql2

import "fmt"

// PropertyType identifies the advertised data type for a queryable property.
type PropertyType string

// Supported property data types. The values align with the CQL2/OGC API
// queryables type names, lower-cased for API consistency.
const (
	// PropertyTypeAny permits the property in any scalar context. It is used by
	// name-only property allow-lists where no type information is available.
	PropertyTypeAny PropertyType = ""

	PropertyTypeString             PropertyType = "string"
	PropertyTypeNumber             PropertyType = "number"
	PropertyTypeInteger            PropertyType = "integer"
	PropertyTypeBoolean            PropertyType = "boolean"
	PropertyTypeDate               PropertyType = "date"
	PropertyTypeTimestamp          PropertyType = "timestamp"
	PropertyTypeInterval           PropertyType = "interval"
	PropertyTypePoint              PropertyType = "point"
	PropertyTypeMultiPoint         PropertyType = "multipoint"
	PropertyTypeLineString         PropertyType = "linestring"
	PropertyTypeMultiLineString    PropertyType = "multilinestring"
	PropertyTypePolygon            PropertyType = "polygon"
	PropertyTypeMultiPolygon       PropertyType = "multipolygon"
	PropertyTypeGeometry           PropertyType = "geometry"
	PropertyTypeGeometryCollection PropertyType = "geometrycollection"
	PropertyTypeArray              PropertyType = "array"

	propertyTypeUnsupported PropertyType = "\x00unsupported"
)

// PropertyDefinition describes one allowed queryable property.
type PropertyDefinition struct {
	Name string
	Type PropertyType
}

type propertyRegistry struct {
	defs       map[string]PropertyDefinition
	restricted bool
}

func newPropertyRegistry(defs []PropertyDefinition, restricted bool) propertyRegistry {
	registry := propertyRegistry{restricted: restricted}
	if len(defs) == 0 {
		return registry
	}
	registry.defs = make(map[string]PropertyDefinition, len(defs))
	for _, def := range defs {
		if def.Name == "" {
			continue
		}
		registry.defs[def.Name] = def
	}
	return registry
}

func (r propertyRegistry) lookup(name string) (PropertyDefinition, bool) {
	if !r.restricted {
		return PropertyDefinition{Name: name, Type: PropertyTypeAny}, true
	}
	def, ok := r.defs[name]
	return def, ok
}

func validatePropertyName(name string, typ PropertyType, loc Location, source Language) error {
	if typ == PropertyTypeAny {
		return nil
	}
	if !isKnownPropertyType(typ) {
		return parseError(source, loc, fmt.Sprintf("property %q has unsupported type %q", name, typ))
	}
	return nil
}

func isKnownPropertyType(typ PropertyType) bool {
	switch typ {
	case PropertyTypeString, PropertyTypeNumber, PropertyTypeInteger, PropertyTypeBoolean,
		PropertyTypeDate, PropertyTypeTimestamp, PropertyTypeInterval,
		PropertyTypePoint, PropertyTypeMultiPoint, PropertyTypeLineString, PropertyTypeMultiLineString,
		PropertyTypePolygon, PropertyTypeMultiPolygon, PropertyTypeGeometry, PropertyTypeGeometryCollection,
		PropertyTypeArray:
		return true
	default:
		return false
	}
}

func propertyRef(name string, src Span, cfg ParseConfig, source Language, errLoc Location) (*PropertyRef, error) {
	def, ok := cfg.properties.lookup(name)
	if !ok {
		return nil, parseError(source, errLoc, fmt.Sprintf("property %q is not allowed", name))
	}
	if err := validatePropertyName(name, def.Type, errLoc, source); err != nil {
		return nil, err
	}
	return &PropertyRef{Name: name, Type: def.Type, Src: src}, nil
}

func isCharacterExpression(scalar ScalarExpression) bool {
	switch value := scalar.(type) {
	case *Literal:
		return value.Kind == LiteralString
	case *PropertyRef:
		return value.Type == PropertyTypeAny || value.Type == PropertyTypeString
	case *FunctionCall:
		return functionCallReturns(value, FunctionTypeString)
	default:
		return false
	}
}

func isNumericExpression(scalar ScalarExpression) bool {
	switch value := scalar.(type) {
	case *Literal:
		return value.Kind == LiteralNumber
	case *PropertyRef:
		return value.Type == PropertyTypeAny || isNumericPropertyType(value.Type)
	case *FunctionCall:
		return functionCallReturns(value, FunctionTypeNumber) || functionCallReturns(value, FunctionTypeInteger)
	case *ArithmeticExpression:
		return true
	default:
		return false
	}
}

func validateScalarExpression(scalar ScalarExpression, source Language) error {
	prop, ok := scalar.(*PropertyRef)
	if !ok || prop.Type == PropertyTypeAny || isScalarPropertyType(prop.Type) {
		return nil
	}
	return parseError(source, prop.Span().Start, fmt.Sprintf("property %q of type %q cannot be used as a scalar expression", prop.Name, prop.Type))
}

func validateComparisonOperands(op ComparisonOp, left, right ScalarExpression, source Language) error {
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
	if op != OpEqual && op != OpNotEqual && !isOrderedComparisonType(leftType, rightType) {
		return parseError(source, left.Span().Start, fmt.Sprintf("operator %q is not supported for %s expressions", op, describePropertyType(leftType)))
	}
	return nil
}

func validateInOperands(expr ScalarExpression, values []ScalarExpression, source Language) error {
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

func validateArrayPredicateOperands(left, right Node, source Language) error {
	if err := validateArrayOperand(left, source); err != nil {
		return err
	}
	return validateArrayOperand(right, source)
}

func validateArrayOperand(node Node, source Language) error {
	switch value := node.(type) {
	case *ArrayLiteral:
		return nil
	case *PropertyRef:
		if value.Type == PropertyTypeAny || value.Type == PropertyTypeArray {
			return nil
		}
		return parseError(source, value.Span().Start, fmt.Sprintf("property %q of type %q cannot be used as an array operand", value.Name, value.Type))
	case *FunctionCall:
		if functionCallReturns(value, FunctionTypeArray) || functionCallReturnsExact(value, FunctionTypeAny) {
			return nil
		}
		return parseError(source, value.Span().Start, fmt.Sprintf("function %q does not return array", value.Name))
	default:
		return parseError(source, node.Span().Start, "expected array operand", "array", "array property", "array function")
	}
}

func functionCallReturnsExact(call *FunctionCall, typ FunctionType) bool {
	for _, ret := range call.ReturnTypes {
		if ret == typ {
			return true
		}
	}
	return false
}

func scalarExpressionType(scalar ScalarExpression) PropertyType {
	switch value := scalar.(type) {
	case *Literal:
		switch value.Kind {
		case LiteralString:
			return PropertyTypeString
		case LiteralNumber:
			return PropertyTypeNumber
		case LiteralBool:
			return PropertyTypeBoolean
		default:
			return PropertyTypeAny
		}
	case *PropertyRef:
		return value.Type
	case *ArithmeticExpression:
		return PropertyTypeNumber
	case *TemporalInstant:
		if value.Kind == TemporalInstantDate {
			return PropertyTypeDate
		}
		return PropertyTypeTimestamp
	case *FunctionCall:
		return functionReturnPropertyType(value)
	default:
		return propertyTypeUnsupported
	}
}

func areComparableTypes(left, right PropertyType) bool {
	if isUnsupportedPropertyType(left) || isUnsupportedPropertyType(right) {
		return false
	}
	if left == PropertyTypeAny || right == PropertyTypeAny {
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

func isOrderedComparisonType(left, right PropertyType) bool {
	if isUnsupportedPropertyType(left) || isUnsupportedPropertyType(right) {
		return false
	}
	if left == PropertyTypeAny || right == PropertyTypeAny {
		return true
	}
	if isNumericPropertyType(left) && isNumericPropertyType(right) {
		return true
	}
	if isInstantPropertyType(left) && isInstantPropertyType(right) {
		return true
	}
	return left == PropertyTypeString && right == PropertyTypeString
}

func isScalarPropertyType(typ PropertyType) bool {
	switch typ {
	case PropertyTypeString, PropertyTypeNumber, PropertyTypeInteger, PropertyTypeBoolean, PropertyTypeDate, PropertyTypeTimestamp:
		return true
	default:
		return false
	}
}

func isNumericPropertyType(typ PropertyType) bool {
	return typ == PropertyTypeNumber || typ == PropertyTypeInteger
}

func isInstantPropertyType(typ PropertyType) bool {
	return typ == PropertyTypeDate || typ == PropertyTypeTimestamp
}

func isUnsupportedPropertyType(typ PropertyType) bool {
	return typ == propertyTypeUnsupported || typ != PropertyTypeAny && !isKnownPropertyType(typ)
}

func describePropertyType(typ PropertyType) string {
	if typ == PropertyTypeAny {
		return "untyped"
	}
	if isUnsupportedPropertyType(typ) {
		return "unsupported"
	}
	return string(typ)
}
