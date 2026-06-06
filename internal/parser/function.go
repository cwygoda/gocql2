package parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cwygoda/gocql2/api"
)

const functionTypeUnsupported api.FunctionType = "\x00unsupported"

type functionRegistry struct {
	defs        map[string]api.FunctionDefinition
	initialized bool
}

func newFunctionRegistry(defs []api.FunctionDefinition) functionRegistry {
	registry := functionRegistry{initialized: true}
	if len(defs) == 0 {
		return registry
	}
	registry.defs = make(map[string]api.FunctionDefinition, len(defs))
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
	return newFunctionRegistry(nil)
}

func (r functionRegistry) lookup(name string) (api.FunctionDefinition, bool) {
	name = normalizeFunctionName(name)
	if !r.initialized {
		r = functionRegistryDefaults()
	}
	def, ok := r.defs[name]
	return def, ok
}

func validateFunctionCall(name string, args []api.Node, cfg ParseConfig, source api.Language, loc api.Location) (api.FunctionDefinition, error) {
	name = normalizeFunctionName(name)
	def, ok := cfg.functions.lookup(name)
	if !ok {
		return api.FunctionDefinition{}, parseError(source, loc, fmt.Sprintf("function %q is not allowed", name))
	}
	if err := validateFunctionDefinition(def, source, loc); err != nil {
		return api.FunctionDefinition{}, err
	}

	minArgs, maxArgs := functionArgumentBounds(def.Args)
	if len(args) < minArgs || maxArgs >= 0 && len(args) > maxArgs {
		return api.FunctionDefinition{}, parseError(source, loc, functionArgumentCountMessage(name, minArgs, maxArgs))
	}

	for i, arg := range args {
		spec := functionArgumentAt(def.Args, i)
		if len(spec.Types) == 0 {
			continue
		}
		actual := nodeFunctionTypes(arg)
		if !functionTypesOverlap(actual, spec.Types) {
			return api.FunctionDefinition{}, parseError(source, arg.Span().Start, fmt.Sprintf("argument %d to function %q has type %s, expected %s", i+1, name, describeFunctionTypes(actual), describeFunctionTypes(spec.Types)))
		}
	}
	return def, nil
}

