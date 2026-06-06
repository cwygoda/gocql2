package gocql2

import (
	"github.com/cwygoda/gocql2/api"
	"github.com/cwygoda/gocql2/internal/parser"
)

// Parser parses CQL2 and exposes the capabilities it was configured with.
type Parser struct {
	inner *parser.Parser
}

// NewParser builds a reusable parser. Chain setup methods before concurrent use.
func NewParser() *Parser {
	return &Parser{inner: parser.NewParser()}
}

// WithMaxDepth limits recursive parse depth.
func (p *Parser) WithMaxDepth(n int) *Parser {
	p.inner.WithMaxDepth(n)
	return p
}

// WithAllowedProperties configures a fail-closed property registry. Any property
// reference not present in the registry is rejected, and registered types are
// used to validate character, numeric, comparison, and IN-list contexts.
func (p *Parser) WithAllowedProperties(defs ...api.PropertyDefinition) *Parser {
	p.inner.WithAllowedProperties(defs...)
	return p
}

// WithAllowedFunctions adds function definitions to the fail-closed function
// registry. Any function reference not present in the registry is rejected, and
// registered signatures are used to validate argument counts, argument types,
// and return-type contexts. Definitions added later override earlier
// definitions with the same normalized name.
func (p *Parser) WithAllowedFunctions(defs ...api.FunctionDefinition) *Parser {
	p.inner.WithAllowedFunctions(defs...)
	return p
}

// WithConformanceClasses records the parser's advertised conformance classes.
func (p *Parser) WithConformanceClasses(classes ...string) *Parser {
	p.inner.WithConformanceClasses(classes...)
	return p
}

// WithConformance records CQL2 conformance classes and configures the standard
// functions required by those classes. Arguments may be api conformance
// constants, full CQL2 conformance/requirements URIs, /conf/<class> fragments,
// or class slugs such as "case-insensitive-comparison".
//
// The Functions conformance class does not define any concrete function names;
// combine it with WithAllowedFunctions to advertise implementation-specific
// functions.
func (p *Parser) WithConformance(classes ...string) *Parser {
	p.inner.WithConformance(classes...)
	return p
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
