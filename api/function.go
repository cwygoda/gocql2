package api

import (
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

func normalizeFunctionName(name string) string {
	return strings.ToLower(name)
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

func mergeFunctionDefinitions(base, extra []FunctionDefinition) []FunctionDefinition {
	defs := map[string]FunctionDefinition{}
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
