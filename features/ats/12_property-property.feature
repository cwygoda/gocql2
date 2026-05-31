# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @property-property
Feature: A.12 Property-Property Comparisons abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.
  The expected-fail tag marks ATS entries not yet covered by regular package tests.

  @expected-fail @test-46
  Scenario: A.12.1. Conformance Test 46 /conf/property-property/comparison-value-property
    # ATS section: A.12.1
    # ATS id: /conf/property-property/comparison-value-property
    # Requirements:
    #   /req/property-property/withdraw-permissions
    # Purpose:
    #   Test comparison predicates with properties on the right-hand side and values on the left-hand side
    Given One or more data sources, each with a list of queryables.
    When For each queryable {queryable} of one of the data types String, Boolean, Number, Integer, Timestamp or Date, evaluate the following filter expressions
    And {value} = {queryable}
    And {value} <> {queryable}
    And {value} > {queryable}
    And {value} < {queryable}
    And {value} >= {queryable}
    And {value} <= {queryable}
    And where {value} depends on the data type:
    And String: 'foo'
    And Boolean: true
    And Number: 3.14
    And Integer: 1
    And Timestamp: TIMESTAMP('2022-04-14T14:48:46Z')
    And Date: DATE('2022-04-14')
    Then assert successful execution of the evaluation;
    And assert that the two result sets for each queryable for the operators = and <> have no item in common;
    And assert that the two result sets for each queryable for the operators > and <= have no item in common;
    And assert that the two result sets for each queryable for the operators < and >= have no item in common;
    And store the valid predicates for each data source.

  @expected-fail @test-47
  Scenario: A.12.2. Conformance Test 47 /conf/property-property/comparison-property-property
    # ATS section: A.12.2
    # ATS id: /conf/property-property/comparison-property-property
    # Requirements:
    #   /req/property-property/withdraw-permissions
    # Purpose:
    #   Test comparison predicates with properties on both sides
    Given One or more data sources, each with a list of queryables.
    When For each queryable {queryable} of one of the data types String, Boolean, Number, Integer, Timestamp or Date, evaluate the following filter expressions
    And {queryable} = {queryable}
    And {queryable} <> {queryable}
    And {queryable} > {queryable}
    And {queryable} < {queryable}
    And {queryable} >= {queryable}
    And {queryable} <= {queryable}
    Then assert successful execution of the evaluation;
    And assert that the result sets for each queryable for the operators <> , < and > is empty;
    And assert that the result sets for each queryable for the operators = , >= and <= are identical;
    And store the valid predicates for each data source.

  @expected-fail @test-48
  Scenario: A.12.3. Conformance Test 48 /conf/property-property/comparison-value-value
    # ATS section: A.12.3
    # ATS id: /conf/property-property/comparison-value-value
    # Requirements:
    #   /req/property-property/withdraw-permissions
    # Purpose:
    #   Test comparison predicates with values on both sides
    Given n/a
    When Evaluate the following filter expressions
    And {value} = {value}
    And {value} <> {value}
    And {value} > {value}
    And {value} < {value}
    And {value} >= {value}
    And {value} <= {value}
    And for each {value} from the following list:
    And 'foo'
    And true
    And 3.14
    And 1
    And TIMESTAMP('2022-04-14T14:48:46Z')
    And DATE('2022-04-14')
    Then assert successful execution of the evaluation;
    And assert that the result sets for each queryable for the operators <> , < and > is empty;
    And assert that the result sets for each queryable for the operators = , >= and <= are identical;
    And store the valid predicates for each data source.

  @expected-fail @test-49
  Scenario: A.12.4. Conformance Test 49 /conf/property-property/test-data
    # ATS section: A.12.4
    # ATS id: /conf/property-property/test-data
    # Requirements:
    #   all requirements
    # Purpose:
    #   Test predicates against the test dataset
    Given The implementation under test uses the test dataset.
    When Evaluate each predicate in Predicates and expected results , if the conditional dependency is met.
    Then assert successful execution of the evaluation;
    And assert that the expected result is returned;
    And store the valid predicates for each data source.

  @expected-fail @test-50
  Scenario: A.12.5. Conformance Test 50 /conf/property-property/logical
    # ATS section: A.12.5
    # ATS id: /conf/property-property/logical
    # Requirements:
    #   n/a
    # Purpose:
    #   Test filter expressions with AND, OR and NOT including sub-expressions
    Given The stored predicates for each data source, including from the dependencies.
    When For each data source, select at least 10 random combinations of four predicates ( {p1} to {p4} ) from the stored predicates and evaluate the filter expression ((NOT {p1} AND {p2}) OR ({p3} and NOT {p4}) or not ({p1} AND {p4})) .
    Then assert successful execution of the evaluation.
