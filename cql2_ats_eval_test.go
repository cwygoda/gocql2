package gocql2

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const (
	basicTestATSID                = "/conf/basic-cql2/basic-test"
	basicComparisonATSID          = "/conf/basic-cql2/comparison"
	basicIsNullATSID              = "/conf/basic-cql2/is-null"
	basicBooleanATSID             = "/conf/basic-cql2/boolean"
	basicTestDataATSID            = "/conf/basic-cql2/test-data"
	basicLogicalATSID             = "/conf/basic-cql2/logical"
	advancedLikeATSID             = "/conf/advanced-comparison-operators/like"
	advancedBetweenATSID          = "/conf/advanced-comparison-operators/between"
	advancedInATSID               = "/conf/advanced-comparison-operators/in"
	advancedTestDataATSID         = "/conf/advanced-comparison-operators/test-data"
	advancedLogicalATSID          = "/conf/advanced-comparison-operators/logical"
	caseiATSID                    = "/conf/case-insensitive-comparison/casei"
	caseiLikeATSID                = "/conf/case-insensitive-comparison/casei-like"
	caseiTestDataATSID            = "/conf/case-insensitive-comparison/test-data"
	caseiLogicalATSID             = "/conf/case-insensitive-comparison/logical"
	accentiATSID                  = "/conf/accent-insensitive-comparison/accenti"
	accentiLikeATSID              = "/conf/accent-insensitive-comparison/accenti-like"
	accentiCaseiATSID             = "/conf/accent-insensitive-comparison/accenti-casei"
	accentiCaseiLikeATSID         = "/conf/accent-insensitive-comparison/accenti-casei-like"
	accentiTestDataATSID          = "/conf/accent-insensitive-comparison/test-data"
	accentiLogicalATSID           = "/conf/accent-insensitive-comparison/logical"
	basicSpatialTestDataATSID     = "/conf/basic-spatial-functions/test-data"
	basicSpatialLogicalATSID      = "/conf/basic-spatial-functions/logical"
	basicSpatialPlusTestDataATSID = "/conf/basic-spatial-functions-plus/test-data"
	spatialTestDataATSID          = "/conf/spatial-functions/test-data"
	spatialLogicalATSID           = "/conf/spatial-functions/logical"
	temporalTestDataATSID         = "/conf/temporal-functions/test-data"
	temporalLogicalATSID          = "/conf/temporal-functions/logical"
)

//nolint:govet // Test evaluation records keep filter/query metadata before result payloads.
type atsEvaluation struct {
	Filter      string
	Queryable   string
	Key         string
	IDs         []string
	ExpectedIDs []string
	Err         error
}

type atsStoredPredicate struct {
	Filter string
	IDs    []string
}

func (s *cql2ATSSuite) isImplementedScalarATS() bool {
	switch s.current.ID {
	case basicTestATSID,
		basicComparisonATSID,
		basicIsNullATSID,
		basicBooleanATSID,
		basicTestDataATSID,
		basicLogicalATSID,
		advancedLikeATSID,
		advancedBetweenATSID,
		advancedInATSID,
		advancedTestDataATSID,
		advancedLogicalATSID,
		caseiATSID,
		caseiLikeATSID,
		caseiTestDataATSID,
		caseiLogicalATSID,
		accentiATSID,
		accentiLikeATSID,
		accentiCaseiATSID,
		accentiCaseiLikeATSID,
		accentiTestDataATSID,
		accentiLogicalATSID,
		basicSpatialTestDataATSID,
		basicSpatialLogicalATSID,
		basicSpatialPlusTestDataATSID,
		spatialTestDataATSID,
		spatialLogicalATSID,
		temporalTestDataATSID,
		temporalLogicalATSID:
		return true
	default:
		return false
	}
}

func (s *cql2ATSSuite) oneOrMoreDataSources() error {
	if s.isImplementedScalarATS() {
		s.executedByStep = true
	}
	return nil
}

func (s *cql2ATSSuite) nA() error {
	if !s.isImplementedScalarATS() {
		return s.unimplementedATSFixtureStep()
	}
	s.executedByStep = true
	return nil
}

