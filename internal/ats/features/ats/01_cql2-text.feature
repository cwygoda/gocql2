# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @cql2-text
Feature: A.1 CQL2 Text abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.

  @test-1
  Scenario Outline: A.1.1. Conformance Test 1 /conf/cql2-text/validate
    # ATS section: A.1.1
    # ATS id: /conf/cql2-text/validate
    # Requirements:
    #   all requirements
    # Purpose:
    #   Validate that CQL2 Text is supported by the server
    When I parse the CQL2 Text filter "<filter>"
    Then parsing succeeds

    Examples:
      | filter                         |
      | name = 'alice' AND height >= 1 |

  @test-2
  Scenario Outline: A.1.2. Conformance Test 2 /conf/cql2-text/escaping
    # ATS section: A.1.2
    # ATS id: /conf/cql2-text/escaping
    # Requirements:
    #   /req/cql2-text/escaping
    # Purpose:
    #   Test escaping in string literals.
    When I parse the CQL2 Text filter "<filter>"
    Then the comparison right literal is "<value>"

    Examples:
      | filter             | value   |
      | name = 'O''Brien'  | O'Brien |
      | name = 'O\'Brien' | O'Brien |
