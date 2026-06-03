# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @spatial-functions
Feature: A.9 Spatial Functions abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.
  The expected-fail tag marks ATS entries not yet covered by regular package tests.

  @test-30
  Scenario: A.9.1. Conformance Test 30 /conf/spatial-functions/s_intersects
    # ATS section: A.9.1
    # ATS id: /conf/spatial-functions/s_intersects
    # Requirements:
    #   /req/spatial-functions/spatial-functions , /req/spatial-functions/spatial-data-types
    # Purpose:
    #   Test the S_INTERSECTS spatial function
    Given One or more data sources, each with a list of queryables.
    And At least one queryable has a geometry data type.
    When For each queryable {queryable} with a geometry data type, evaluate the following filter expressions
    And S_INTERSECTS({queryable},BBOX(-180,-90,180,90))
    And S_INTERSECTS({queryable},POLYGON((-180 -90,180 -90,180 90,-180 90,-180 -90)))
    And S_INTERSECTS({queryable},LINESTRING(7 50, 10 51))
    And S_INTERSECTS({queryable},POINT(7.02 49.92))
    And S_INTERSECTS({queryable},POINT(90 180))
    Then assert successful execution of the evaluation for the first four filter expressions;
    And assert unsuccessful execution of the evaluation for the fifth filter expressions (invalid coordinate);
    And assert that the two result sets of the first two filter expressions for each queryable are identical;
    And store the valid predicates for each data source.

  @test-31
  Scenario: A.9.2. Conformance Test 31 /conf/spatial-functions/s_disjoint
    # ATS section: A.9.2
    # ATS id: /conf/spatial-functions/s_disjoint
    # Requirements:
    #   /req/spatial-functions/spatial-functions , /req/spatial-functions/spatial-data-types
    # Purpose:
    #   Test the S_DISJOINT spatial function
    Given One or more data sources, each with a list of queryables.
    When for each queryable {queryable} with a geometry data type, evaluate the following filter expressions
    And S_DISJOINT({queryable},BBOX(-180,-90,180,90))
    And S_DISJOINT({queryable},POLYGON((-180 -90,180 -90,180 90,-180 90,-180 -90)))
    And S_DISJOINT({queryable},LINESTRING(7 50,10 51))
    And S_DISJOINT({queryable},POINT(7.02 49.92))
    And S_DISJOINT({queryable},POINT(90 180))
    Then assert successful execution of the evaluation for the first four filter expressions;
    And assert unsuccessful execution of the evaluation for the fifth filter expressions (invalid coordinate);
    And assert that the two result sets of the first two filter expressions for each queryable are empty;
    And assert that the results sets of the third and fourth filter expressions for each queryable do not have an item in common with the corresponding S_INTERSECTS expression;
    And store the valid predicates for each data source.

  @test-32
  Scenario: A.9.3. Conformance Test 32 /conf/spatial-functions/s_equals
    # ATS section: A.9.3
    # ATS id: /conf/spatial-functions/s_equals
    # Requirements:
    #   /req/spatial-functions/spatial-functions , /req/spatial-functions/spatial-data-types
    # Purpose:
    #   Test the S_EQUALS spatial function
    Given One or more data sources, each with a list of queryables.
    When for each queryable {queryable} with a geometry data type, evaluate the following filter expressions
    And S_EQUALS({queryable},POLYGON((-180 -90,180 -90,180 90,-180 90,-180 -90)))
    And S_EQUALS({queryable},LINESTRING(7 50,10 51))
    And S_EQUALS({queryable},POINT(7.02 49.92))
    Then assert successful execution of the evaluation;
    And assert that the two result sets of the first two filter expressions for each queryable are identical;
    And store the valid predicates for each data source.

  @test-33
  Scenario: A.9.4. Conformance Test 33 /conf/spatial-functions/s_touches
    # ATS section: A.9.4
    # ATS id: /conf/spatial-functions/s_touches
    # Requirements:
    #   /req/spatial-functions/spatial-functions , /req/spatial-functions/spatial-data-types
    # Purpose:
    #   Test the S_TOUCHES spatial function
    Given One or more data sources, each with a list of queryables.
    When for each queryable {queryable} with a geometry data type, evaluate the following filter expressions
    And S_TOUCHES({queryable},BBOX(-180,-90,180,90))
    And S_TOUCHES({queryable},POLYGON((-180 -90,180 -90,180 90,-180 90,-180 -90)))
    And S_TOUCHES({queryable},LINESTRING(7 50,10 51))
    Then assert successful execution of the evaluation;
    And store the valid predicates for each data source.

  @test-34
  Scenario: A.9.5. Conformance Test 34 /conf/spatial-functions/s_crosses
    # ATS section: A.9.5
    # ATS id: /conf/spatial-functions/s_crosses
    # Requirements:
    #   /req/spatial-functions/spatial-functions , /req/spatial-functions/spatial-data-types
    # Purpose:
    #   Test the S_CROSSES spatial function
    Given One or more data sources, each with a list of queryables.
    When for each queryable {queryable} of type Point, MultiPoint, LineString or MultiLineString, evaluate the following filter expressions
    And S_CROSSES({queryable},BBOX(-180,-90,180,90))
    And S_CROSSES({queryable},POLYGON((-180 -90,180 -90,180 90,-180 90,-180 -90)))
    And S_CROSSES({queryable},LINESTRING(7 50,10 51))
    Then assert successful execution of the evaluation;
    And store the valid predicates for each data source.

  @test-35
  Scenario: A.9.6. Conformance Test 35 /conf/spatial-functions/s_within
    # ATS section: A.9.6
    # ATS id: /conf/spatial-functions/s_within
    # Requirements:
    #   /req/spatial-functions/spatial-functions , /req/spatial-functions/spatial-data-types
    # Purpose:
    #   Test the S_WITHIN spatial function
    Given One or more data sources, each with a list of queryables.
    When for each queryable {queryable} with a geometry data type, evaluate the following filter expressions
    And S_WITHIN({queryable},BBOX(-180,-90,180,90))
    And S_WITHIN({queryable},POLYGON((-180 -90,180 -90,180 90,-180 90,-180 -90)))
    And S_WITHIN({queryable},LINESTRING(7 50,10 51))
    And S_WITHIN({queryable},MULTIPOINT(7 50,10 51))
    Then assert successful execution of the evaluation;
    And assert that the two result sets of the first two filter expressions for each queryable are identical;
    And store the valid predicates for each data source.

  @test-36
  Scenario: A.9.7. Conformance Test 36 /conf/spatial-functions/s_contains
    # ATS section: A.9.7
    # ATS id: /conf/spatial-functions/s_contains
    # Requirements:
    #   /req/spatial-functions/spatial-functions , /req/spatial-functions/spatial-data-types
    # Purpose:
    #   Test the S_CONTAINS spatial function
    Given One or more data sources, each with a list of queryables.
    When for each queryable {queryable} with a geometry data type, evaluate the following filter expressions
    And S_CONTAINS({queryable},BBOX(-180,-90,180,90))
    And S_CONTAINS({queryable},POLYGON((-180 -90,180 -90,180 90,-180 90,-180 -90)))
    And S_CONTAINS({queryable},LINESTRING(7 50,10 51))
    And S_CONTAINS({queryable},MULTIPOINT(7 50,10 51))
    Then assert successful execution of the evaluation;
    And assert that the two result sets of the first two filter expressions for each queryable are identical;
    And assert that the results sets for each queryable do not have an item in common with the corresponding S_WITHIN expression;
    And store the valid predicates for each data source.

  @test-37
  Scenario: A.9.8. Conformance Test 37 /conf/spatial-functions/s_overlaps
    # ATS section: A.9.8
    # ATS id: /conf/spatial-functions/s_overlaps
    # Requirements:
    #   /req/spatial-functions/spatial-functions , /req/spatial-functions/spatial-data-types
    # Purpose:
    #   Test the S_OVERLAPS spatial function
    Given One or more data sources, each with a list of queryables.
    When For each queryable {queryable} of type Point or MultiPoint, evaluate the filter expression S_OVERLAPS({queryable},MULTIPOINT(7 50,10 51))
    And For each queryable {queryable} of type LineString or MultiLineString, evaluate the filter expression S_OVERLAPS({queryable},LINESTRING(7 50,10 51))
    And For each queryable {queryable} of type Polygon or MultiPolygon, evaluate the filter expression S_OVERLAPS({queryable},POLYGON((-180 -90,180 -90,180 90,-180 90,-180 -90)))
    Then assert successful execution of the evaluation;
    And store the valid predicates for each data source.

  @expected-fail @test-38
  Scenario: A.9.9. Conformance Test 38 /conf/spatial-functions/test-data
    # ATS section: A.9.9
    # ATS id: /conf/spatial-functions/test-data
    # Requirements:
    #   all requirements
    # Purpose:
    #   Test predicates against the test dataset
    Given The implementation under test uses the test dataset.
    When Evaluate each predicate in Predicates and expected results .
    Then assert successful execution of the evaluation;
    And assert that the expected result is returned;
    And store the valid predicates for each data source.

  @expected-fail @test-39
  Scenario: A.9.10. Conformance Test 39 /conf/spatial-functions/logical
    # ATS section: A.9.10
    # ATS id: /conf/spatial-functions/logical
    # Requirements:
    #   n/a
    # Purpose:
    #   Test filter expressions with AND, OR and NOT including sub-expressions
    Given The stored predicates for each data source, including from the dependencies.
    When For each data source, select at least 10 random combinations of four predicates ( {p1} to {p4} ) from the stored predicates and evaluate the filter expression ((NOT {p1} AND {p2}) OR ({p3} and NOT {p4}) or not ({p1} AND {p4})) .
    Then assert successful execution of the evaluation.
