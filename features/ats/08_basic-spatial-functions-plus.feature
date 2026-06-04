# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @basic-spatial-functions-plus
Feature: A.8 Basic Spatial Functions with additional Spatial Literals abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.

  @test-28
  Scenario: A.8.1. Conformance Test 28 /conf/basic-spatial-functions-plus/s_intersects
    # ATS section: A.8.1
    # ATS id: /conf/basic-spatial-functions-plus/s_intersects
    # Requirements:
    #   /req/basic-spatial-functions-plus/spatial-predicate , /req/basic-spatial-functions-plus/spatial-functions , /req/basic-spatial-functions-plus/spatial-data-types
    # Purpose:
    #   Test the S_INTERSECTS spatial comparison function with points, multi-points, line strings, multi-line string, polygons, multi-polygons, geometry collections and bounding boxes.
    Given One or more data sources, each with a list of queryables.
    And At least one queryable has a geometry data type.
    When For each queryable {queryable} with a geometry data type, evaluate the following filter expressions
    And S_INTERSECTS({queryable},MULTIPOINT(7.02 49.92, 90 180))
    And S_INTERSECTS({queryable},LINESTRING(-180 -45, 0 -45))
    And S_INTERSECTS({queryable},MULTILINESTRING((-180 -45, 0 -45), (0 45, 180 45)))
    And S_INTERSECTS({queryable},POLYGON((-180 -90, -90 -90, -90 90, -180 90, -180 -90), (-120 -50, -100 -50, -100 -40, -120 -40, -120 -50)))
    And S_INTERSECTS({queryable},MULTIPOLYGON(((-180 -90, -90 -90, -90 90, -180 90, -180 -90), (-120 -50, -100 -50, -100 -40, -120 -40, -120 -50)),((0 0, 10 0, 10 10, 0 10, 0 0))))
    And S_INTERSECTS({queryable},GEOMETRYCOLLECTION(POINT(7.02 49.92), POLYGON((0 0, 10 0, 10 10, 0 10, 0 0))))
    Then assert successful execution of the evaluation for all filter expressions except the first;
    And assert unsuccessful execution of the evaluation for the first filter expressions (invalid coordinate);
    And store the valid predicates for each data source.

  @test-29
  Scenario: A.8.2. Conformance Test 29 /conf/basic-spatial-functions-plus/test-data
    # ATS section: A.8.2
    # ATS id: /conf/basic-spatial-functions-plus/test-data
    # Requirements:
    #   all requirements
    # Purpose:
    #   Test predicates against the test dataset
    Given The implementation under test uses the test dataset.
    When Evaluate each predicate in Predicates and expected results .
    Then assert successful execution of the evaluation;
    And assert that the expected result is returned.