func (s *cql2ATSSuite) dependencyPasses() error {
	if !s.isImplementedScalarATS() {
		return s.unimplementedATSFixtureStep()
	}
	s.executedByStep = true
	return nil
}

func (s *cql2ATSSuite) acknowledgeScalarValueMetadata() error {
	if !s.isImplementedScalarATS() {
		return s.unimplementedATSFixtureStep()
	}
	s.executedByStep = true
	return nil
}

func (s *cql2ATSSuite) acknowledgeScalarTypedValueMetadata(string) error {
	return s.acknowledgeScalarValueMetadata()
}

func (s *cql2ATSSuite) theImplementationUsesTheTestDataset() error {
	if !s.isImplementedScalarATS() {
		return s.unimplementedATSFixtureStep()
	}
	s.executedByStep = true
	return nil
}

func (s *cql2ATSSuite) theStoredPredicatesForEachDataSource() error {
	if !s.isImplementedScalarATS() {
		return nil
	}
	s.executedByStep = true
	if len(s.storedATSPredicates) < 4 {
		return fmt.Errorf("need at least four stored ATS predicates, got %d", len(s.storedATSPredicates))
	}
	return nil
}

func (s *cql2ATSSuite) assertAtLeastOneQueryableForEachDataSource() error {
	if !s.isImplementedScalarATS() {
		return nil
	}
	s.executedByStep = true
	if len(atsFixture.Queryables) == 0 {
		return fmt.Errorf("ATS fixture data source %q has no queryables", atsFixture.Name)
	}
	return nil
}

func (s *cql2ATSSuite) assertDataTypeSpecifiedForEachQueryable() error {
	if !s.isImplementedScalarATS() {
		return nil
	}
	s.executedByStep = true
	for _, queryable := range atsFixture.Queryables {
		if queryable.Type == PropertyTypeAny {
			return fmt.Errorf("queryable %q has no data type", queryable.Name)
		}
	}
	return nil
}

func (s *cql2ATSSuite) assertAtLeastOneScalarQueryable() error {
	if !s.isImplementedScalarATS() {
		return nil
	}
	s.executedByStep = true
	if len(atsFixtureQueryablesOfTypes(PropertyTypeString, PropertyTypeBoolean, PropertyTypeNumber, PropertyTypeInteger, PropertyTypeTimestamp, PropertyTypeDate)) == 0 {
		return fmt.Errorf("ATS fixture data source %q has no scalar queryable", atsFixture.Name)
	}
	return nil
}

func (s *cql2ATSSuite) forEachScalarQueryableEvaluateComparisonFilters() error {
	if s.current.ID != basicComparisonATSID {
		return nil
	}
	s.executedByStep = true
	return nil
}

func (s *cql2ATSSuite) forEachQueryableEvaluateFilters() error {
	if s.current.ID != basicIsNullATSID {
		return nil
	}
	s.executedByStep = true
	return nil
}

func (s *cql2ATSSuite) forEachDataSourceEvaluateFilters() error {
	if s.current.ID != basicBooleanATSID {
		return nil
	}
	s.executedByStep = true
	return nil
}

func (s *cql2ATSSuite) forEachStringQueryableEvaluateFilters() error {
	if s.current.ID != advancedLikeATSID && !s.isInsensitiveStringPredicateATS() {
		return nil
	}
	s.executedByStep = true
	return nil
}

func (s *cql2ATSSuite) isInsensitiveStringPredicateATS() bool {
	switch s.current.ID {
	case caseiATSID,
		caseiLikeATSID,
		accentiATSID,
		accentiLikeATSID,
		accentiCaseiATSID,
		accentiCaseiLikeATSID:
		return true
	default:
		return false
	}
}

func (s *cql2ATSSuite) forEachNumericQueryableEvaluateFilters() error {
	if s.current.ID != advancedBetweenATSID {
		return nil
	}
	s.executedByStep = true
	return nil
}

func (s *cql2ATSSuite) evaluateQueryableValueComparisonTemplate(template string) error {
	if s.current.ID != basicComparisonATSID {
		return nil
	}
	s.executedByStep = true
	op := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(template, "{queryable}"), "{value}"))
	for _, queryable := range atsFixtureQueryablesOfTypes(PropertyTypeString, PropertyTypeBoolean, PropertyTypeNumber, PropertyTypeInteger, PropertyTypeTimestamp, PropertyTypeDate) {
		filter := queryable.Name + " " + op + " " + atsComparisonLiteralForType(queryable.Type)
		s.recordATSEvaluation(filter, queryable.Name, op, nil)
	}
	return nil
}

