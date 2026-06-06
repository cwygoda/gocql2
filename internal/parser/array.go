package parser

import (
	"strings"

	"github.com/cwygoda/gocql2/api"
)

var arrayPredicateOps = map[string]api.ArrayPredicateOp{
	"a_contains":    api.ArrayOpContains,
	"a_containedby": api.ArrayOpContainedBy,
	"a_equals":      api.ArrayOpEquals,
	"a_overlaps":    api.ArrayOpOverlaps,
}

var jsonArrayPredicateOps = map[string]api.ArrayPredicateOp{
	"a_contains":    api.ArrayOpContains,
	"a_containedBy": api.ArrayOpContainedBy,
	"a_equals":      api.ArrayOpEquals,
	"a_overlaps":    api.ArrayOpOverlaps,
}

func isArrayPredicateOp(name string) (api.ArrayPredicateOp, bool) {
	op, ok := arrayPredicateOps[strings.ToLower(name)]
	return op, ok
}

func isJSONArrayPredicateOp(name string) (api.ArrayPredicateOp, bool) {
	op, ok := jsonArrayPredicateOps[name]
	return op, ok
}
