package gocql2

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cucumber/godog"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type cql2AbstractTest struct {
	Section string
	ID      string
}

//nolint:govet // Test fixture fields are grouped by scenario state and summary counters for readability.
type cql2ATSSuite struct {
	executeErr error
	parseErr   error
	parsed     Expression
	current    cql2AbstractTest

	total  int
	passed int
	failed int

	mu             sync.Mutex
	executedByStep bool

	spatialParseErrs []error
	spatialFilters   []string

	atsEvaluations      []atsEvaluation
	storedATSPredicates []atsStoredPredicate
	atsDB               *sql.DB
	atsParseOpts        []ParseOption
	atsSQLOpts          []SQLOption
}

func TestCQL2AbstractTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PostGIS-backed ATS runner in short mode")
	}

	db, cleanup := setupATSPostGISDatabase(t)
	defer cleanup()

	props := atsFixturePostGISSQLProperties()
	suiteState := &cql2ATSSuite{
		atsDB: db,
		atsParseOpts: []ParseOption{
			WithConformance(
				ConformanceAdvancedComparisonOperators,
				ConformanceCaseInsensitiveComparison,
				ConformanceAccentInsensitiveComparison,
				ConformanceArithmetic,
				ConformanceTemporalFunctions,
				ConformanceArrayFunctions,
				ConformanceSpatialFunctions,
				ConformancePropertyProperty,
			),
			WithAllowedProperties(SQLPropertyDefinitions(props...)...),
		},
		atsSQLOpts: []SQLOption{WithSQLProperties(props...)},
	}

	suite := godog.TestSuite{
		Name:                "cql2-abstract-test-suite",
		ScenarioInitializer: suiteState.initializeScenario,
		Options: &godog.Options{
			Format:   "progress",
			Paths:    []string{"features/ats"},
			TestingT: t,
		},
	}

	status := suite.Run()
	summary := fmt.Sprintf(
		"CQL2 ATS summary: %d/%d passed; %d/%d failed",
		suiteState.passed,
		suiteState.total,
		suiteState.failed,
		suiteState.total,
	)
	t.Log(summary)
	t.Run(fmt.Sprintf("summary: %d of %d passed", suiteState.passed, suiteState.total), func(t *testing.T) {})

	if status != 0 {
		t.Fatalf("CQL2 Abstract Test Suite failed with status %d", status)
	}
}

func setupATSPostGISDatabase(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	ctr, err := postgres.Run(
		ctx,
		"postgis/postgis:16-3.4",
		postgres.WithDatabase("cql2"),
		postgres.WithUsername("cql2"),
		postgres.WithPassword("cql2"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		cancel()
		t.Skipf("PostGIS container unavailable for ATS runner: %v", err)
	}
	t.Cleanup(func() { testcontainers.CleanupContainer(t, ctr) })

	conn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	db, err := sql.Open("pgx", conn)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if err := setupPostGISFixture(ctx, db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("close ATS database after setup failure: %v", closeErr)
		}
		cancel()
		t.Fatal(err)
	}

	return db, func() {
		if err := db.Close(); err != nil {
			t.Errorf("close ATS database: %v", err)
		}
		cancel()
	}
}

