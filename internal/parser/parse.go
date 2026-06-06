package parser

import (
	"fmt"
	"sort"

	"github.com/cwygoda/gocql2/api"
)

const defaultMaxDepth = 128

// ParseConfig configures parser behavior.
type ParseConfig struct {
	properties  propertyRegistry
	functions   functionRegistry
	conformance conformanceCapabilities

	MaxDepth int
}

// Parser parses CQL2 and exposes the capabilities it was configured with.
type Parser struct {
	supportedProperties []string
	supportedFunctions  []string
	conformanceClasses  []string
	cfg                 ParseConfig
}

// NewParser builds a reusable parser. Chain setup methods before concurrent use.
func NewParser() *Parser {
	return &Parser{
		cfg: ParseConfig{MaxDepth: defaultMaxDepth, functions: functionRegistryDefaults()},
	}
}

// WithMaxDepth limits recursive parse depth.
func (p *Parser) WithMaxDepth(n int) *Parser {
	p.cfg.MaxDepth = n
	if p.cfg.MaxDepth <= 0 {
		p.cfg.MaxDepth = defaultMaxDepth
	}
	return p
}

// WithAllowedProperties configures a fail-closed property registry. Any property
// reference not present in the registry is rejected, and registered types are
// used to validate character, numeric, comparison, and IN-list contexts.
func (p *Parser) WithAllowedProperties(defs ...api.PropertyDefinition) *Parser {
	p.supportedProperties = propertyNames(defs)
	p.cfg.properties = newPropertyRegistry(defs, true)
	return p
}

// WithAllowedFunctions adds function definitions to the fail-closed function
// registry. Any function reference not present in the registry is rejected, and
// registered signatures are used to validate argument counts, argument types,
// and return-type contexts. Definitions added later override earlier
// definitions with the same normalized name.
func (p *Parser) WithAllowedFunctions(defs ...api.FunctionDefinition) *Parser {
	merged := mergeFunctionDefinitions(cloneFunctionDefinitions(p.cfg.functions.defs), defs)
	p.supportedFunctions = functionNames(merged)
	p.cfg.functions = newFunctionRegistry(merged)
	return p
}

// WithConformanceClasses records the parser's advertised conformance classes.
func (p *Parser) WithConformanceClasses(classes ...string) *Parser {
	p.conformanceClasses = cloneStrings(classes)
	return p
}

// SupportedProperties returns the advertised property names.
func (p *Parser) SupportedProperties() []string {
	return cloneStrings(p.supportedProperties)
}

// SupportedPropertyDefinitions returns the configured allowed properties.
func (p *Parser) SupportedPropertyDefinitions() []api.PropertyDefinition {
	return clonePropertyDefinitions(p.cfg.properties.defs)
}

// SupportedFunctions returns the advertised function names.
func (p *Parser) SupportedFunctions() []string {
	return cloneStrings(p.supportedFunctions)
}

// SupportedFunctionDefinitions returns the configured allowed functions.
func (p *Parser) SupportedFunctionDefinitions() []api.FunctionDefinition {
	return cloneFunctionDefinitions(p.cfg.functions.defs)
}

// ConformanceClasses returns the advertised conformance class IDs.
func (p *Parser) ConformanceClasses() []string {
	return cloneStrings(p.conformanceClasses)
}

// Parse parses input in the requested CQL2 language.
func (p *Parser) Parse(input []byte, lang api.Language) (api.Expression, error) {
	switch lang {
	case api.LanguageText:
		return p.ParseText(string(input))
	case api.LanguageJSON:
		return p.ParseJSON(input)
	default:
		return nil, fmt.Errorf("unsupported CQL2 language %q", lang)
	}
}

// ParseText parses CQL2 Text into an AST.
func (p *Parser) ParseText(input string) (api.Expression, error) {
	return parseText(input, p.cfg)
}

// ParseJSON parses CQL2 JSON into an AST.
func (p *Parser) ParseJSON(input []byte) (api.Expression, error) {
	return parseJSON(input, p.cfg)
}

func applyParseConfigDefaults(cfg ParseConfig) ParseConfig {
	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = defaultMaxDepth
	}
	if !cfg.functions.initialized {
		cfg.functions = functionRegistryDefaults()
	}
	return cfg
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func propertyNames(defs []api.PropertyDefinition) []string {
	if len(defs) == 0 {
		return nil
	}
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		if def.Name != "" {
			names = append(names, def.Name)
		}
	}
	return names
}

func clonePropertyDefinitions(defs map[string]api.PropertyDefinition) []api.PropertyDefinition {
	if len(defs) == 0 {
		return nil
	}
	out := make([]api.PropertyDefinition, 0, len(defs))
	for _, def := range defs {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
