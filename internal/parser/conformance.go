package parser

import "github.com/cwygoda/cql2/api"

// WithConformance records CQL2 conformance classes and configures the standard
// functions required by those classes. Arguments may be api conformance
// constants, full CQL2 conformance/requirements URIs, /conf/<class> fragments,
// or class slugs such as "case-insensitive-comparison".
//
// The Functions conformance class does not define any concrete function names;
// combine it with WithAllowedFunctions or WithSupportedFunctions to advertise
// implementation-specific functions.
func WithConformance(classes ...string) ParseOption {
	return func(p *Parser) {
		canonical := api.CanonicalConformanceClasses(classes...)
		p.conformanceClasses = canonical
		p.cfg.conformance = conformanceCapabilitiesForClasses(canonical...)
		defs := mergeFunctionDefinitions(cloneFunctionDefinitions(p.cfg.functions.defs), api.StandardFunctionsForConformance(canonical...))
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
	for _, class := range api.CanonicalConformanceClasses(classes...) {
		switch class {
		case api.ConformanceAdvancedComparisonOperators:
			caps.advancedComparisonOperators = true
		case api.ConformanceBasicSpatialFunctions:
			caps.basicSpatialFunctions = true
		case api.ConformanceBasicSpatialFunctionsPlus:
			caps.basicSpatialFunctionsPlus = true
		case api.ConformanceSpatialFunctions:
			caps.spatialFunctions = true
		case api.ConformanceTemporalFunctions:
			caps.temporalFunctions = true
		case api.ConformanceArrayFunctions:
			caps.arrayFunctions = true
		case api.ConformancePropertyProperty:
			caps.propertyProperty = true
		case api.ConformanceArithmetic:
			caps.arithmetic = true
		}
	}
	return caps
}

func (c conformanceCapabilities) allowsSpatialPredicate(op api.SpatialPredicateOp) bool {
	if c.spatialFunctions {
		return true
	}
	return (c.basicSpatialFunctions || c.basicSpatialFunctionsPlus) && op == api.SpatialOpIntersects
}

func (c conformanceCapabilities) allowsTemporalPredicate(api.TemporalPredicateOp) bool {
	return c.temporalFunctions
}

func (c conformanceCapabilities) allowsArrayPredicate(api.ArrayPredicateOp) bool {
	return c.arrayFunctions
}