func (s *cql2ATSSuite) evaluateQueryableIsNullTemplate(template string) error {
	if s.current.ID != basicIsNullATSID {
		return nil
	}
	s.executedByStep = true
	not := strings.Contains(strings.ToLower(template), "not null")
	key := "is null"
	if not {
		key = "is not null"
	}
	for _, queryable := range atsFixture.Queryables {
		filter := queryable.Name + " " + key
		s.recordATSEvaluation(filter, queryable.Name, key, nil)
	}
	return nil
}

func (s *cql2ATSSuite) evaluateBooleanFilter(filter string) error {
	if s.current.ID != basicBooleanATSID {
		return nil
	}
	s.executedByStep = true
	s.recordATSEvaluation(filter, atsFixture.Name, strings.ToLower(filter), nil)
	return nil
}

func (s *cql2ATSSuite) evaluateStringPredicateTemplate(template string) error {
	if !s.isImplementedScalarATS() {
		return s.unimplementedATSFixtureStep()
	}
	if s.current.ID != advancedLikeATSID && !s.isInsensitiveStringPredicateATS() {
		return nil
	}
	s.executedByStep = true
	for _, queryable := range atsFixtureQueryablesOfTypes(PropertyTypeString) {
		filter := strings.ReplaceAll(template, "{queryable}", queryable.Name)
		s.recordATSEvaluation(filter, queryable.Name, atsPatternKey(filter), nil)
	}
	return nil
}

func (s *cql2ATSSuite) evaluateNumericPredicateTemplate(template string) error {
	if s.current.ID != advancedBetweenATSID {
		return nil
	}
	s.executedByStep = true
	for _, queryable := range atsFixtureQueryablesOfTypes(PropertyTypeNumber, PropertyTypeInteger) {
		filter := strings.ReplaceAll(template, "{queryable}", queryable.Name)
		s.recordATSEvaluation(filter, queryable.Name, filter, nil)
	}
	return nil
}

func (s *cql2ATSSuite) evaluateTypedQueryableFilterExpression(typ, template string) error {
	if s.current.ID != advancedInATSID {
		return nil
	}
	s.executedByStep = true
	for _, queryable := range atsFixtureQueryablesForATSName(typ) {
		filter := strings.ReplaceAll(template, "{queryable}", queryable.Name)
		filter = normalizeATSTemporalInLiterals(typ, filter)
		s.recordATSEvaluation(filter, queryable.Name, filter, nil)
	}
	return nil
}

func (s *cql2ATSSuite) evaluateEachFixturePredicate() error {
	if !s.isImplementedScalarATS() {
		return s.unimplementedATSFixtureStep()
	}
	if s.current.ID != basicTestDataATSID &&
		s.current.ID != advancedTestDataATSID &&
		s.current.ID != caseiTestDataATSID &&
		s.current.ID != accentiTestDataATSID &&
		s.current.ID != basicSpatialTestDataATSID &&
		s.current.ID != basicSpatialPlusTestDataATSID &&
		s.current.ID != spatialTestDataATSID &&
		s.current.ID != temporalTestDataATSID {
		return nil
	}
	s.executedByStep = true
	for _, predicate := range atsFixture.Predicates {
		if !s.fixturePredicateApplies(predicate) {
			continue
		}
		s.recordATSEvaluation(predicate.Filter, atsFixture.Name, predicate.Filter, predicate.WantIDs)
	}
	return nil
}

func (s *cql2ATSSuite) evaluateStoredPredicateCombinations() error {
	if s.current.ID != basicLogicalATSID {
		return nil
	}
	return s.evaluateSpecificStoredPredicateCombination("(NOT ({p2}) AND {p1}) OR ({p3} and {p4}) or not ({p1} OR {p4})")
}

