package gocql2

import "strings"

const (
	conformanceURIBase  = "http://www.opengis.net/spec/cql2/1.0/conf/"
	requirementsURIBase = "http://www.opengis.net/spec/cql2/1.0/req/"
)

// CQL2 1.0 conformance class URIs.
const (
	ConformanceBasicCQL2                   = conformanceURIBase + "basic-cql2"
	ConformanceAdvancedComparisonOperators = conformanceURIBase + "advanced-comparison-operators"
	ConformanceCaseInsensitiveComparison   = conformanceURIBase + "case-insensitive-comparison"
	ConformanceAccentInsensitiveComparison = conformanceURIBase + "accent-insensitive-comparison"
	ConformanceBasicSpatialFunctions       = conformanceURIBase + "basic-spatial-functions"
	ConformanceBasicSpatialFunctionsPlus   = conformanceURIBase + "basic-spatial-functions-plus"
	ConformanceSpatialFunctions            = conformanceURIBase + "spatial-functions"
	ConformanceTemporalFunctions           = conformanceURIBase + "temporal-functions"
	ConformanceArrayFunctions              = conformanceURIBase + "array-functions"
	ConformancePropertyProperty            = conformanceURIBase + "property-property"
	ConformanceFunctions                   = conformanceURIBase + "functions"
	ConformanceArithmetic                  = conformanceURIBase + "arithmetic"
	ConformanceCQL2Text                    = conformanceURIBase + "cql2-text"
	ConformanceCQL2JSON                    = conformanceURIBase + "cql2-json"
)

var conformanceBySlug = map[string]string{
	"basic-cql2":                    ConformanceBasicCQL2,
	"advanced-comparison-operators": ConformanceAdvancedComparisonOperators,
	"case-insensitive-comparison":   ConformanceCaseInsensitiveComparison,
	"accent-insensitive-comparison": ConformanceAccentInsensitiveComparison,
	"basic-spatial-functions":       ConformanceBasicSpatialFunctions,
	"basic-spatial-functions-plus":  ConformanceBasicSpatialFunctionsPlus,
	"spatial-functions":             ConformanceSpatialFunctions,
	"temporal-functions":            ConformanceTemporalFunctions,
	"array-functions":               ConformanceArrayFunctions,
	"property-property":             ConformancePropertyProperty,
	"functions":                     ConformanceFunctions,
	"arithmetic":                    ConformanceArithmetic,
	"cql2-text":                     ConformanceCQL2Text,
	"cql2-json":                     ConformanceCQL2JSON,
}

// WithConformance records CQL2 conformance classes and configures the standard
// functions required by those classes. Arguments may be the constants above,
// full CQL2 conformance/requirements URIs, /conf/<class> fragments, or class
// slugs such as "case-insensitive-comparison".
//
// The Functions conformance class does not define any concrete function names;
// combine it with WithAllowedFunctions or WithSupportedFunctions to advertise
// implementation-specific functions.
func WithConformance(classes ...string) ParseOption {
	return func(p *Parser) {
		canonical := canonicalConformanceClasses(classes)
		p.conformanceClasses = canonical
		p.cfg.conformance = conformanceCapabilitiesForClasses(canonical...)
		defs := mergeFunctionDefinitions(cloneFunctionDefinitions(p.cfg.functions.defs), StandardFunctionsForConformance(canonical...))
		p.supportedFunctions = functionNames(defs)
		p.cfg.functions = newFunctionRegistry(defs)
	}
}

type conformanceCapabilities struct {
	advancedComparisonOperators bool
	basicSpatialFunctions       bool
	basicSpatialFunctionsPlus   bool
	spatialFunctions            bool
	temporalFunctions           bool
	arrayFunctions              bool
	propertyProperty            bool
	arithmetic                  bool
}

func conformanceCapabilitiesForClasses(classes ...string) conformanceCapabilities {
	var caps conformanceCapabilities
	for _, class := range canonicalConformanceClasses(classes) {
		switch class {
		case ConformanceAdvancedComparisonOperators:
			caps.advancedComparisonOperators = true
		case ConformanceBasicSpatialFunctions:
			caps.basicSpatialFunctions = true
		case ConformanceBasicSpatialFunctionsPlus:
			caps.basicSpatialFunctionsPlus = true
		case ConformanceSpatialFunctions:
			caps.spatialFunctions = true
		case ConformanceTemporalFunctions:
			caps.temporalFunctions = true
		case ConformanceArrayFunctions:
			caps.arrayFunctions = true
		case ConformancePropertyProperty:
			caps.propertyProperty = true
		case ConformanceArithmetic:
			caps.arithmetic = true
		}
	}
	return caps
}

func (c conformanceCapabilities) allowsSpatialPredicate(op SpatialPredicateOp) bool {
	if c.spatialFunctions {
		return true
	}
	return (c.basicSpatialFunctions || c.basicSpatialFunctionsPlus) && op == SpatialOpIntersects
}

func (c conformanceCapabilities) allowsTemporalPredicate(TemporalPredicateOp) bool {
	return c.temporalFunctions
}

func (c conformanceCapabilities) allowsArrayPredicate(ArrayPredicateOp) bool {
	return c.arrayFunctions
}

