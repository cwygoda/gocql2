package gocql2

import (
	"fmt"
	"sort"
	"strings"
)

// FunctionType identifies a CQL2 function argument or return type.
type FunctionType string

// Supported function signature types. The non-array non-empty values align with
// the OGC CQL2 function metadata schema; array is included for CQL2 array
// values and future array functions.
const (
	// FunctionTypeAny permits any CQL2 value. It is useful for name-only function
	// allow-lists where no signature metadata is available.
	FunctionTypeAny FunctionType = ""

	FunctionTypeString    FunctionType = "string"
	FunctionTypeNumber    FunctionType = "number"
	FunctionTypeInteger   FunctionType = "integer"
	FunctionTypeDate      FunctionType = "date"
	FunctionTypeTimestamp FunctionType = "timestamp"
	FunctionTypeDateTime  FunctionType = "datetime"
	FunctionTypeInterval  FunctionType = "interval"
	FunctionTypeGeometry  FunctionType = "geometry"
	FunctionTypeBoolean   FunctionType = "boolean"
	FunctionTypeArray     FunctionType = "array"
)

// Standard CQL2 text function names.
const (
	FunctionNameCaseI   = "casei"
	FunctionNameAccenti = "accenti"
)

// FunctionArgument describes one positional function argument. If Variadic is
// true, this argument must be the last argument and it may be repeated.
type FunctionArgument struct {
	Name     string
	Types    []FunctionType
	Variadic bool
}

// FunctionDefinition describes one allowed function signature.
type FunctionDefinition struct {
	Name    string
	Args    []FunctionArgument
	Returns []FunctionType
}

type functionRegistry struct {
	defs        map[string]FunctionDefinition
	initialized bool
}

// CaseIFunction returns the CQL2 CASEI text function definition.
func CaseIFunction() FunctionDefinition {
	return FunctionDefinition{
		Name:    FunctionNameCaseI,
		Args:    []FunctionArgument{{Name: "value", Types: []FunctionType{FunctionTypeString}}},
		Returns: []FunctionType{FunctionTypeString},
	}
}

// AccentiFunction returns the CQL2 ACCENTI text function definition.
func AccentiFunction() FunctionDefinition {
	return FunctionDefinition{
		Name:    FunctionNameAccenti,
		Args:    []FunctionArgument{{Name: "value", Types: []FunctionType{FunctionTypeString}}},
		Returns: []FunctionType{FunctionTypeString},
	}
}

// StandardTextFunctions returns the CQL2-standard text functions. CQL2 does not
// define a broader SQL-like string function library; CASEI and ACCENTI are the
// standardized text functions provided by the case/accent-insensitive comparison
// requirements classes.
func StandardTextFunctions() []FunctionDefinition {
	return []FunctionDefinition{CaseIFunction(), AccentiFunction()}
}

func newFunctionRegistry(defs []FunctionDefinition) functionRegistry {
	registry := functionRegistry{initialized: true}
	if len(defs) == 0 {
		return registry
	}
	registry.defs = make(map[string]FunctionDefinition, len(defs))
	for _, def := range defs {
		def = cloneFunctionDefinition(def)
		def.Name = normalizeFunctionName(def.Name)
		if def.Name == "" {
			continue
		}
		registry.defs[def.Name] = def
	}
	return registry
}

func functionRegistryDefaults() functionRegistry {
	return newFunctionRegistry(StandardTextFunctions())
}

func (r functionRegistry) lookup(name string) (FunctionDefinition, bool) {
	name = normalizeFunctionName(name)
	if !r.initialized {
		r = functionRegistryDefaults()
	}
	def, ok := r.defs[name]
	return def, ok
}

func validateFunctionCall(name string, args []Node, cfg ParseConfig, source Language, loc Location) (FunctionDefinition, error) {
	name = normalizeFunctionName(name)
	def, ok := cfg.functions.lookup(name)
	if !ok {
		return FunctionDefinition{}, parseError(source, loc, fmt.Sprintf("function %q is not allowed", name))
	}
	if err := validateFunctionDefinition(def, source, loc); err != nil {
		return FunctionDefinition{}, err
	}

	minArgs, maxArgs := functionArgumentBounds(def.Args)
	if len(args) < minArgs || maxArgs >= 0 && len(args) > maxArgs {
		return FunctionDefinition{}, parseError(source, loc, functionArgumentCountMessage(name, minArgs, maxArgs))
	}

	for i, arg := range args {
		spec := functionArgumentAt(def.Args, i)
		if len(spec.Types) == 0 {
			continue
		}
		actual := nodeFunctionTypes(arg)
		if !functionTypesOverlap(actual, spec.Types) {
			return FunctionDefinition{}, parseError(source, arg.Span().Start, fmt.Sprintf("argument %d to function %q has type %s, expected %s", i+1, name, describeFunctionTypes(actual), describeFunctionTypes(spec.Types)))
		}
	}
	return def, nil
}