func (s *cql2ATSSuite) initializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		s.current = cql2AbstractTestFromScenario(sc)
		s.executeErr = nil
		s.parseErr = nil
		s.parsed = nil
		s.executedByStep = false
		s.spatialParseErrs = nil
		s.spatialFilters = nil
		s.atsEvaluations = nil
		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, stepErr error) (context.Context, error) {
		if stepErr != nil {
			s.executeErr = stepErr
		} else if !s.executedByStep {
			s.executeErr = fmt.Errorf("CQL2 abstract test execution is not implemented for %s", s.current.ID)
		}
		return ctx, s.recordAbstractTestResult()
	})

	ctx.Step(`^I parse the CQL2 Text filter "([^"]*)"$`, s.iParseTheCQL2TextFilter)
	ctx.Step(`^I parse the CQL2 JSON filter:$`, s.iParseTheCQL2JSONFilter)
	ctx.Step(`^parsing succeeds$`, s.parsingSucceeds)
	ctx.Step(`^the comparison right literal is "([^"]*)"$`, s.theComparisonRightLiteralIs)

	ctx.Step(`^One or more data sources, each with a list of queryables\.$`, s.oneOrMoreDataSourcesWithQueryableLists)
	ctx.Step(`^One or more data sources\.$`, s.oneOrMoreDataSources)
	ctx.Step(`^n/a$`, s.nA)
	ctx.Step(`^Test '/conf/basic-cql2/basic-test' passes\.$`, s.dependencyPasses)
	ctx.Step(`^The conformance class Advanced Comparison Operators passes\.$`, s.dependencyPasses)
	ctx.Step(`^The conformance class Case-insensitive Comparison passes\.$`, s.dependencyPasses)
	ctx.Step(`^The implementation under test uses the test dataset\.$`, s.theImplementationUsesTheTestDataset)
	ctx.Step(`^The stored predicates for each data source\.$`, s.theStoredPredicatesForEachDataSource)
	ctx.Step(`^The stored predicates for each data source, including from the dependencies\.$`, s.theStoredPredicatesForEachDataSource)
	ctx.Step(`^assert that there is at least one queryable for each data source;$`, s.assertAtLeastOneQueryableForEachDataSource)
	ctx.Step(`^assert that the data type \(.+\) is specified for each queryable;$`, s.assertDataTypeSpecifiedForEachQueryable)
	ctx.Step(`^assert that at least one queryable for each data source is of data type String, Boolean, Number, Integer, Timestamp or Date\.$`, s.assertAtLeastOneScalarQueryable)
	ctx.Step(`^(?:For|for) each queryable \{queryable\} of one of the data types String, Boolean, Number, Integer, Timestamp or Date, evaluate the following filter expressions$`, s.forEachScalarQueryableEvaluateComparisonFilters)
	ctx.Step(`^For each queryable \{queryable\} , evaluate the following filter expressions$`, s.forEachQueryableEvaluateFilters)
	ctx.Step(`^For each data source, evaluate the following filter expressions$`, s.forEachDataSourceEvaluateFilters)
	ctx.Step(`^(\{queryable\} (?:=|<>|>|<|>=|<=) \{value\})$`, s.evaluateQueryableValueComparisonTemplate)
	ctx.Step(`^(\{value\} (?:=|<>|>|<|>=|<=) \{queryable\})$`, s.evaluateValueQueryableComparisonTemplate)
	ctx.Step(`^(\{queryable\} (?:=|<>|>|<|>=|<=) \{queryable\})$`, s.evaluateQueryableQueryableComparisonTemplate)
	ctx.Step(`^(\{value\} (?:=|<>|>|<|>=|<=) \{value\})$`, s.evaluateValueValueComparisonTemplate)
	ctx.Step(`^(\{queryable\} IS NULL)$`, s.evaluateQueryableIsNullTemplate)
	ctx.Step(`^(\{queryable\} is not null)$`, s.evaluateQueryableIsNullTemplate)
	ctx.Step(`^(true)$`, s.evaluateBooleanFilter)
	ctx.Step(`^(false)$`, s.evaluateBooleanFilter)
	ctx.Step(`^where \{value\} depends on the data type:$`, s.acknowledgeScalarValueMetadata)
	ctx.Step(`^(String|Boolean|Number|Integer|Timestamp|Date): .+$`, s.acknowledgeScalarTypedValueMetadata)
	ctx.Step(`^Evaluate the following filter expressions$`, s.evaluateFollowingFilterExpressions)
	ctx.Step(`^for each \{value\} from the following list:$`, s.acknowledgeValueList)
	ctx.Step(`^'foo'$`, s.acknowledgeValueLiteral)
	ctx.Step(`^3\.14$`, s.acknowledgeValueLiteral)
	ctx.Step(`^1$`, s.acknowledgeValueLiteral)
	ctx.Step(`^DATE\('\d+-\d+-\d+'\)$`, s.acknowledgeValueLiteral)
	ctx.Step(`^TIMESTAMP\('\d+-\d+-\d+T\d+:\d+:\d+Z'\)$`, s.acknowledgeValueLiteral)
	ctx.Step(`^(?:For|for) each queryable \{queryable\} of type String, evaluate the following filter expressions$`, s.forEachStringQueryableEvaluateFilters)
	ctx.Step(`^((?:\{queryable\}|CASEI\(\{queryable\}\)|ACCENTI\(\{queryable\}\)|ACCENTI\(CASEI\(\{queryable\}\)\)) [Ll][Ii][Kk][Ee] .+)$`, s.evaluateStringPredicateTemplate)
	ctx.Step(`^((?:CASEI\(\{queryable\}\)|ACCENTI\(\{queryable\}\)|ACCENTI\(CASEI\(\{queryable\}\)\)) (?:=|<>) .+)$`, s.evaluateStringPredicateTemplate)
	ctx.Step(`^(?:for|For) each queryable \{queryable\} of type Number or Integer, evaluate the following filter expressions$`, s.forEachNumericQueryableEvaluateFilters)
	ctx.Step(`^At least one queryable has a numeric data type\.$`, s.atLeastOneQueryableHasNumericDataType)
	ctx.Step(`^For each queryable construct multiple valid filter expressions involving arithmetic expressions\.$`, s.forEachQueryableConstructArithmeticExpressions)
	ctx.Step(`^The list of functions with arguments and return type supported by the implementation under test\.$`, s.theListOfFunctionsWithArgumentsAndReturnTypeSupportedByTheImplementationUnderTest)
	ctx.Step(`^For each function construct multiple valid filter expressions involving different operators\.$`, s.forEachFunctionConstructMultipleValidFilterExpressions)
	ctx.Step(`^(\{queryable\} [Bb][Ee][Tt][Ww][Ee][Ee][Nn] .+)$`, s.evaluateNumericPredicateTemplate)
	ctx.Step(`^(?:for|For) each queryable \{queryable\} of type (Number or Integer|String|Boolean|Timestamp|Date), evaluate the following filter expression (.+) ;$`, s.evaluateTypedQueryableFilterExpression)
	ctx.Step(`^Evaluate each predicate in Predicates and expected results(?: , if the conditional dependency is met)? \.$`, s.evaluateEachFixturePredicate)
	ctx.Step(`^Evaluate each predicate in Combinations of predicates and expected results \.$`, s.evaluateStoredPredicateCombinations)
	ctx.Step(`^For the data source 'ne_110m_populated_places_simple', evaluate the filter expression (.+) for each combination of predicates \{p1\} to \{p4\} in Combinations of predicates and expected results \.$`, s.evaluateSpecificStoredPredicateCombination)
	ctx.Step(`^For each data source, select at least 10 random combinations of four predicates \( \{p1\} to \{p4\} \) from the stored predicates and evaluate the filter expression (.+) \.$`, s.evaluateSpecificStoredPredicateCombination)
	ctx.Step(`^assert successful execution of the evaluation;$`, s.assertSuccessfulATSEvaluation)
	ctx.Step(`^assert successful execution of the evaluation\.$`, s.assertSuccessfulATSEvaluation)
	ctx.Step(`^assert that the expected result is returned\.$`, s.assertExpectedATSResultsReturned)
	ctx.Step(`^assert that the expected result is returned;$`, s.assertExpectedATSResultsReturned)
	ctx.Step(`^assert that the two result sets for each queryable for the operators (=|>|<) and (<>|<=|>=) have no item in common;$`, s.assertOperatorResultSetsDisjoint)
	ctx.Step(`^assert that the result sets for each queryable for the operators <> , < and > is empty;$`, s.assertOperatorResultSetsEmpty)
	ctx.Step(`^assert that the result sets for each queryable for the operators = , >= and <= are identical;$`, s.assertOperatorResultSetsIdentical)
	ctx.Step(`^assert that the two result sets for each queryable have no item in common;$`, s.assertPairedResultSetsDisjoint)
	ctx.Step(`^assert that the two result sets for each queryable are identical;$`, s.assertPairedResultSetsIdentical)
	ctx.Step(`^assert that the result sets for false are empty;$`, s.assertFalseResultSetsEmpty)
	ctx.Step(`^assert that the two result sets for each queryable for the pattern expression '([^']*)' and '([^']*)' have no item in common;$`, s.assertPatternResultSetsDisjoint)
	ctx.Step(`^assert that the two result sets for each queryable for the pattern expression '([^']*)' and '([^']*)' are identical;$`, s.assertPatternResultSetsIdentical)
	ctx.Step(`^At least one queryable has an array data type\.$`, s.atLeastOneQueryableHasArrayDataType)
	ctx.Step(`^For each queryable \{queryable\} with an array data type, evaluate the following filter expressions$`, s.forEachArrayQueryableEvaluateFilters)
	ctx.Step(`^(A_(?:CONTAINS|CONTAINEDBY|EQUALS|OVERLAPS)\(\{queryable\},\("foo","bar"\)\))$`, s.iEvaluateTheArrayPredicateTemplate)

	ctx.Step(`^At least one queryable has a geometry data type\.$`, s.atLeastOneQueryableHasGeometryDataType)
	ctx.Step(`^(?:For|for) each queryable \{queryable\} with a geometry data type, evaluate the following filter expressions$`, s.forEachSpatialQueryableEvaluateFilters)
	ctx.Step(`^(?:For|for) each queryable \{queryable\} of type .+, evaluate the following filter expressions$`, s.forEachSpatialQueryableEvaluateFilters)
	ctx.Step(`^(?:For|for) each queryable \{queryable\} of type .+, evaluate the filter expression (S_(?:INTERSECTS|DISJOINT|EQUALS|TOUCHES|CROSSES|WITHIN|CONTAINS|OVERLAPS)\(.+\))$`, s.iEvaluateTheSpatialPredicateTemplate)
	ctx.Step(`^(S_(?:INTERSECTS|DISJOINT|EQUALS|TOUCHES|CROSSES|WITHIN|CONTAINS|OVERLAPS)\(.+\)(?: AND S_(?:INTERSECTS|DISJOINT|EQUALS|TOUCHES|CROSSES|WITHIN|CONTAINS|OVERLAPS)\(.+\))?)$`, s.iEvaluateTheSpatialPredicateTemplate)
	ctx.Step(`^assert successful execution of the evaluation for the first (two|four) filter expressions;$`, s.assertSpatialSuccessFirst)
	ctx.Step(`^assert successful execution of the evaluation for all filter expressions except the first;$`, s.assertSpatialSuccessExceptFirst)
	ctx.Step(`^assert unsuccessful execution of the evaluation for the (first|third|fifth) filter expressions \(invalid coordinate\);$`, s.assertSpatialFailureOrdinal)

	ctx.Step(`^One or more data sources, each with a list of queryables with at least one queryable of type Timestamp or Date\.$`, s.oneOrMoreDataSourcesWithTemporalQueryable)
	ctx.Step(`^One or more data sources, each with a list of queryables with at least two queryables of type Timestamp or Date\.$`, s.oneOrMoreDataSourcesWithTemporalQueryable)
	ctx.Step(`^For each queryable \{queryable\} of data type Timestamp, evaluate the following filter expressions$`, s.forEachTemporalQueryableEvaluateFilters)
	ctx.Step(`^For each queryable \{queryable\} of data type Date, evaluate the following filter expressions$`, s.forEachTemporalQueryableEvaluateFilters)
	ctx.Step(`^For each pair of queryables \{queryable2\} and \{queryable2\} of data type Timestamp, evaluate the following filter expressions$`, s.forEachTemporalQueryableEvaluateFilters)
	ctx.Step(`^For each pair of queryables \{queryable2\} and \{queryable2\} of data type Date, evaluate the following filter expressions$`, s.forEachTemporalQueryableEvaluateFilters)
	ctx.Step(`^(T_(?:AFTER|BEFORE|CONTAINS|DISJOINT|DURING|EQUALS|FINISHEDBY|FINISHES|INTERSECTS|MEETS|METBY|OVERLAPPEDBY|OVERLAPS|STARTEDBY|STARTS)\(.+\))$`, s.iEvaluateTheTemporalPredicateTemplate)

	ctx.Step(`^store the valid predicates for each data source\.$`, s.storeTheValidPredicatesForEachDataSource)
}

