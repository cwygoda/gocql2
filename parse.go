package gocql2

import (
	"fmt"
	"sort"
)

const defaultMaxDepth = 128

// ParseConfig configures parser behavior.
type ParseConfig struct {
	properties propertyRegistry
	functions  functionRegistry

	MaxDepth int
}

// Parser parses CQL2 and exposes the capabilities it was configured with.
type Parser struct {
	supportedProperties []string
	supportedFunctions  []string
	conformanceClasses  []string
	cfg                 ParseConfig
}

// ParseOption mutates Parser configuration.
type ParseOption func(*Parser)

// NewParser builds a reusable parser.
func NewParser(opts ...ParseOption) *Parser {
	p := &Parser{
		cfg: ParseConfig{MaxDepth: defaultMaxDepth, functions: functionRegistryDefaults()},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}
	if p.cfg.MaxDepth <= 0 {
		p.cfg.MaxDepth = defaultMaxDepth
	}
	if !p.cfg.functions.initialized {
		p.cfg.functions = functionRegistryDefaults()
	}
	return p
}

// WithMaxDepth limits recursive parse depth.
func WithMaxDepth(n int) ParseOption {
	return func(p *Parser) { p.cfg.MaxDepth = n }
}

// WithSupportedProperties records the parser's advertised property set and
// restricts parsing to that allow-list. Properties are treated as untyped; use
// WithAllowedProperties when type validation is needed.
func WithSupportedProperties(names ...string) ParseOption {
	return func(p *Parser) {
		p.supportedProperties = cloneStrings(names)
		defs := make([]PropertyDefinition, 0, len(names))
		for _, name := range names {
			defs = append(defs, PropertyDefinition{Name: name, Type: PropertyTypeAny})
		}
		p.cfg.properties = newPropertyRegistry(defs, true)
	}
}

// WithAllowedProperties configures a fail-closed property registry. Any property
// reference not present in the registry is rejected, and registered types are
// used to validate character, numeric, comparison, and IN-list contexts.
func WithAllowedProperties(defs ...PropertyDefinition) ParseOption {
	return func(p *Parser) {
		p.supportedProperties = propertyNames(defs)
		p.cfg.properties = newPropertyRegistry(defs, true)
	}
}

// WithSupportedFunctions configures a fail-closed name-only function registry.
// Registered functions accept any number of arguments of any type and have an
// unknown return type. Use WithAllowedFunctions when signature validation is
// needed.
func WithSupportedFunctions(names ...string) ParseOption {
	return func(p *Parser) {
		p.supportedFunctions = functionNames(allowedAnyFunctions(names))
		p.cfg.functions = newFunctionRegistry(allowedAnyFunctions(names))
	}
}

// WithAllowedFunctions configures a fail-closed function registry. Any function
// reference not present in the registry is rejected, and registered signatures
// are used to validate argument counts, argument types, and return-type contexts.
func WithAllowedFunctions(defs ...FunctionDefinition) ParseOption {
	return func(p *Parser) {
		p.supportedFunctions = functionNames(defs)
		p.cfg.functions = newFunctionRegistry(defs)
	}
}

// WithConformanceClasses records the parser's advertised conformance classes.
func WithConformanceClasses(classes ...string) ParseOption {
	return func(p *Parser) { p.conformanceClasses = cloneStrings(classes) }
}

// SupportedProperties returns the advertised property names.
func (p *Parser) SupportedProperties() []string {
	return cloneStrings(p.supportedProperties)
}

// SupportedPropertyDefinitions returns the configured allowed properties.
func (p *Parser) SupportedPropertyDefinitions() []PropertyDefinition {
	return clonePropertyDefinitions(p.cfg.properties.defs)
}

// SupportedFunctions returns the advertised function names.
func (p *Parser) SupportedFunctions() []string {
	return cloneStrings(p.supportedFunctions)
}

// SupportedFunctionDefinitions returns the configured allowed functions.
func (p *Parser) SupportedFunctionDefinitions() []FunctionDefinition {
	return cloneFunctionDefinitions(p.cfg.functions.defs)
}

// ConformanceClasses returns the advertised conformance class IDs.
func (p *Parser) ConformanceClasses() []string {
	return cloneStrings(p.conformanceClasses)
}

// Parse parses input in the requested CQL2 language.
func (p *Parser) Parse(input []byte, lang Language) (Expression, error) {
	switch lang {
	case LanguageText:
		return p.ParseText(string(input))
	case LanguageJSON:
		return p.ParseJSON(input)
	default:
		return nil, fmt.Errorf("unsupported CQL2 language %q", lang)
	}
}

// ParseText parses CQL2 Text into an AST.
func (p *Parser) ParseText(input string) (Expression, error) {
	return parseText(input, p.cfg)
}

// ParseJSON parses CQL2 JSON into an AST.
func (p *Parser) ParseJSON(input []byte) (Expression, error) {
	return parseJSON(input, p.cfg)
}

// Parse parses input in the requested CQL2 language.
func Parse(input []byte, lang Language, opts ...ParseOption) (Expression, error) {
	return NewParser(opts...).Parse(input, lang)
}

// ParseText parses CQL2 Text into an AST.
func ParseText(input string, opts ...ParseOption) (Expression, error) {
	return NewParser(opts...).ParseText(input)
}

// ParseJSON parses CQL2 JSON into an AST.
func ParseJSON(input []byte, opts ...ParseOption) (Expression, error) {
	return NewParser(opts...).ParseJSON(input)
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

func propertyNames(defs []PropertyDefinition) []string {
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

func clonePropertyDefinitions(defs map[string]PropertyDefinition) []PropertyDefinition {
	if len(defs) == 0 {
		return nil
	}
	out := make([]PropertyDefinition, 0, len(defs))
	for _, def := range defs {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