func validateFunctionDefinition(def FunctionDefinition, source Language, loc Location) error {
	if def.Name == "" {
		return parseError(source, loc, "function name must not be empty")
	}
	if len(def.Returns) == 0 {
		return parseError(source, loc, fmt.Sprintf("function %q must declare at least one return type", normalizeFunctionName(def.Name)))
	}
	for _, ret := range def.Returns {
		if !isKnownFunctionType(ret) {
			return parseError(source, loc, fmt.Sprintf("function %q has unsupported return type %q", normalizeFunctionName(def.Name), ret))
		}
	}
	for i, arg := range def.Args {
		if arg.Variadic && i != len(def.Args)-1 {
			return parseError(source, loc, fmt.Sprintf("function %q has a variadic argument that is not last", normalizeFunctionName(def.Name)))
		}
		for _, typ := range arg.Types {
			if !isKnownFunctionType(typ) {
				return parseError(source, loc, fmt.Sprintf("function %q argument %d has unsupported type %q", normalizeFunctionName(def.Name), i+1, typ))
			}
		}
	}
	return nil
}

func functionArgumentBounds(args []FunctionArgument) (int, int) {
	if len(args) == 0 {
		return 0, 0
	}
	if args[len(args)-1].Variadic {
		return len(args) - 1, -1
	}
	return len(args), len(args)
}

func functionArgumentAt(args []FunctionArgument, index int) FunctionArgument {
	if index < len(args) {
		return args[index]
	}
	return args[len(args)-1]
}

func functionArgumentCountMessage(name string, minArgs, maxArgs int) string {
	switch {
	case maxArgs < 0:
		return fmt.Sprintf("function %q expects at least %d arguments", name, minArgs)
	case minArgs == maxArgs:
		return fmt.Sprintf("function %q expects exactly %d arguments", name, minArgs)
	default:
		return fmt.Sprintf("function %q expects %d to %d arguments", name, minArgs, maxArgs)
	}
}

func nodeFunctionTypes(node Node) []FunctionType {
	switch value := node.(type) {
	case *Literal:
		switch value.Kind {
		case LiteralString:
			return []FunctionType{FunctionTypeString}
		case LiteralNumber:
			return []FunctionType{FunctionTypeNumber}
		case LiteralBool:
			return []FunctionType{FunctionTypeBoolean}
		default:
			return []FunctionType{FunctionTypeAny}
		}
	case *PropertyRef:
		return []FunctionType{propertyFunctionType(value.Type)}
	case *ArithmeticExpression:
		return []FunctionType{FunctionTypeNumber}
	case *TemporalInstant:
		if value.Kind == TemporalInstantDate {
			return []FunctionType{FunctionTypeDate}
		}
		return []FunctionType{FunctionTypeTimestamp}
	case *TemporalInterval:
		return []FunctionType{FunctionTypeInterval}
	case *FunctionCall:
		if len(value.ReturnTypes) == 0 {
			return []FunctionType{FunctionTypeAny}
		}
		return cloneFunctionTypes(value.ReturnTypes)
	case *ArrayLiteral:
		return []FunctionType{FunctionTypeArray}
	case *GeometryLiteral:
		return []FunctionType{FunctionTypeGeometry}
	case Expression:
		return []FunctionType{FunctionTypeBoolean}
	default:
		return []FunctionType{FunctionTypeAny}
	}
}

func propertyFunctionType(typ PropertyType) FunctionType {
	switch typ {
	case PropertyTypeAny:
		return FunctionTypeAny
	case PropertyTypeString:
		return FunctionTypeString
	case PropertyTypeNumber:
		return FunctionTypeNumber
	case PropertyTypeInteger:
		return FunctionTypeInteger
	case PropertyTypeBoolean:
		return FunctionTypeBoolean
	case PropertyTypeDate:
		return FunctionTypeDate
	case PropertyTypeTimestamp:
		return FunctionTypeTimestamp
	case PropertyTypeInterval:
		return FunctionTypeInterval
	case PropertyTypePoint, PropertyTypeMultiPoint, PropertyTypeLineString, PropertyTypeMultiLineString,
		PropertyTypePolygon, PropertyTypeMultiPolygon, PropertyTypeGeometry, PropertyTypeGeometryCollection:
		return FunctionTypeGeometry
	case PropertyTypeArray:
		return FunctionTypeArray
	default:
		return FunctionTypeAny
	}
}

