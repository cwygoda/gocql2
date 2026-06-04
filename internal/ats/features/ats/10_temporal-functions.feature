# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @temporal-functions
Feature: A.10 Temporal Functions abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.

  @test-40
  Scenario: A.10.1. Conformance Test 40 /conf/temporal-functions/temporal-functions-1
    # ATS section: A.10.1
    # ATS id: /conf/temporal-functions/temporal-functions-1
    # Requirements:
    #   /req/temporal-functions/temporal-predicates , /req/temporal-functions/temporal-functions
    # Purpose:
    #   Test the T_AFTER, T_BEFORE, T_DISJOINT, T_EQUALS, T_INTERSECTS temporal comparison functions.
    Given One or more data sources, each with a list of queryables with at least one queryable of type Timestamp or Date.
    When For each queryable {queryable} of data type Timestamp, evaluate the following filter expressions
    And T_AFTER({queryable},TIMESTAMP('2022-04-24T07:59:57Z'))
    And T_AFTER({queryable},INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_BEFORE({queryable},TIMESTAMP('2022-04-24T07:59:57Z'))
    And T_BEFORE({queryable},INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_DISJOINT({queryable},TIMESTAMP('2022-04-24T07:59:57Z'))
    And T_DISJOINT({queryable},INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_EQUALS({queryable},TIMESTAMP('2022-04-24T07:59:57Z'))
    And T_EQUALS({queryable},INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_INTERSECTS({queryable},TIMESTAMP('2022-04-24T07:59:57Z'))
    And T_INTERSECTS({queryable},INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And For each queryable {queryable} of data type Date, evaluate the following filter expressions
    And T_AFTER({queryable},DATE('2022-04-24'))
    And T_AFTER({queryable},INTERVAL('2021-01-01','2021-12-31'))
    And T_BEFORE({queryable},DATE('2022-04-24'))
    And T_BEFORE({queryable},INTERVAL('2021-01-01','2021-12-31'))
    And T_DISJOINT({queryable},DATE('2022-04-24'))
    And T_DISJOINT({queryable},INTERVAL('2021-01-01','2021-12-31'))
    And T_EQUALS({queryable},DATE('2022-04-24'))
    And T_EQUALS({queryable},INTERVAL('2021-01-01','2021-12-31'))
    And T_INTERSECTS({queryable},DATE('2022-04-24'))
    And T_INTERSECTS({queryable},INTERVAL('2021-01-01','2021-12-31'))
    Then assert successful execution of the evaluation;
    And store the valid predicates for each data source.

  @test-41
  Scenario: A.10.2. Conformance Test 41 /conf/temporal-functions/temporal-functions-2
    # ATS section: A.10.2
    # ATS id: /conf/temporal-functions/temporal-functions-2
    # Requirements:
    #   /req/temporal-functions/temporal-predicates , /req/temporal-functions/temporal-functions
    # Purpose:
    #   Test the temporal comparison functions with intervals
    Given One or more data sources, each with a list of queryables with at least two queryables of type Timestamp or Date.
    When For each pair of queryables {queryable2} and {queryable2} of data type Timestamp, evaluate the following filter expressions
    And T_AFTER(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_BEFORE(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_DISJOINT(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_EQUALS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_INTERSECTS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_CONTAINS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_DURING(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_FINISHEDBY(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_FINISHES(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_MEETS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_METBY(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_OVERLAPPEDBY(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_OVERLAPS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_STARTEDBY(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And T_STARTS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01T00:00:00Z','2021-12-31T23:59:59Z'))
    And For each pair of queryables {queryable2} and {queryable2} of data type Date, evaluate the following filter expressions
    And T_AFTER(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_BEFORE(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_DISJOINT(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_EQUALS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_INTERSECTS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_CONTAINS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_DURING(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_FINISHEDBY(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_FINISHES(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_MEETS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_METBY(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_OVERLAPPEDBY(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_OVERLAPS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_STARTEDBY(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    And T_STARTS(INTERVAL({queryable1},{queryable2}),INTERVAL('2021-01-01','2021-12-31'))
    Then assert successful execution of the evaluation;
    And store the valid predicates for each data source.

  @test-42
  Scenario: A.10.3. Conformance Test 42 /conf/temporal-functions/test-data
    # ATS section: A.10.3
    # ATS id: /conf/temporal-functions/test-data
    # Requirements:
    #   all requirements
    # Purpose:
    #   Test predicates against the test dataset
    Given The implementation under test uses the test dataset.
    When Evaluate each predicate in Predicates and expected results .
    Then assert successful execution of the evaluation;
    And assert that the expected result is returned;
    And store the valid predicates for each data source.

  @test-43
  Scenario: A.10.4. Conformance Test 43 /conf/temporal-functions/logical
    # ATS section: A.10.4
    # ATS id: /conf/temporal-functions/logical
    # Requirements:
    #   n/a
    # Purpose:
    #   Test filter expressions with AND, OR and NOT including sub-expressions
    Given The stored predicates for each data source, including from the dependencies.
    When For each data source, select at least 10 random combinations of four predicates ( {p1} to {p4} ) from the stored predicates and evaluate the filter expression ((NOT {p1} AND {p2}) OR ({p3} and NOT {p4}) or not ({p1} AND {p4})) .
    Then assert successful execution of the evaluation.
