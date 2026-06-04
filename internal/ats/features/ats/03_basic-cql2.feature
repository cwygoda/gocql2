# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @basic-cql2
Feature: A.3 Basic CQL2 abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.

  @test-4
  Scenario: A.3.1. Conformance Test 4 /conf/basic-cql2/basic-test
    # ATS section: A.3.1
    # ATS id: /conf/basic-cql2/basic-test
    # Requirements:
    #   n/a
    # Purpose:
    #   Implementation under test provides sufficient information to construct filter expressions and supports comparison predicates
    Given One or more data sources, each with a list of queryables.
    When n/a
    Then assert that there is at least one queryable for each data source;
    And assert that the data type (String, Number, Integer, Boolean, Timestamp, Date, Interval, Point, MultiPoint, LineString, MultiLineString, Polygon, MultiPolygon, Geometry, GeometryCollection, or Array) is specified for each queryable;
    And assert that at least one queryable for each data source is of data type String, Boolean, Number, Integer, Timestamp or Date.

  @test-5
  Scenario: A.3.2. Conformance Test 5 /conf/basic-cql2/comparison
    # ATS section: A.3.2
    # ATS id: /conf/basic-cql2/comparison
    # Requirements:
    #   /req/basic-cql2/cql2-filter , /req/basic-cql2/property , /req/basic-cql2/binary-comparison-predicate
    # Purpose:
    #   Test comparison predicates
    Given One or more data sources, each with a list of queryables.
    And Test '/conf/basic-cql2/basic-test' passes.
    When For each queryable {queryable} of one of the data types String, Boolean, Number, Integer, Timestamp or Date, evaluate the following filter expressions
    And {queryable} = {value}
    And {queryable} <> {value}
    And {queryable} > {value}
    And {queryable} < {value}
    And {queryable} >= {value}
    And {queryable} <= {value}
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

  @test-6
  Scenario: A.3.3. Conformance Test 6 /conf/basic-cql2/is-null
    # ATS section: A.3.3
    # ATS id: /conf/basic-cql2/is-null
    # Requirements:
    #   /req/basic-cql2/cql2-filter , /req/basic-cql2/property , /req/basic-cql2/null-predicate
    # Purpose:
    #   Test IS NULL predicate
    Given One or more data sources, each with a list of queryables.
    And Test '/conf/basic-cql2/basic-test' passes.
    When For each queryable {queryable} , evaluate the following filter expressions
    And {queryable} IS NULL
    And {queryable} is not null
    Then assert successful execution of the evaluation;
    And assert that the two result sets for each queryable have no item in common;
    And store the valid predicates for each data source.

  @test-7
  Scenario: A.3.4. Conformance Test 7 /conf/basic-cql2/boolean
    # ATS section: A.3.4
    # ATS id: /conf/basic-cql2/boolean
    # Requirements:
    #   /req/basic-cql2/cql2-filter
    # Purpose:
    #   Test boolean filter expression
    Given One or more data sources.
    And Test '/conf/basic-cql2/basic-test' passes.
    When For each data source, evaluate the following filter expressions
    And true
    And false
    Then assert successful execution of the evaluation;
    And assert that the result sets for false are empty;
    And store the valid predicates for each data source.

  @test-8
  Scenario: A.3.5. Conformance Test 8 /conf/basic-cql2/test-data
    # ATS section: A.3.5
    # ATS id: /conf/basic-cql2/test-data
    # Requirements:
    #   all requirements
    # Purpose:
    #   Test predicates against the test dataset
    Given The implementation under test uses the test dataset.
    When Evaluate each predicate in Predicates and expected results .
    Then assert successful execution of the evaluation;
    And assert that the expected result is returned;
    And store the valid predicates for each data source.

  @test-9
  Scenario: A.3.6. Conformance Test 9 /conf/basic-cql2/logical
    # ATS section: A.3.6
    # ATS id: /conf/basic-cql2/logical
    # Requirements:
    #   /req/basic-cql2/cql2-filter
    # Purpose:
    #   Test filter expressions with AND, OR and NOT including sub-expressions
    Given One or more data sources.
    And The stored predicates for each data source.
    When Evaluate each predicate in Combinations of predicates and expected results .
    And For the data source 'ne_110m_populated_places_simple', evaluate the filter expression (NOT ({p2}) AND {p1}) OR ({p3} and {p4}) or not ({p1} OR {p4}) for each combination of predicates {p1} to {p4} in Combinations of predicates and expected results .
    Then assert successful execution of the evaluation;
    And assert that the expected result is returned.
