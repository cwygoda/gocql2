# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @accent-insensitive-comparison
Feature: A.6 Accent-insensitive Comparison abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.
  The expected-fail tag marks ATS entries not yet covered by regular package tests.

  @expected-fail @test-19
  Scenario: A.6.1. Conformance Test 19 /conf/accent-insensitive-comparison/accenti
    # ATS section: A.6.1
    # ATS id: /conf/accent-insensitive-comparison/accenti
    # Requirements:
    #   /req/accent-insensitive-comparison/accenti-function
    # Purpose:
    #   Test the ACCENTI function in comparisons
    Given One or more data sources, each with a list of queryables.
    When For each queryable {queryable} of type String, evaluate the following filter expressions
    And ACCENTI({queryable}) = accenti('äöüéáí')
    And ACCENTI({queryable}) <> accenti('aoueai')
    Then assert successful execution of the evaluation;
    And assert that the two result sets for each queryable have no item in common;
    And store the valid predicates for each data source.

  @expected-fail @test-20
  Scenario: A.6.2. Conformance Test 20 /conf/accent-insensitive-comparison/accenti-like
    # ATS section: A.6.2
    # ATS id: /conf/accent-insensitive-comparison/accenti-like
    # Requirements:
    #   /req/accent-insensitive-comparison/accenti-function
    # Purpose:
    #   Test the ACCENTI function in LIKE predicates
    Given One or more data sources, each with a list of queryables.
    And The conformance class Advanced Comparison Operators passes.
    When For each queryable {queryable} of type String, evaluate the following filter expressions
    And ACCENTI({queryable}) LIKE accenti('Ä%')
    And ACCENTI({queryable}) LIKE accenti('A%')
    Then assert successful execution of the evaluation;
    And assert that the two result sets for each queryable are identical;
    And store the valid predicates for each data source.

  @expected-fail @test-21
  Scenario: A.6.3. Conformance Test 21 /conf/accent-insensitive-comparison/accenti-casei
    # ATS section: A.6.3
    # ATS id: /conf/accent-insensitive-comparison/accenti-casei
    # Requirements:
    #   /req/accent-insensitive-comparison/accenti-function
    # Purpose:
    #   Test the ACCENTI function with the CASEI function
    Given One or more data sources, each with a list of queryables.
    And The conformance class Case-insensitive Comparison passes.
    When For each queryable {queryable} of type String, evaluate the following filter expressions
    And ACCENTI(CASEI({queryable})) = accenti(casei('ÄÉ'))
    And ACCENTI(CASEI({queryable})) = accenti(casei('ae'))
    Then assert successful execution of the evaluation;
    And assert that the two result sets for each queryable are identical;
    And store the valid predicates for each data source.

  @expected-fail @test-22
  Scenario: A.6.4. Conformance Test 22 /conf/accent-insensitive-comparison/accenti-casei-like
    # ATS section: A.6.4
    # ATS id: /conf/accent-insensitive-comparison/accenti-casei-like
    # Requirements:
    #   /req/accent-insensitive-comparison/accenti-function
    # Purpose:
    #   Test the ACCENTI function with the CASEI function in LIKE predicates
    Given One or more data sources, each with a list of queryables.
    And The conformance class Case-insensitive Comparison passes.
    And The conformance class Advanced Comparison Operators passes.
    When For each queryable {queryable} of type String, evaluate the following filter expressions
    And ACCENTI(CASEI({queryable})) LIKE accenti(casei('Ä%'))
    And ACCENTI(CASEI({queryable})) LIKE accenti(casei('a%'))
    Then assert successful execution of the evaluation;
    And assert that the two result sets for each queryable are identical;
    And store the valid predicates for each data source.

  @expected-fail @test-23
  Scenario: A.6.5. Conformance Test 23 /conf/accent-insensitive-comparison/test-data
    # ATS section: A.6.5
    # ATS id: /conf/accent-insensitive-comparison/test-data
    # Requirements:
    #   all requirements
    # Purpose:
    #   Test predicates against the test dataset
    Given The implementation under test uses the test dataset.
    When Evaluate each predicate in Predicates and expected results , if the conditional dependency is met.
    Then assert successful execution of the evaluation;
    And assert that the expected result is returned;
    And store the valid predicates for each data source.

  @expected-fail @test-24
  Scenario: A.6.6. Conformance Test 24 /conf/accent-insensitive-comparison/logical
    # ATS section: A.6.6
    # ATS id: /conf/accent-insensitive-comparison/logical
    # Requirements:
    #   n/a
    # Purpose:
    #   Test filter expressions with AND, OR and NOT including sub-expressions
    Given The stored predicates for each data source, including from the dependencies.
    When For each data source, select at least 10 random combinations of four predicates ( {p1} to {p4} ) from the stored predicates and evaluate the filter expression ((NOT {p1} AND {p2}) OR ({p3} and NOT {p4}) or not ({p1} AND {p4})) .
    Then assert successful execution of the evaluation.
