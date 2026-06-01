# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @array-functions
Feature: A.11 Array Functions abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.
  The expected-fail tag marks ATS entries not yet covered by regular package tests.

  @test-44
  Scenario: A.11.1. Conformance Test 44 /conf/array-functions/array-predicates
    # ATS section: A.11.1
    # ATS id: /conf/array-functions/array-predicates
    # Requirements:
    #   /req/array-functions/array-predicates
    # Purpose:
    #   Test the array comparison functions
    Given One or more data sources, each with a list of queryables.
    And At least one queryable has an array data type.
    When For each queryable {queryable} with an array data type, evaluate the following filter expressions
    And A_CONTAINS({queryable},("foo","bar"))
    And A_CONTAINEDBY({queryable},("foo","bar"))
    And A_EQUALS({queryable},("foo","bar"))
    And A_OVERLAPS({queryable},("foo","bar"))
    Then assert successful execution of the evaluation;
    And store the valid predicates for each data source.

  @expected-fail @test-45
  Scenario: A.11.2. Conformance Test 45 /conf/array-functions/logical
    # ATS section: A.11.2
    # ATS id: /conf/array-functions/logical
    # Requirements:
    #   n/a
    # Purpose:
    #   Test filter expressions with AND, OR and NOT including sub-expressions
    Given The stored predicates for each data source, including from the dependencies.
    When For each data source, select at least 10 random combinations of four predicates ( {p1} to {p4} ) from the stored predicates and evaluate the filter expression ((NOT {p1} AND {p2}) OR ({p3} and NOT {p4}) or not ({p1} AND {p4})) .
    Then assert successful execution of the evaluation.
