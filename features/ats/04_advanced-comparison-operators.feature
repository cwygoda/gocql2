# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @advanced-comparison-operators
Feature: A.4 Advanced Comparison Operators abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.
  The expected-fail tag marks ATS entries not yet covered by regular package tests.

  @expected-fail @test-10
  Scenario: A.4.1. Conformance Test 10 /conf/advanced-comparison-operators/like
    # ATS section: A.4.1
    # ATS id: /conf/advanced-comparison-operators/like
    # Requirements:
    #   /req/advanced-comparison-operators/like-predicate
    # Purpose:
    #   Test LIKE predicate
    Given One or more data sources, each with a list of queryables.
    When For each queryable {queryable} of type String, evaluate the following filter expressions
    And {queryable} LIKE '%'
    And {queryable} like '_%'
    And {queryable} like ''
    And {queryable} like '%%'
    And {queryable} like '\\%\\_'
    Then assert successful execution of the evaluation;
    And assert that the two result sets for each queryable for the pattern expression '_%' and '' have no item in common;
    And assert that the two result sets for each queryable for the pattern expression '%' and '%%' are identical;
    And store the valid predicates for each data source.

  @expected-fail @test-11
  Scenario: A.4.2. Conformance Test 11 /conf/advanced-comparison-operators/between
    # ATS section: A.4.2
    # ATS id: /conf/advanced-comparison-operators/between
    # Requirements:
    #   /req/advanced-comparison-operators/between-predicate
    # Purpose:
    #   Test BETWEEN predicate
    Given One or more data sources, each with a list of queryables.
    When for each queryable {queryable} of type Number or Integer, evaluate the following filter expressions
    And {queryable} BETWEEN 0 AND 100
    And {queryable} between 100.0 and 1.0
    Then assert successful execution of the evaluation;
    And store the valid predicates for each data source.

  @expected-fail @test-12
  Scenario: A.4.3. Conformance Test 12 /conf/advanced-comparison-operators/in
    # ATS section: A.4.3
    # ATS id: /conf/advanced-comparison-operators/in
    # Requirements:
    #   /req/advanced-comparison-operators/in-predicate
    # Purpose:
    #   Test IN predicate
    Given One or more data sources, each with a list of queryables.
    When for each queryable {queryable} of type Number or Integer, evaluate the following filter expression {queryable} IN (1, 2, 3) ;
    And for each queryable {queryable} of type String, evaluate the following filter expression {queryable} in ('foo', 'bar') ;
    And for each queryable {queryable} of type Boolean, evaluate the following filter expression {queryable} in (true) ;
    And for each queryable {queryable} of type Timestamp, evaluate the following filter expression {queryable} in ('2022-04-14T14:52:56Z', '2022-04-14T15:52:56Z') ;
    And for each queryable {queryable} of type Date, evaluate the following filter expression {queryable} in ('2022-04-14', '2022-04-15') ;
    Then assert successful execution of the evaluation;
    And store the valid predicates for each data source.

  @expected-fail @test-13
  Scenario: A.4.4. Conformance Test 13 /conf/advanced-comparison-operators/test-data
    # ATS section: A.4.4
    # ATS id: /conf/advanced-comparison-operators/test-data
    # Requirements:
    #   all requirements
    # Purpose:
    #   Test predicates against the test dataset
    Given The implementation under test uses the test dataset.
    When Evaluate each predicate in Predicates and expected results .
    Then assert successful execution of the evaluation;
    And assert that the expected result is returned;
    And store the valid predicates for each data source.

  @expected-fail @test-14
  Scenario: A.4.5. Conformance Test 14 /conf/advanced-comparison-operators/logical
    # ATS section: A.4.5
    # ATS id: /conf/advanced-comparison-operators/logical
    # Requirements:
    #   n/a
    # Purpose:
    #   Test filter expressions with AND, OR and NOT including sub-expressions
    Given The stored predicates for each data source, including from the dependencies.
    When For each data source, select at least 10 random combinations of four predicates ( {p1} to {p4} ) from the stored predicates and evaluate the filter expression ((NOT {p1} AND {p2}) OR ({p3} and NOT {p4}) or not ({p1} AND {p4})) .
    Then assert successful execution of the evaluation.
