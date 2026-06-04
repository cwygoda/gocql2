package gocql2

import (
	"github.com/cwygoda/cql2/api"
	"github.com/cwygoda/cql2/internal/parser"
)

// Parser parses CQL2 and exposes the capabilities it was configured with.
type Parser struct {
	inner *parser.Parser
}

// ParseOption configures parser behavior.
type ParseOption struct {
	inner parser.ParseOption
}

// NewParser builds a reusable parser.
func NewParser(opts ...ParseOption) *Parser {
	return &Parser{inner: parser.NewParser(parserOptions(opts)...)}
}

// WithMaxDepth limits recursive parse depth.
func WithMaxDepth(n int) ParseOption { return parseOption(parser.WithMaxDepth(n)) }

// WithSupportedProperties records the parser's advertised property set and
// restricts parsing to that allow-list. Properties are treated as untyped; use
// WithAllowedProperties when type validation is needed.
func WithSupportedProperties(names ...string) ParseOption {
	return parseOption(parser.WithSupportedProperties(names...))
}

// WithAllowedProperties configures a fail-closed property registry. Any property
// reference not present in the registry is rejected, and registered types are
// used to validate character, numeric, comparison, and IN-list contexts.
func WithAllowedProperties(defs ...api.PropertyDefinition) ParseOption {
	return parseOption(parser.WithAllowedProperties(defs...))
}

// WithSupportedFunctions adds names to the fail-closed name-only function
// registry. Registered functions accept any number of arguments of any type and
// have an unknown return type. Use WithAllowedFunctions when signature
// validation is needed.
func WithSupportedFunctions(names ...string) ParseOption {
	return parseOption(parser.WithSupportedFunctions(names...))
}

// WithAllowedFunctions adds function definitions to the fail-closed function
// registry. Any function reference not present in the registry is rejected, and
// registered signatures are used to validate argument counts, argument types,
// and return-type contexts. Definitions added later override earlier
// definitions with the same normalized name.
func WithAllowedFunctions(defs ...api.FunctionDefinition) ParseOption {
	return parseOption(parser.WithAllowedFunctions(defs...))
}

// WithConformanceClasses records the parser's advertised conformance classes.
func WithConformanceClasses(classes ...string) ParseOption {
	return parseOption(parser.WithConformanceClasses(classes...))
}

// WithConformance records CQL2 conformance classes and configures the standard
// functions required by those classes. Arguments may be api conformance
// constants, full CQL2 conformance/requirements URIs, /conf/<class> fragments,
// or class slugs such as "case-insensitive-comparison".
//
// The Functions conformance class does not define any concrete function names;
// combine it with WithAllowedFunctions or WithSupportedFunctions to advertise
// implementation-specific functions.
func WithConformance(classes ...string) ParseOption {
	return parseOption(parser.WithConformance(classes...))
}

// SupportedProperties returns the advertised property names.
func (p *Parser) SupportedProperties() []string { return p.inner.SupportedProperties() }

// SupportedPropertyDefinitions returns the configured allowed properties.
func (p *Parser) SupportedPropertyDefinitions() []api.PropertyDefinition {
	return p.inner.SupportedPropertyDefinitions()
}

// SupportedFunctions returns the advertised function names.
func (p *Parser) SupportedFunctions() []string { return p.inner.SupportedFunctions() }

// SupportedFunctionDefinitions returns the configured allowed functions.
func (p *Parser) SupportedFunctionDefinitions() []api.FunctionDefinition {
	return p.inner.SupportedFunctionDefinitions()
}

// ConformanceClasses returns the advertised conformance class IDs.
func (p *Parser) ConformanceClasses() []string { return p.inner.ConformanceClasses() }

// Parse parses input in the requested CQL2 language.
func (p *Parser) Parse(input []byte, lang api.Language) (api.Expression, error) {
	return p.inner.Parse(input, lang)
}

// ParseText parses CQL2 Text into an AST.
func (p *Parser) ParseText(input string) (api.Expression, error) { return p.inner.ParseText(input) }

// ParseJSON parses CQL2 JSON into an AST.
func (p *Parser) ParseJSON(input []byte) (api.Expression, error) { return p.inner.ParseJSON(input) }

// Parse parses input in the requested CQL2 language.
func Parse(input []byte, lang api.Language, opts ...ParseOption) (api.Expression, error) {
	return NewParser(opts...).Parse(input, lang)
}

// ParseText parses CQL2 Text into an AST.
func ParseText(input string, opts ...ParseOption) (api.Expression, error) {
	return NewParser(opts...).ParseText(input)
}

// ParseJSON parses CQL2 JSON into an AST.
func ParseJSON(input []byte, opts ...ParseOption) (api.Expression, error) {
	return NewParser(opts...).ParseJSON(input)
}

func parseOption(opt parser.ParseOption) ParseOption {
	return ParseOption{inner: opt}
}

func parserOptions(opts []ParseOption) []parser.ParseOption {
	out := make([]parser.ParseOption, 0, len(opts))
	for _, opt := range opts {
		if opt.inner != nil {
			out = append(out, opt.inner)
		}
	}
	return out
}