func (s *cql2ATSSuite) evaluateSpecificStoredPredicateCombination(template string) error {
	if !s.isLogicalCombinationATS() {
		return nil
	}
	s.executedByStep = true
	combinations := atsPredicateCombinations(s.storedATSPredicates, 10)
	for _, combo := range combinations {
		filter := template
		for i, predicate := range combo {
			filter = strings.ReplaceAll(filter, fmt.Sprintf("{p%d}", i+1), "("+predicate.Filter+")")
		}
		expected := atsExpectedLogicalCombinationIDs(s.current.ID, combo)
		s.recordATSEvaluation(filter, atsFixture.Name, filter, expected)
	}
	return nil
}

func (s *cql2ATSSuite) assertSuccessfulATSEvaluation() error {
	if !s.isImplementedScalarATS() {
		if (s.isSpatialATS() || s.isArrayPredicateATS() || s.isTemporalFunctionsATS()) && len(s.spatialFilters) > 0 || s.isArrayPredicateATS() || s.isTemporalFunctionsATS() {
			return s.arrayPredicateParsingSucceeds()
		}
		return s.unimplementedATSFixtureStep()
	}
	s.executedByStep = true
	for _, evaluation := range s.atsEvaluations {
		if evaluation.Err != nil {
			return fmt.Errorf("evaluate %q: %w", evaluation.Filter, evaluation.Err)
		}
	}
	return nil
}

func (s *cql2ATSSuite) assertExpectedATSResultsReturned() error {
	if !s.isImplementedScalarATS() {
		return s.unimplementedATSFixtureStep()
	}
	s.executedByStep = true
	for _, evaluation := range s.atsEvaluations {
		if evaluation.ExpectedIDs == nil {
			continue
		}
		if !reflect.DeepEqual(evaluation.IDs, evaluation.ExpectedIDs) {
			return fmt.Errorf("ids for %q = %#v, want %#v", evaluation.Filter, evaluation.IDs, evaluation.ExpectedIDs)
		}
	}
	return nil
}

func (s *cql2ATSSuite) assertOperatorResultSetsDisjoint(left, right string) error {
	if s.current.ID != basicComparisonATSID {
		return nil
	}
	s.executedByStep = true
	return s.assertEvaluationsByQueryablePair(left, right, atsAssertDisjoint)
}

func (s *cql2ATSSuite) assertPairedResultSetsDisjoint() error {
	if s.current.ID == basicIsNullATSID {
		s.executedByStep = true
		return s.assertEvaluationsByQueryablePair("is null", "is not null", atsAssertDisjoint)
	}
	if s.isInsensitiveStringPredicateATS() {
		s.executedByStep = true
		return s.assertFirstTwoEvaluationsByQueryable(atsAssertDisjoint)
	}
	return nil
}

func (s *cql2ATSSuite) assertPairedResultSetsIdentical() error {
	if !s.isInsensitiveStringPredicateATS() {
		return nil
	}
	s.executedByStep = true
	return s.assertFirstTwoEvaluationsByQueryable(atsAssertIdentical)
}

func (s *cql2ATSSuite) assertFalseResultSetsEmpty() error {
	if s.current.ID != basicBooleanATSID {
		return nil
	}
	s.executedByStep = true
	for _, evaluation := range s.atsEvaluations {
		if evaluation.Key == "false" && len(evaluation.IDs) != 0 {
			return fmt.Errorf("false filter returned %#v, want empty result set", evaluation.IDs)
		}
	}
	return nil
}

func (s *cql2ATSSuite) assertPatternResultSetsDisjoint(left, right string) error {
	if s.current.ID != advancedLikeATSID {
		return nil
	}
	s.executedByStep = true
	return s.assertEvaluationsByQueryablePair(left, right, atsAssertDisjoint)
}

func (s *cql2ATSSuite) assertPatternResultSetsIdentical(left, right string) error {
	if s.current.ID != advancedLikeATSID {
		return nil
	}
	s.executedByStep = true
	return s.assertEvaluationsByQueryablePair(left, right, atsAssertIdentical)
}

func (s *cql2ATSSuite) unimplementedATSFixtureStep() error {
	return fmt.Errorf("ATS fixture evaluation is not implemented for %s", s.current.ID)
}

