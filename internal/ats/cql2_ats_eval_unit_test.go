package ats

import (
	"reflect"
	"testing"

	"github.com/cwygoda/gocql2/api"
)

func TestATSValueValueComparisonLiteralsCoverFeatureList(t *testing.T) {
	want := map[api.PropertyType]string{
		api.PropertyTypeString:    "'foo'",
		api.PropertyTypeBoolean:   "true",
		api.PropertyTypeNumber:    "3.14",
		api.PropertyTypeInteger:   "1",
		api.PropertyTypeTimestamp: "TIMESTAMP('2022-04-14T14:48:46Z')",
		api.PropertyTypeDate:      "DATE('2022-04-14')",
	}
	literals := atsValueValueComparisonLiterals()
	if len(literals) != len(want) {
		t.Fatalf("got %d literals, want %d", len(literals), len(want))
	}
	for _, literal := range literals {
		if want[literal.Type] != literal.Text {
			t.Fatalf("literal for %s = %q, want %q", literal.Type, literal.Text, want[literal.Type])
		}
		delete(want, literal.Type)
	}
	if len(want) != 0 {
		t.Fatalf("missing literal types: %#v", want)
	}
}

func TestATSPredicateCombinations(t *testing.T) {
	predicates := make([]atsStoredPredicate, 8)
	for i := range predicates {
		predicates[i] = atsStoredPredicate{Filter: "p" + string(rune('0'+i))}
	}

	first := atsPredicateCombinations(predicates)
	second := atsPredicateCombinations(predicates)
	if len(first) != 10 {
		t.Fatalf("got %d combinations, want 10", len(first))
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("combinations are not deterministic:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if reflect.DeepEqual(atsCombinationFilters(first[0]), []string{"p0", "p1", "p2", "p3"}) {
		t.Fatalf("first combination follows the old lexicographic/sliding-window order: %#v", atsCombinationFilters(first[0]))
	}

	seen := map[string]struct{}{}
	for _, combo := range first {
		if len(combo) != 4 {
			t.Fatalf("combination length = %d, want 4", len(combo))
		}
		withinCombo := map[string]struct{}{}
		for _, predicate := range combo {
			if _, ok := withinCombo[predicate.Filter]; ok {
				t.Fatalf("combination contains duplicate predicate: %#v", atsCombinationFilters(combo))
			}
			withinCombo[predicate.Filter] = struct{}{}
		}
		key := atsCombinationKey(combo)
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate combination %q", key)
		}
		seen[key] = struct{}{}
	}
}

func TestATSPredicateCombinationsRequireFourPredicates(t *testing.T) {
	predicates := []atsStoredPredicate{{Filter: "p0"}, {Filter: "p1"}, {Filter: "p2"}}
	if combinations := atsPredicateCombinations(predicates); combinations != nil {
		t.Fatalf("got %#v, want nil", combinations)
	}
}

func atsCombinationFilters(combo []atsStoredPredicate) []string {
	filters := make([]string, len(combo))
	for i, predicate := range combo {
		filters[i] = predicate.Filter
	}
	return filters
}

func atsCombinationKey(combo []atsStoredPredicate) string {
	key := ""
	for i, filter := range atsCombinationFilters(combo) {
		if i > 0 {
			key += ","
		}
		key += filter
	}
	return key
}
