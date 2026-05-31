package gocql2

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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
	current    cql2AbstractTest

	total              int
	passed             int
	expectedFailures   int
	unexpectedPasses   int
	unexpectedFailures int

	mu           sync.Mutex
	expectedFail bool
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
		s.expectedFail = scenarioHasTag(sc, "@expected-fail")
		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, stepErr error) (context.Context, error) {
		if stepErr != nil {
			return ctx, stepErr
		}

		s.executeErr = executeCQL2AbstractTest(ctx, s.current)
		return ctx, s.recordAbstractTestResult()
	})

	ctx.Step(`^.*$`, s.theATSStepIsDocumented)
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

func (s *cql2ATSSuite) theATSStepIsDocumented() error {
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

func executeCQL2AbstractTest(_ context.Context, test cql2AbstractTest) error {
	if test.ID == "" {
		return errors.New("missing CQL2 abstract test id")
	}

	// TODO: Wire this to the parser/evaluator implementation as CQL2 support is built.
	return fmt.Errorf("CQL2 abstract test execution is not implemented for %s", test.ID)
}