var cql2AbstractTestScenarioPattern = regexp.MustCompile(`^(A\.\d+(?:\.\d+)*)\. .* (/conf/\S+)$`)

func cql2AbstractTestFromScenario(sc *godog.Scenario) cql2AbstractTest {
	matches := cql2AbstractTestScenarioPattern.FindStringSubmatch(sc.Name)
	if matches == nil {
		return cql2AbstractTest{}
	}

	return cql2AbstractTest{Section: matches[1], ID: matches[2]}
}

func (s *cql2ATSSuite) iParseTheCQL2TextFilter(filter string) error {
	s.executedByStep = true
	s.parsed, s.parseErr = ParseText(filter)
	return nil
}

func (s *cql2ATSSuite) iParseTheCQL2JSONFilter(doc *godog.DocString) error {
	s.executedByStep = true
	s.parsed, s.parseErr = ParseJSON([]byte(doc.Content))
	return nil
}

func (s *cql2ATSSuite) parsingSucceeds() error {
	s.executedByStep = true
	return s.parseErr
}

func (s *cql2ATSSuite) theComparisonRightLiteralIs(want string) error {
	s.executedByStep = true
	if s.parseErr != nil {
		return s.parseErr
	}
	comparison, ok := s.parsed.(*ComparisonExpression)
	if !ok {
		return fmt.Errorf("parsed expression is %T, want *ComparisonExpression", s.parsed)
	}
	literal, ok := comparison.Right.(*Literal)
	if !ok {
		return fmt.Errorf("comparison right operand is %T, want *Literal", comparison.Right)
	}
	if literal.Value != want {
		return fmt.Errorf("comparison right literal is %q, want %q", literal.Value, want)
	}
	return nil
}

