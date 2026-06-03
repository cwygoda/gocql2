package gocql2

import "strings"

var arrayPredicateOps = map[string]ArrayPredicateOp{
	"a_contains":    ArrayOpContains,
	"a_containedby": ArrayOpContainedBy,
	"a_equals":      ArrayOpEquals,
	"a_overlaps":    ArrayOpOverlaps,
}

var jsonArrayPredicateOps = map[string]ArrayPredicateOp{
	"a_contains":    ArrayOpContains,
	"a_containedBy": ArrayOpContainedBy,
	"a_equals":      ArrayOpEquals,
	"a_overlaps":    ArrayOpOverlaps,
}

func isArrayPredicateOp(name string) (ArrayPredicateOp, bool) {
	op, ok := arrayPredicateOps[strings.ToLower(name)]
	return op, ok
}

func isJSONArrayPredicateOp(name string) (ArrayPredicateOp, bool) {
	op, ok := jsonArrayPredicateOps[name]
	return op, ok
}
