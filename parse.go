package gocql2

import "fmt"

const defaultMaxDepth = 128

// ParseConfig configures parser behavior.
type ParseConfig struct {
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
	p := &Parser{cfg: ParseConfig{MaxDepth: defaultMaxDepth}}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}
	if p.cfg.MaxDepth <= 0 {
		p.cfg.MaxDepth = defaultMaxDepth
	}
	return p
}

// WithMaxDepth limits recursive parse depth.
func WithMaxDepth(n int) ParseOption {
	return func(p *Parser) { p.cfg.MaxDepth = n }
}

// WithSupportedProperties records the parser's advertised property set.
func WithSupportedProperties(names ...string) ParseOption {
	return func(p *Parser) { p.supportedProperties = cloneStrings(names) }
}

// WithSupportedFunctions records the parser's advertised function set.
func WithSupportedFunctions(names ...string) ParseOption {
	return func(p *Parser) { p.supportedFunctions = cloneStrings(names) }
}

// WithConformanceClasses records the parser's advertised conformance classes.
func WithConformanceClasses(classes ...string) ParseOption {
	return func(p *Parser) { p.conformanceClasses = cloneStrings(classes) }
}

// SupportedProperties returns the advertised property names.
func (p *Parser) SupportedProperties() []string {
	return cloneStrings(p.supportedProperties)
}

// SupportedFunctions returns the advertised function names.
func (p *Parser) SupportedFunctions() []string {
	return cloneStrings(p.supportedFunctions)
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

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}