const (
	arrayPredicateATSID     = "/conf/array-functions/array-predicates"
	temporalFunctions1ATSID = "/conf/temporal-functions/temporal-functions-1"
	temporalFunctions2ATSID = "/conf/temporal-functions/temporal-functions-2"
)

func (s *cql2ATSSuite) isArrayPredicateATS() bool {
	return s.current.ID == arrayPredicateATSID
}

func (s *cql2ATSSuite) isTemporalFunctionsATS() bool {
	return s.current.ID == temporalFunctions1ATSID || s.current.ID == temporalFunctions2ATSID
}

func (s *cql2ATSSuite) isSpatialATS() bool {
	return strings.Contains(s.current.ID, "spatial-functions")
}

func (s *cql2ATSSuite) oneOrMoreDataSourcesWithQueryableLists() error {
	if s.isImplementedScalarATS() || s.isArrayPredicateATS() || s.isSpatialATS() {
		s.executedByStep = true
	}
	return nil
}

func (s *cql2ATSSuite) atLeastOneQueryableHasArrayDataType() error {
	if s.isArrayPredicateATS() {
		s.executedByStep = true
	}
	return nil
}

func (s *cql2ATSSuite) forEachArrayQueryableEvaluateFilters() error {
	if s.isArrayPredicateATS() {
		s.executedByStep = true
	}
	return nil
}

