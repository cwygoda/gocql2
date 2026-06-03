# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @basic-spatial-functions
Feature: A.7 Basic Spatial Functions abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.
  The expected-fail tag marks ATS entries not yet covered by regular package tests.

  @test-25
  Scenario: A.7.1. Conformance Test 25 /conf/basic-spatial-functions/s_intersects
    # ATS section: A.7.1
    # ATS id: /conf/basic-spatial-functions/s_intersects
    # Requirements:
    #   /req/basic-spatial-functions/spatial-predicate , /req/basic-spatial-functions/spatial-functions , /req/basic-spatial-functions/spatial-data-types
    # Purpose:
    #   Test the S_INTERSECTS spatial comparison function with points and bounding boxes.
    Given One or more data sources, each with a list of queryables.
    And At least one queryable has a geometry data type.
    When For each queryable {queryable} with a geometry data type, evaluate the following filter expressions
    And S_INTERSECTS({queryable},BBOX(-180,-90,180,90))
    And S_INTERSECTS({queryable},POINT(7.02 49.92))
    And S_INTERSECTS({queryable},POINT(90 180))
    And S_INTERSECTS({queryable},BBOX(-180,-90,-90,90)) AND S_INTERSECTS({queryable},BBOX(90,-90,180,90))
    Then assert successful execution of the evaluation for the first two filter expressions;
    And assert unsuccessful execution of the evaluation for the third filter expressions (invalid coordinate);
    And store the valid predicates for each data source.

  @expected-fail @test-26
  Scenario: A.7.2. Conformance Test 26 /conf/basic-spatial-functions/test-data
    # ATS section: A.7.2
    # ATS id: /conf/basic-spatial-functions/test-data
    # Requirements:
    #   all requirements
    # Purpose:
    #   Test predicates against the test dataset
    Given The implementation under test uses the test dataset.
    When Evaluate each predicate in Predicates and expected results .
    Then assert successful execution of the evaluation;
    And assert that the expected result is returned;
    And store the valid predicates for each data source.

  @expected-fail @test-27
  Scenario: A.7.3. Conformance Test 27 /conf/basic-spatial-functions/logical
    # ATS section: A.7.3
    # ATS id: /conf/basic-spatial-functions/logical
    # Requirements:
    #   n/a
    # Purpose:
    #   Test filter expressions with AND, OR and NOT including sub-expressions
    Given The stored predicates for each data source, including from the dependencies.
    When For each data source, select at least 10 random combinations of four predicates ( {p1} to {p4} ) from the stored predicates and evaluate the filter expression ((NOT {p1} AND {p2}) OR ({p3} and NOT {p4}) or not ({p1} AND {p4})) .
    Then assert successful execution of the evaluation.