func (s *cql2ATSSuite) recordATSEvaluation(filter, queryable, key string, expected []string) {
	ids, err := s.evaluateATSFixtureFilter(filter)
	s.recordATSEvaluationResult(filter, queryable, key, ids, expected, err)
}

func (s *cql2ATSSuite) recordATSEvaluationResult(filter, queryable, key string, ids, expected []string, err error) {
	var expectedIDs []string
	if expected != nil {
		expectedIDs = append([]string{}, expected...)
	}
	s.atsEvaluations = append(s.atsEvaluations, atsEvaluation{
		Filter:      filter,
		Queryable:   queryable,
		Key:         key,
		IDs:         ids,
		ExpectedIDs: expectedIDs,
		Err:         err,
	})
}

func (s *cql2ATSSuite) storeSuccessfulATSPredicates() {
	seen := map[string]struct{}{}
	for _, predicate := range s.storedATSPredicates {
		seen[predicate.Filter] = struct{}{}
	}
	for _, evaluation := range s.atsEvaluations {
		if evaluation.Err != nil {
			continue
		}
		if _, ok := seen[evaluation.Filter]; ok {
			continue
		}
		seen[evaluation.Filter] = struct{}{}
		s.storedATSPredicates = append(s.storedATSPredicates, atsStoredPredicate{Filter: evaluation.Filter, IDs: append([]string(nil), evaluation.IDs...)})
	}
}

func (s *cql2ATSSuite) fixturePredicateApplies(predicate atsFixturePredicate) bool {
	switch s.current.ID {
	case basicTestDataATSID:
		return predicate.Conformance == ConformanceBasicCQL2
	case advancedTestDataATSID:
		return predicate.Conformance == ConformanceAdvancedComparisonOperators
	case caseiTestDataATSID:
		return predicate.Conformance == ConformanceCaseInsensitiveComparison
	case accentiTestDataATSID:
		return predicate.Conformance == ConformanceAccentInsensitiveComparison
	case basicSpatialTestDataATSID, basicSpatialPlusTestDataATSID, spatialTestDataATSID:
		return predicate.Conformance == ConformanceSpatialFunctions
	case temporalTestDataATSID:
		return predicate.Conformance == ConformanceTemporalFunctions
	default:
		return false
	}
}

func (s *cql2ATSSuite) isLogicalCombinationATS() bool {
	switch s.current.ID {
	case basicLogicalATSID,
		advancedLogicalATSID,
		caseiLogicalATSID,
		accentiLogicalATSID,
		basicSpatialLogicalATSID,
		spatialLogicalATSID,
		temporalLogicalATSID:
		return true
	default:
		return false
	}
}

func (s *cql2ATSSuite) assertFirstTwoEvaluationsByQueryable(assert func([]string, []string) error) error {
	byQueryable := map[string][]atsEvaluation{}
	for _, evaluation := range s.atsEvaluations {
		byQueryable[evaluation.Queryable] = append(byQueryable[evaluation.Queryable], evaluation)
	}
	for queryable, evaluations := range byQueryable {
		if len(evaluations) < 2 {
			return fmt.Errorf("queryable %q has %d evaluations, want at least two", queryable, len(evaluations))
		}
		if err := assert(evaluations[0].IDs, evaluations[1].IDs); err != nil {
			return fmt.Errorf("queryable %q first two result sets: %w", queryable, err)
		}
	}
	return nil
}

func (s *cql2ATSSuite) assertEvaluationsByQueryablePair(leftKey, rightKey string, assert func([]string, []string) error) error {
	left := map[string]atsEvaluation{}
	right := map[string]atsEvaluation{}
	for _, evaluation := range s.atsEvaluations {
		switch evaluation.Key {
		case leftKey:
			left[evaluation.Queryable] = evaluation
		case rightKey:
			right[evaluation.Queryable] = evaluation
		}
	}
	for queryable, leftEvaluation := range left {
		rightEvaluation, ok := right[queryable]
		if !ok {
			return fmt.Errorf("no %q evaluation for queryable %q", rightKey, queryable)
		}
		if err := assert(leftEvaluation.IDs, rightEvaluation.IDs); err != nil {
			return fmt.Errorf("queryable %q result sets for %q and %q: %w", queryable, leftKey, rightKey, err)
		}
	}
	return nil
}

