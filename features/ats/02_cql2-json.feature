# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @cql2-json
Feature: A.2 CQL2 JSON abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.
  The expected-fail tag marks ATS entries not yet covered by regular package tests.

  @expected-fail @test-3
  Scenario: A.2.1. Conformance Test 3 /conf/cql2-json/validate
    # ATS section: A.2.1
    # ATS id: /conf/cql2-json/validate
    # Requirements:
    #   all requirements
    # Purpose:
    #   Validate that CQL2 JSON is supported by the server
    Given A filter expression
    When Execute conformance tests for all supported conformance classes with the parameter "Filter Language". Use the value "CQL2 JSON". Note that the filter expressions in the test cases have to be converted to a CQL2 JSON representation.
    Then assert the validation is successful.