func (s *cql2ATSSuite) iEvaluateTheArrayPredicateTemplate(filter string) error {
	if !s.isArrayPredicateATS() {
		return nil
	}
	s.executedByStep = true
	filter = strings.ReplaceAll(filter, "{queryable}", "tags")
	s.parsed, s.parseErr = ParseText(
		filter,
		WithConformance(ConformanceArrayFunctions),
		WithAllowedProperties(
			PropertyDefinition{Name: "tags", Type: PropertyTypeArray},
			PropertyDefinition{Name: "foo", Type: PropertyTypeString},
			PropertyDefinition{Name: "bar", Type: PropertyTypeString},
		),
	)
	return s.parseErr
}

func (s *cql2ATSSuite) atLeastOneQueryableHasGeometryDataType() error {
	if s.isSpatialATS() {
		s.executedByStep = true
	}
	return nil
}

func (s *cql2ATSSuite) forEachSpatialQueryableEvaluateFilters() error {
	if s.isSpatialATS() {
		s.executedByStep = true
	}
	return nil
}

func (s *cql2ATSSuite) iEvaluateTheSpatialPredicateTemplate(filter string) error {
	if !s.isSpatialATS() {
		return nil
	}
	s.executedByStep = true
	filter = strings.ReplaceAll(filter, "{queryable}", "geom")
	s.parsed, s.parseErr = ParseText(
		filter,
		WithConformance(ConformanceSpatialFunctions),
		WithAllowedProperties(
			PropertyDefinition{Name: "geom", Type: PropertyTypeGeometry},
		),
	)
	s.spatialFilters = append(s.spatialFilters, filter)
	s.spatialParseErrs = append(s.spatialParseErrs, s.parseErr)
	return nil
}

func (s *cql2ATSSuite) oneOrMoreDataSourcesWithTemporalQueryable() error {
	if s.isTemporalFunctionsATS() {
		s.executedByStep = true
	}
	return nil
}

func (s *cql2ATSSuite) forEachTemporalQueryableEvaluateFilters() error {
	if s.isTemporalFunctionsATS() {
		s.executedByStep = true
	}
	return nil
}

