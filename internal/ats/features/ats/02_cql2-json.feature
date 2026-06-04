# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @cql2-json
Feature: A.2 CQL2 JSON abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.

  @test-3
  Scenario: A.2.1. Conformance Test 3 /conf/cql2-json/validate
    # ATS section: A.2.1
    # ATS id: /conf/cql2-json/validate
    # Requirements:
    #   all requirements
    # Purpose:
    #   Validate that CQL2 JSON is supported by the server
    When I parse the CQL2 JSON filter:
      """
      {
        "op": "and",
        "args": [
          { "op": "=", "args": [{ "property": "name" }, "alice"] },
          { "op": ">=", "args": [{ "property": "height" }, 1] }
        ]
      }
      """
    Then parsing succeeds