// StandardFunctionsForConformance returns the CQL2-standard function
// definitions implied by the provided conformance classes.
func StandardFunctionsForConformance(classes ...string) []FunctionDefinition {
	defs := []FunctionDefinition{}
	for _, class := range canonicalConformanceClasses(classes) {
		switch class {
		case ConformanceCaseInsensitiveComparison:
			defs = append(defs, CaseIFunction())
		case ConformanceAccentInsensitiveComparison:
			defs = append(defs, AccentiFunction())
		case ConformanceBasicSpatialFunctions, ConformanceBasicSpatialFunctionsPlus:
			defs = append(defs, spatialPredicateFunction(SpatialOpIntersects))
		case ConformanceSpatialFunctions:
			defs = append(defs, standardSpatialFunctionDefinitions()...)
		case ConformanceTemporalFunctions:
			defs = append(defs, standardTemporalFunctionDefinitions()...)
		case ConformanceArrayFunctions:
			defs = append(defs, standardArrayFunctionDefinitions()...)
		}
	}
	return mergeFunctionDefinitions(nil, defs)
}

func canonicalConformanceClasses(classes []string) []string {
	if len(classes) == 0 {
		return nil
	}
	out := make([]string, 0, len(classes))
	seen := map[string]struct{}{}
	for _, class := range classes {
		canonical := canonicalConformanceClass(class)
		if canonical == "" {
			continue
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	return out
}

func canonicalConformanceClass(class string) string {
	value := strings.TrimSpace(class)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	for _, prefix := range []string{conformanceURIBase, requirementsURIBase} {
		if strings.HasPrefix(lower, prefix) {
			slug := strings.TrimPrefix(lower, prefix)
			if canonical, ok := conformanceBySlug[slug]; ok {
				return canonical
			}
			return value
		}
	}
	if strings.HasPrefix(lower, "/conf/") || strings.HasPrefix(lower, "/req/") {
		parts := strings.Split(strings.Trim(lower, "/"), "/")
		if len(parts) >= 2 {
			if canonical, ok := conformanceBySlug[parts[1]]; ok {
				return canonical
			}
		}
		return value
	}
	if canonical, ok := conformanceBySlug[lower]; ok {
		return canonical
	}
	return value
}

func standardSpatialFunctionDefinitions() []FunctionDefinition {
	return []FunctionDefinition{
		spatialPredicateFunction(SpatialOpIntersects),
		spatialPredicateFunction(SpatialOpContains),
		spatialPredicateFunction(SpatialOpCrosses),
		spatialPredicateFunction(SpatialOpDisjoint),
		spatialPredicateFunction(SpatialOpEquals),
		spatialPredicateFunction(SpatialOpOverlaps),
		spatialPredicateFunction(SpatialOpTouches),
		spatialPredicateFunction(SpatialOpWithin),
	}
}

func spatialPredicateFunction(op SpatialPredicateOp) FunctionDefinition {
	return FunctionDefinition{
		Name: string(op),
		Args: []FunctionArgument{
			{Name: "left", Types: []FunctionType{FunctionTypeGeometry}},
			{Name: "right", Types: []FunctionType{FunctionTypeGeometry}},
		},
		Returns: []FunctionType{FunctionTypeBoolean},
	}
}

func standardTemporalFunctionDefinitions() []FunctionDefinition {
	return []FunctionDefinition{
		temporalPredicateFunction(TemporalOpAfter),
		temporalPredicateFunction(TemporalOpBefore),
		temporalPredicateFunction(TemporalOpContains),
		temporalPredicateFunction(TemporalOpDisjoint),
		temporalPredicateFunction(TemporalOpDuring),
		temporalPredicateFunction(TemporalOpEquals),
		temporalPredicateFunction(TemporalOpFinishedBy),
		temporalPredicateFunction(TemporalOpFinishes),
		temporalPredicateFunction(TemporalOpIntersects),
		temporalPredicateFunction(TemporalOpMeets),
		temporalPredicateFunction(TemporalOpMetBy),
		temporalPredicateFunction(TemporalOpOverlappedBy),
		temporalPredicateFunction(TemporalOpOverlaps),
		temporalPredicateFunction(TemporalOpStartedBy),
		temporalPredicateFunction(TemporalOpStarts),
	}
}

func temporalPredicateFunction(op TemporalPredicateOp) FunctionDefinition {
	types := []FunctionType{FunctionTypeDateTime, FunctionTypeInterval}
	if isIntervalOnlyTemporalPredicate(op) {
		types = []FunctionType{FunctionTypeInterval}
	}
	return FunctionDefinition{
		Name: string(op),
		Args: []FunctionArgument{
			{Name: "left", Types: types},
			{Name: "right", Types: types},
		},
		Returns: []FunctionType{FunctionTypeBoolean},
	}
}

func standardArrayFunctionDefinitions() []FunctionDefinition {
	return []FunctionDefinition{
		arrayPredicateFunction(ArrayOpContainedBy),
		arrayPredicateFunction(ArrayOpContains),
		arrayPredicateFunction(ArrayOpEquals),
		arrayPredicateFunction(ArrayOpOverlaps),
	}
}

func arrayPredicateFunction(op ArrayPredicateOp) FunctionDefinition {
	return FunctionDefinition{
		Name: string(op),
		Args: []FunctionArgument{
			{Name: "left", Types: []FunctionType{FunctionTypeArray}},
			{Name: "right", Types: []FunctionType{FunctionTypeArray}},
		},
		Returns: []FunctionType{FunctionTypeBoolean},
	}
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
