# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @arithmetic
Feature: A.14 Arithmetic Expressions abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.

  @test-52
  Scenario: A.14.1. Conformance Test 52 /conf/arithmetic/arithmetic
    # ATS section: A.14.1
    # ATS id: /conf/arithmetic/arithmetic
    # Requirements:
    #   /req/arithmetic/arithmetic
    # Purpose:
    #   Test predicates with arithmetic expressions
    Given One or more data sources, each with a list of queryables.
    And At least one queryable has a numeric data type.
    When For each queryable construct multiple valid filter expressions involving arithmetic expressions.
    Then assert successful execution of the evaluation.

  @test-53
  Scenario: A.14.2. Conformance Test 53 /conf/arithmetic/test-data
    # ATS section: A.14.2
    # ATS id: /conf/arithmetic/test-data
    # Requirements:
    #   all requirements
    # Purpose:
    #   Test predicates against the test dataset
    Given The implementation under test uses the test dataset.
    When Evaluate each predicate in Predicates and expected results , if the conditional dependency is met.
    Then assert successful execution of the evaluation;
    And assert that the expected result is returned;
    And store the valid predicates for each data source.

  @test-54
  Scenario: A.14.3. Conformance Test 54 /conf/arithmetic/logical
    # ATS section: A.14.3
    # ATS id: /conf/arithmetic/logical
    # Requirements:
    #   n/a
    # Purpose:
    #   Test filter expressions with AND, OR and NOT including sub-expressions
    Given The stored predicates for each data source, including from the dependencies.
    When For each data source, select at least 10 random combinations of four predicates ( {p1} to {p4} ) from the stored predicates and evaluate the filter expression ((NOT {p1} AND {p2}) OR ({p3} and NOT {p4}) or not ({p1} AND {p4})) .
    Then assert successful execution of the evaluation.
