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
	p := NewParser().WithMaxDepth(8).WithAllowedProperties(defs...).WithAllowedFunctions(fn).WithConformance(api.ConformanceCaseInsensitiveComparison).WithConformanceClasses("manual")

	if got := p.SupportedProperties(); len(got) != 1 || got[0] != "name" {
		t.Fatalf("SupportedProperties = %#v", got)
	}
	if got := p.SupportedPropertyDefinitions(); len(got) != 1 || got[0].Name != "name" {
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

	if _, err := p.ParseText("name = 'x'"); err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if _, err := p.ParseJSON([]byte(`{"op":"=","args":[{"property":"name"},"x"]}`)); err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if _, err := p.Parse([]byte("name = 'x'"), api.LanguageText); err != nil {
		t.Fatalf("Parse text dispatch: %v", err)
	}
	if _, err := p.Parse([]byte(`{"op":"=","args":[{"property":"name"},"x"]}`), api.LanguageJSON); err != nil {
		t.Fatalf("Parse JSON dispatch: %v", err)
	}
	if _, err := p.Parse([]byte("name = 'x'"), api.Language("unknown")); err == nil {
		t.Fatal("Parse unsupported language succeeded")
	}
}