func functionTypeCompatible(expected, actual FunctionType) bool {
	if expected == FunctionTypeAny || actual == FunctionTypeAny || expected == actual {
		return true
	}
	if expected == FunctionTypeNumber && actual == FunctionTypeInteger {
		return true
	}
	if expected == FunctionTypeDateTime {
		return actual == FunctionTypeDate || actual == FunctionTypeTimestamp
	}
	return actual == FunctionTypeDateTime && (expected == FunctionTypeDate || expected == FunctionTypeTimestamp)
}

func functionTypesOverlap(actual, expected []FunctionType) bool {
	if len(actual) == 0 || len(expected) == 0 {
		return true
	}
	for _, exp := range expected {
		for _, act := range actual {
			if functionTypeCompatible(exp, act) {
				return true
			}
		}
	}
	return false
}

func functionCallReturns(call *FunctionCall, expected FunctionType) bool {
	return functionTypesOverlap(call.ReturnTypes, []FunctionType{expected})
}

func isKnownFunctionType(typ FunctionType) bool {
	switch typ {
	case FunctionTypeAny, FunctionTypeString, FunctionTypeNumber, FunctionTypeInteger,
		FunctionTypeDate, FunctionTypeTimestamp, FunctionTypeDateTime, FunctionTypeInterval,
		FunctionTypeGeometry, FunctionTypeBoolean, FunctionTypeArray:
		return true
	default:
		return false
	}
}

func normalizeFunctionName(name string) string {
	return strings.ToLower(name)
}

func functionNames(defs []FunctionDefinition) []string {
	if len(defs) == 0 {
		return nil
	}
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		if def.Name != "" {
			names = append(names, normalizeFunctionName(def.Name))
		}
	}
	return names
}

func allowedAnyFunctions(names []string) []FunctionDefinition {
	defs := make([]FunctionDefinition, 0, len(names))
	for _, name := range names {
		name = normalizeFunctionName(name)
		if name == "" {
			continue
		}
		defs = append(defs, FunctionDefinition{
			Name:    name,
			Args:    []FunctionArgument{{Types: []FunctionType{FunctionTypeAny}, Variadic: true}},
			Returns: []FunctionType{FunctionTypeAny},
		})
	}
	return defs
}

func cloneFunctionDefinitions(defs map[string]FunctionDefinition) []FunctionDefinition {
	if len(defs) == 0 {
		return nil
	}
	out := make([]FunctionDefinition, 0, len(defs))
	for _, def := range defs {
		out = append(out, cloneFunctionDefinition(def))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func cloneFunctionDefinition(def FunctionDefinition) FunctionDefinition {
	out := FunctionDefinition{Name: normalizeFunctionName(def.Name), Returns: cloneFunctionTypes(def.Returns)}
	if len(def.Args) > 0 {
		out.Args = make([]FunctionArgument, len(def.Args))
		for i, arg := range def.Args {
			out.Args[i] = FunctionArgument{Name: arg.Name, Types: cloneFunctionTypes(arg.Types), Variadic: arg.Variadic}
		}
	}
	return out
}

func cloneFunctionTypes(values []FunctionType) []FunctionType {
	if len(values) == 0 {
		return nil
	}
	out := make([]FunctionType, len(values))
	copy(out, values)
	return out
}

func describeFunctionTypes(types []FunctionType) string {
	if len(types) == 0 {
		return "any"
	}
	parts := make([]string, len(types))
	for i, typ := range types {
		if typ == FunctionTypeAny {
			parts[i] = "any"
			continue
		}
		parts[i] = string(typ)
	}
	return strings.Join(parts, " or ")
}

func functionReturnPropertyType(call *FunctionCall) PropertyType {
	if len(call.ReturnTypes) != 1 {
		return PropertyTypeAny
	}
	switch call.ReturnTypes[0] {
	case FunctionTypeString:
		return PropertyTypeString
	case FunctionTypeNumber:
		return PropertyTypeNumber
	case FunctionTypeInteger:
		return PropertyTypeInteger
	case FunctionTypeBoolean:
		return PropertyTypeBoolean
	case FunctionTypeDate:
		return PropertyTypeDate
	case FunctionTypeTimestamp, FunctionTypeDateTime:
		return PropertyTypeTimestamp
	case FunctionTypeInterval:
		return PropertyTypeInterval
	case FunctionTypeGeometry:
		return PropertyTypeGeometry
	case FunctionTypeArray:
		return PropertyTypeArray
	default:
		return PropertyTypeAny
	}
}
