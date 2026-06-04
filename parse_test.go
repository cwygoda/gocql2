package gocql2

import (
	"testing"

	"github.com/cwygoda/cql2/api"
)

func TestRootParserFacade(t *testing.T) {
	defs := []api.PropertyDefinition{{Name: "name", Type: api.PropertyTypeString}}
	fn := api.FunctionDefinition{
		Name:    "is_named",
		Args:    []api.FunctionArgument{{Name: "value", Types: []api.FunctionType{api.FunctionTypeString}}},
		Returns: []api.FunctionType{api.FunctionTypeBoolean},
	}
	p := NewParser().WithMaxDepth(8).WithAllowedProperties(defs...).WithSupportedProperties("ignored").WithSupportedFunctions("legacy_fn").WithAllowedFunctions(fn).WithConformance(api.ConformanceCaseInsensitiveComparison).WithConformanceClasses("manual")

	if got := p.SupportedProperties(); len(got) != 1 || got[0] != "ignored" {
		t.Fatalf("SupportedProperties = %#v", got)
	}
	if got := p.SupportedPropertyDefinitions(); len(got) != 1 || got[0].Name != "ignored" {
		t.Fatalf("SupportedPropertyDefinitions = %#v", got)
	}
	if got := p.SupportedFunctions(); len(got) == 0 {
		t.Fatal("SupportedFunctions should not be empty")
	}
	if got := p.SupportedFunctionDefinitions(); len(got) == 0 {
		t.Fatal("SupportedFunctionDefinitions should not be empty")
	}
	if got := p.ConformanceClasses(); len(got) != 1 || got[0] != "manual" {
		t.Fatalf("ConformanceClasses = %#v", got)
	}

	if _, err := p.ParseText("ignored = 'x'"); err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if _, err := p.ParseJSON([]byte(`{"op":"=","args":[{"property":"ignored"},"x"]}`)); err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if _, err := p.Parse([]byte("ignored = 'x'"), api.LanguageText); err != nil {
		t.Fatalf("Parse text dispatch: %v", err)
	}
	if _, err := p.Parse([]byte(`{"op":"=","args":[{"property":"ignored"},"x"]}`), api.LanguageJSON); err != nil {
		t.Fatalf("Parse JSON dispatch: %v", err)
	}
	if _, err := p.Parse([]byte("ignored = 'x'"), api.Language("unknown")); err == nil {
		t.Fatal("Parse unsupported language succeeded")
	}
}

func TestRootParseHelpers(t *testing.T) {
	if _, err := ParseText("name = 'x'"); err != nil {
		t.Fatalf("ParseText helper: %v", err)
	}
	if _, err := ParseJSON([]byte(`{"op":"=","args":[{"property":"name"},"x"]}`)); err != nil {
		t.Fatalf("ParseJSON helper: %v", err)
	}
	if _, err := Parse([]byte("name = 'x'"), api.LanguageText); err != nil {
		t.Fatalf("Parse helper: %v", err)
	}
}
