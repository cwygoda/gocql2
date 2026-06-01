package gocql2

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/cucumber/godog"
)

type cql2AbstractTest struct {
	Section string
	ID      string
}

type cql2ATSSuite struct {
	executeErr error
	parseErr   error
	parsed     Expression
	current    cql2AbstractTest

	total              int
	passed             int
	expectedFailures   int
	unexpectedPasses   int
	unexpectedFailures int

	mu             sync.Mutex
	expectedFail   bool
	executedByStep bool
}

func TestCQL2AbstractTestSuite(t *testing.T) {
	suiteState := &cql2ATSSuite{}

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
		"CQL2 ATS summary: %d/%d failed as expected; %d/%d passed; %d unexpected pass(es); %d unexpected failure(s)",
		suiteState.expectedFailures,
		suiteState.total,
		suiteState.passed,
		suiteState.total,
		suiteState.unexpectedPasses,
		suiteState.unexpectedFailures,
	)
	t.Log(summary)
	t.Run(fmt.Sprintf("summary: %d of %d failed as expected", suiteState.expectedFailures, suiteState.total), func(t *testing.T) {})

	if status != 0 {
		t.Fatalf("CQL2 Abstract Test Suite failed with status %d", status)
	}
}

func (s *cql2ATSSuite) initializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		s.current = cql2AbstractTestFromScenario(sc)
		s.executeErr = nil
		s.parseErr = nil
		s.parsed = nil
		s.expectedFail = scenarioHasTag(sc, "@expected-fail")
		s.executedByStep = false
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
	ctx.Step(`^At least one queryable has an array data type\.$`, s.atLeastOneQueryableHasArrayDataType)
	ctx.Step(`^For each queryable \{queryable\} with an array data type, evaluate the following filter expressions$`, s.forEachArrayQueryableEvaluateFilters)
	ctx.Step(`^(A_(?:CONTAINS|CONTAINEDBY|EQUALS|OVERLAPS)\(\{queryable\},\("foo","bar"\)\))$`, s.iEvaluateTheArrayPredicateTemplate)
	ctx.Step(`^assert successful execution of the evaluation;$`, s.arrayPredicateParsingSucceeds)
	ctx.Step(`^store the valid predicates for each data source\.$`, s.storeTheValidPredicatesForEachDataSource)
}

func scenarioHasTag(sc *godog.Scenario, tag string) bool {
	for _, scenarioTag := range sc.Tags {
		if scenarioTag.Name == tag {
			return true
		}
	}
	return false
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

const arrayPredicateATSID = "/conf/array-functions/array-predicates"

func (s *cql2ATSSuite) isArrayPredicateATS() bool {
	return s.current.ID == arrayPredicateATSID
}

func (s *cql2ATSSuite) oneOrMoreDataSourcesWithQueryableLists() error {
	if s.isArrayPredicateATS() {
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
	s.parsed, s.parseErr = ParseText(filter, WithAllowedProperties(
		PropertyDefinition{Name: "tags", Type: PropertyTypeArray},
		PropertyDefinition{Name: "foo", Type: PropertyTypeString},
		PropertyDefinition{Name: "bar", Type: PropertyTypeString},
	))
	return s.parseErr
}

func (s *cql2ATSSuite) arrayPredicateParsingSucceeds() error {
	if !s.isArrayPredicateATS() {
		return nil
	}
	s.executedByStep = true
	return s.parseErr
}

func (s *cql2ATSSuite) storeTheValidPredicatesForEachDataSource() error {
	if s.isArrayPredicateATS() {
		s.executedByStep = true
	}
	return nil
}

func (s *cql2ATSSuite) recordAbstractTestResult() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.total++

	if s.executeErr == nil {
		if s.expectedFail {
			s.unexpectedPasses++
			return fmt.Errorf("%s passed but is still tagged @expected-fail; remove the tag to mark it implemented", s.current.ID)
		}

		s.passed++
		return nil
	}

	if s.expectedFail {
		s.expectedFailures++
		return nil
	}

	s.unexpectedFailures++
	return s.executeErr
}