func validateFunctionDefinition(def api.FunctionDefinition, source api.Language, loc api.Location) error {
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

func functionArgumentBounds(args []api.FunctionArgument) (int, int) {
	if len(args) == 0 {
		return 0, 0
	}
	if args[len(args)-1].Variadic {
		return len(args) - 1, -1
	}
	return len(args), len(args)
}

func functionArgumentAt(args []api.FunctionArgument, index int) api.FunctionArgument {
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

func nodeFunctionTypes(node api.Node) []api.FunctionType {
	switch value := node.(type) {
	case *api.Literal:
		switch value.Kind {
		case api.LiteralString:
			return []api.FunctionType{api.FunctionTypeString}
		case api.LiteralNumber:
			return []api.FunctionType{api.FunctionTypeNumber}
		case api.LiteralBool:
			return []api.FunctionType{api.FunctionTypeBoolean}
		default:
			return []api.FunctionType{functionTypeUnsupported}
		}
	case *api.PropertyRef:
		return []api.FunctionType{propertyFunctionType(value.Type)}
	case *api.ArithmeticExpression:
		return []api.FunctionType{api.FunctionTypeNumber}
	case *api.TemporalInstant:
		if value.Kind == api.TemporalInstantDate {
			return []api.FunctionType{api.FunctionTypeDate}
		}
		return []api.FunctionType{api.FunctionTypeTimestamp}
	case *api.TemporalInterval:
		return []api.FunctionType{api.FunctionTypeInterval}
	case *api.FunctionCall:
		return functionCallReturnTypes(value)
	case *api.ArrayLiteral:
		return []api.FunctionType{api.FunctionTypeArray}
	case *api.GeometryLiteral:
		return []api.FunctionType{api.FunctionTypeGeometry}
	case api.Expression:
		return []api.FunctionType{api.FunctionTypeBoolean}
	default:
		return []api.FunctionType{functionTypeUnsupported}
	}
}

func propertyFunctionType(typ api.PropertyType) api.FunctionType {
	switch typ {
	case api.PropertyTypeAny:
		return api.FunctionTypeAny
	case api.PropertyTypeString:
		return api.FunctionTypeString
	case api.PropertyTypeNumber:
		return api.FunctionTypeNumber
	case api.PropertyTypeInteger:
		return api.FunctionTypeInteger
	case api.PropertyTypeBoolean:
		return api.FunctionTypeBoolean
	case api.PropertyTypeDate:
		return api.FunctionTypeDate
	case api.PropertyTypeTimestamp:
		return api.FunctionTypeTimestamp
	case api.PropertyTypeInterval:
		return api.FunctionTypeInterval
	case api.PropertyTypePoint, api.PropertyTypeMultiPoint, api.PropertyTypeLineString, api.PropertyTypeMultiLineString,
		api.PropertyTypePolygon, api.PropertyTypeMultiPolygon, api.PropertyTypeGeometry, api.PropertyTypeGeometryCollection:
		return api.FunctionTypeGeometry
	case api.PropertyTypeArray:
		return api.FunctionTypeArray
	default:
		return functionTypeUnsupported
	}
}

func functionTypeCompatible(expected, actual api.FunctionType) bool {
	if !isKnownFunctionType(expected) || !isKnownFunctionType(actual) {
		return false
	}
	if expected == api.FunctionTypeAny || actual == api.FunctionTypeAny || expected == actual {
		return true
	}
	if expected == api.FunctionTypeNumber && actual == api.FunctionTypeInteger {
		return true
	}
	if expected == api.FunctionTypeDateTime {
		return actual == api.FunctionTypeDate || actual == api.FunctionTypeTimestamp
	}
	return actual == api.FunctionTypeDateTime && (expected == api.FunctionTypeDate || expected == api.FunctionTypeTimestamp)
}

func functionTypesOverlap(actual, expected []api.FunctionType) bool {
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

func functionCallReturns(call *api.FunctionCall, expected api.FunctionType) bool {
	return functionTypesOverlap(functionCallReturnTypes(call), []api.FunctionType{expected})
}

func functionCallReturnTypes(call *api.FunctionCall) []api.FunctionType {
	if len(call.ReturnTypes) == 0 {
		return []api.FunctionType{functionTypeUnsupported}
	}
	out := cloneFunctionTypes(call.ReturnTypes)
	for i, typ := range out {
		if !isKnownFunctionType(typ) {
			out[i] = functionTypeUnsupported
		}
	}
	return out
}

func isKnownFunctionType(typ api.FunctionType) bool {
	switch typ {
	case api.FunctionTypeAny, api.FunctionTypeString, api.FunctionTypeNumber, api.FunctionTypeInteger,
		api.FunctionTypeDate, api.FunctionTypeTimestamp, api.FunctionTypeDateTime, api.FunctionTypeInterval,
		api.FunctionTypeGeometry, api.FunctionTypeBoolean, api.FunctionTypeArray:
		return true
	default:
		return false
	}
}

func normalizeFunctionName(name string) string {
	return strings.ToLower(name)
}

func functionNames(defs []api.FunctionDefinition) []string {
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

func cloneFunctionDefinitions(defs map[string]api.FunctionDefinition) []api.FunctionDefinition {
	if len(defs) == 0 {
		return nil
	}
	out := make([]api.FunctionDefinition, 0, len(defs))
	for _, def := range defs {
		out = append(out, cloneFunctionDefinition(def))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func cloneFunctionDefinition(def api.FunctionDefinition) api.FunctionDefinition {
	out := api.FunctionDefinition{Name: normalizeFunctionName(def.Name), Returns: cloneFunctionTypes(def.Returns)}
	if len(def.Args) > 0 {
		out.Args = make([]api.FunctionArgument, len(def.Args))
		for i, arg := range def.Args {
			out.Args[i] = api.FunctionArgument{Name: arg.Name, Types: cloneFunctionTypes(arg.Types), Variadic: arg.Variadic}
		}
	}
	return out
}

func cloneFunctionTypes(values []api.FunctionType) []api.FunctionType {
	if len(values) == 0 {
		return nil
	}
	out := make([]api.FunctionType, len(values))
	copy(out, values)
	return out
}

func describeFunctionTypes(types []api.FunctionType) string {
	if len(types) == 0 {
		return "any"
	}
	parts := make([]string, len(types))
	for i, typ := range types {
		if typ == api.FunctionTypeAny {
			parts[i] = "any"
			continue
		}
		if typ == functionTypeUnsupported {
			parts[i] = "unsupported"
			continue
		}
		parts[i] = string(typ)
	}
	return strings.Join(parts, " or ")
}

func functionReturnPropertyType(call *api.FunctionCall) api.PropertyType {
	returnTypes := functionCallReturnTypes(call)
	for _, typ := range returnTypes {
		if typ == functionTypeUnsupported {
			return propertyTypeUnsupported
		}
	}
	if len(returnTypes) != 1 {
		return api.PropertyTypeAny
	}
	switch returnTypes[0] {
	case api.FunctionTypeAny:
		return api.PropertyTypeAny
	case api.FunctionTypeString:
		return api.PropertyTypeString
	case api.FunctionTypeNumber:
		return api.PropertyTypeNumber
	case api.FunctionTypeInteger:
		return api.PropertyTypeInteger
	case api.FunctionTypeBoolean:
		return api.PropertyTypeBoolean
	case api.FunctionTypeDate:
		return api.PropertyTypeDate
	case api.FunctionTypeTimestamp, api.FunctionTypeDateTime:
		return api.PropertyTypeTimestamp
	case api.FunctionTypeInterval:
		return api.PropertyTypeInterval
	case api.FunctionTypeGeometry:
		return api.PropertyTypeGeometry
	case api.FunctionTypeArray:
		return api.PropertyTypeArray
	default:
		return propertyTypeUnsupported
	}
}

func mergeFunctionDefinitions(base, extra []api.FunctionDefinition) []api.FunctionDefinition {
	defs := map[string]api.FunctionDefinition{}
	for _, def := range base {
		def = cloneFunctionDefinition(def)
		if def.Name != "" {
			defs[def.Name] = def
		}
	}
	for _, def := range extra {
		def = cloneFunctionDefinition(def)
		if def.Name != "" {
			defs[def.Name] = def
		}
	}
	return cloneFunctionDefinitions(defs)
}