func (s *cql2ATSSuite) iEvaluateTheTemporalPredicateTemplate(filter string) error {
	if !s.isTemporalFunctionsATS() {
		return nil
	}
	s.executedByStep = true
	filter = strings.ReplaceAll(filter, "{queryable1}", "start_time")
	filter = strings.ReplaceAll(filter, "{queryable2}", "end_time")
	filter = strings.ReplaceAll(filter, "{queryable}", "event_time")
	s.parsed, s.parseErr = ParseText(
		filter,
		WithConformance(ConformanceTemporalFunctions),
		WithAllowedProperties(
			PropertyDefinition{Name: "event_time", Type: PropertyTypeAny},
			PropertyDefinition{Name: "start_time", Type: PropertyTypeAny},
			PropertyDefinition{Name: "end_time", Type: PropertyTypeAny},
		),
	)
	return s.parseErr
}

func (s *cql2ATSSuite) arrayPredicateParsingSucceeds() error {
	if s.isSpatialATS() {
		s.executedByStep = true
		return s.assertSpatialSuccessAll()
	}
	if !s.isArrayPredicateATS() && !s.isTemporalFunctionsATS() {
		return nil
	}
	s.executedByStep = true
	return s.parseErr
}

func (s *cql2ATSSuite) storeTheValidPredicatesForEachDataSource() error {
	if s.isImplementedScalarATS() {
		s.executedByStep = true
		s.storeSuccessfulATSPredicates()
		return nil
	}
	if s.isArrayPredicateATS() || s.isTemporalFunctionsATS() || s.isSpatialATS() && len(s.spatialFilters) > 0 {
		s.executedByStep = true
	}
	return nil
}

func (s *cql2ATSSuite) assertSpatialSuccessFirst(countWord string) error {
	if !s.isSpatialATS() {
		return nil
	}
	s.executedByStep = true
	count := map[string]int{"two": 2, "four": 4}[countWord]
	return s.assertSpatialRangeSuccess(0, count)
}

func (s *cql2ATSSuite) assertSpatialSuccessExceptFirst() error {
	if !s.isSpatialATS() {
		return nil
	}
	s.executedByStep = true
	if len(s.spatialParseErrs) < 2 {
		return fmt.Errorf("expected at least two spatial predicate evaluations, got %d", len(s.spatialParseErrs))
	}
	return s.assertSpatialRangeSuccess(1, len(s.spatialParseErrs))
}

func (s *cql2ATSSuite) assertSpatialFailureOrdinal(ordinal string) error {
	if !s.isSpatialATS() {
		return nil
	}
	s.executedByStep = true
	index, err := ordinalIndex(ordinal)
	if err != nil {
		return err
	}
	if index >= len(s.spatialParseErrs) {
		return fmt.Errorf("expected %s spatial predicate evaluation, got %d", ordinal, len(s.spatialParseErrs))
	}
	if s.spatialParseErrs[index] == nil {
		return fmt.Errorf("spatial predicate %d parsed successfully, want invalid coordinate failure: %s", index+1, s.spatialFilters[index])
	}
	return nil
}

func (s *cql2ATSSuite) assertSpatialSuccessAll() error {
	return s.assertSpatialRangeSuccess(0, len(s.spatialParseErrs))
}

func (s *cql2ATSSuite) assertSpatialRangeSuccess(start, end int) error {
	if end > len(s.spatialParseErrs) {
		return fmt.Errorf("expected at least %d spatial predicate evaluations, got %d", end, len(s.spatialParseErrs))
	}
	for i := start; i < end; i++ {
		if err := s.spatialParseErrs[i]; err != nil {
			return fmt.Errorf("spatial predicate %d failed to parse (%s): %w", i+1, s.spatialFilters[i], err)
		}
	}
	return nil
}

func ordinalIndex(ordinal string) (int, error) {
	switch ordinal {
	case "first":
		return 0, nil
	case "third":
		return 2, nil
	case "fifth":
		return 4, nil
	default:
		return 0, fmt.Errorf("unsupported ordinal %q", ordinal)
	}
}

func (s *cql2ATSSuite) recordAbstractTestResult() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.total++

	if s.executeErr == nil {
		s.passed++
		return nil
	}

	s.failed++
	return s.executeErr
}