func atsAssertDisjoint(left, right []string) error {
	seen := map[string]struct{}{}
	for _, id := range left {
		seen[id] = struct{}{}
	}
	for _, id := range right {
		if _, ok := seen[id]; ok {
			return fmt.Errorf("both contain %q", id)
		}
	}
	return nil
}

func atsAssertIdentical(left, right []string) error {
	if !reflect.DeepEqual(left, right) {
		return fmt.Errorf("left = %#v, right = %#v", left, right)
	}
	return nil
}

func (s *cql2ATSSuite) evaluateATSFixtureFilter(filter string) ([]string, error) {
	if s.atsDB == nil {
		return nil, fmt.Errorf("ATS PostGIS fixture is not initialized")
	}
	return postGISQueryIDs(context.Background(), s.atsDB, filter, s.atsParseOpts, s.atsSQLOpts)
}

func atsComparisonLiteralForType(typ PropertyType) string {
	switch typ {
	case PropertyTypeString:
		return "'foo'"
	case PropertyTypeBoolean:
		return "true"
	case PropertyTypeNumber:
		return "3.14"
	case PropertyTypeInteger:
		return "1"
	case PropertyTypeTimestamp:
		return "TIMESTAMP('2022-04-14T14:48:46Z')"
	case PropertyTypeDate:
		return "DATE('2022-04-14')"
	default:
		return "NULL"
	}
}

func atsFixtureQueryablesForATSName(name string) []atsFixtureQueryable {
	switch name {
	case "Number or Integer":
		return atsFixtureQueryablesOfTypes(PropertyTypeNumber, PropertyTypeInteger)
	case "String":
		return atsFixtureQueryablesOfTypes(PropertyTypeString)
	case "Boolean":
		return atsFixtureQueryablesOfTypes(PropertyTypeBoolean)
	case "Timestamp":
		return atsFixtureQueryablesOfTypes(PropertyTypeTimestamp)
	case "Date":
		return atsFixtureQueryablesOfTypes(PropertyTypeDate)
	default:
		return nil
	}
}

func normalizeATSTemporalInLiterals(typ, filter string) string {
	wrapper := ""
	switch typ {
	case "Timestamp":
		wrapper = "TIMESTAMP"
	case "Date":
		wrapper = "DATE"
	default:
		return filter
	}
	return regexp.MustCompile(`'([^']+)'`).ReplaceAllString(filter, wrapper+`('$1')`)
}

func atsPatternKey(filter string) string {
	matches := regexp.MustCompile(`'((?:[^'\\]|\\.)*)'`).FindAllStringSubmatch(filter, -1)
	if len(matches) == 0 {
		return filter
	}
	return strings.ReplaceAll(matches[len(matches)-1][1], `\\`, `\`)
}

func atsPredicateCombinations(predicates []atsStoredPredicate, limit int) [][]atsStoredPredicate {
	if len(predicates) < 4 {
		return nil
	}
	count := limit
	if maxCount := len(predicates) - 3; count > maxCount {
		count = maxCount
	}
	out := make([][]atsStoredPredicate, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, []atsStoredPredicate{predicates[i], predicates[i+1], predicates[i+2], predicates[i+3]})
	}
	return out
}

func atsExpectedLogicalCombinationIDs(atsID string, predicates []atsStoredPredicate) []string {
	ids := []string{}
	for _, row := range atsFixture.Rows {
		p1 := atsIDsContain(predicates[0].IDs, row.ID)
		p2 := atsIDsContain(predicates[1].IDs, row.ID)
		p3 := atsIDsContain(predicates[2].IDs, row.ID)
		p4 := atsIDsContain(predicates[3].IDs, row.ID)
		matched := false
		switch atsID {
		case basicLogicalATSID:
			matched = (!p2 && p1) || (p3 && p4) || (!p1 && !p4)
		case advancedLogicalATSID, caseiLogicalATSID, accentiLogicalATSID, basicSpatialLogicalATSID, spatialLogicalATSID, temporalLogicalATSID:
			matched = (!p1 && p2) || (p3 && !p4) || !p1 || !p4
		}
		if matched {
			ids = append(ids, row.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func atsIDsContain(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
