# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @case-insensitive-comparison
Feature: A.5 Case-insensitive Comparison abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.
  The expected-fail tag marks ATS entries not yet covered by regular package tests.

  @expected-fail @test-15
  Scenario: A.5.1. Conformance Test 15 /conf/case-insensitive-comparison/casei
    # ATS section: A.5.1
    # ATS id: /conf/case-insensitive-comparison/casei
    # Requirements:
    #   /req/case-insensitive-comparison/casei-function
    # Purpose:
    #   Test the CASEI function in comparisons
    Given One or more data sources, each with a list of queryables.
    When For each queryable {queryable} of type String, evaluate the following filter expressions
    And CASEI({queryable}) = casei('foo')
    And CASEI({queryable}) <> casei('FOO')
    Then assert successful execution of the evaluation;
    And assert that the two result sets for each queryable have no item in common;
    And store the valid predicates for each data source.

  @expected-fail @test-16
  Scenario: A.5.2. Conformance Test 16 /conf/case-insensitive-comparison/casei-like
    # ATS section: A.5.2
    # ATS id: /conf/case-insensitive-comparison/casei-like
    # Requirements:
    #   /req/case-insensitive-comparison/casei-function
    # Purpose:
    #   Test the CASEI function in LIKE predicates
    Given One or more data sources, each with a list of queryables.
    And The conformance class Advanced Comparison Operators passes.
    When For each queryable {queryable} of type String, evaluate the following filter expressions
    And CASEI({queryable}) LIKE casei('foo%')
    And CASEI({queryable}) LIKE casei('FOO%')
    Then assert successful execution of the evaluation;
    And assert that the two result sets for each queryable are identical;
    And store the valid predicates for each data source.

  @expected-fail @test-17
  Scenario: A.5.3. Conformance Test 17 /conf/case-insensitive-comparison/test-data
    # ATS section: A.5.3
    # ATS id: /conf/case-insensitive-comparison/test-data
    # Requirements:
    #   all requirements
    # Purpose:
    #   Test predicates against the test dataset
    Given The implementation under test uses the test dataset.
    When Evaluate each predicate in Predicates and expected results , if the conditional dependency is met.
    Then assert successful execution of the evaluation;
    And assert that the expected result is returned;
    And store the valid predicates for each data source.

  @expected-fail @test-18
  Scenario: A.5.4. Conformance Test 18 /conf/case-insensitive-comparison/logical
    # ATS section: A.5.4
    # ATS id: /conf/case-insensitive-comparison/logical
    # Requirements:
    #   n/a
    # Purpose:
    #   Test filter expressions with AND, OR and NOT including sub-expressions
    Given The stored predicates for each data source, including from the dependencies.
    When For each data source, select at least 10 random combinations of four predicates ( {p1} to {p4} ) from the stored predicates and evaluate the filter expression ((NOT {p1} AND {p2}) OR ({p3} and NOT {p4}) or not ({p1} AND {p4})) .
    Then assert successful execution of the evaluation.
