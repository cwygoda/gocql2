# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @cql2-text
Feature: A.1 CQL2 Text abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.
  The expected-fail tag marks ATS entries not yet covered by regular package tests.

  @expected-fail @test-1
  Scenario: A.1.1. Conformance Test 1 /conf/cql2-text/validate
    # ATS section: A.1.1
    # ATS id: /conf/cql2-text/validate
    # Requirements:
    #   all requirements
    # Purpose:
    #   Validate that CQL2 Text is supported by the server
    Given n/a
    When Execute conformance tests for all supported conformance classes with the parameter "Filter Language". Use the value "CQL2 Text".
    Then assert that all conformance tests are successful.

  @expected-fail @test-2
  Scenario: A.1.2. Conformance Test 2 /conf/cql2-text/escaping
    # ATS section: A.1.2
    # ATS id: /conf/cql2-text/escaping
    # Requirements:
    #   /req/cql2-text/escaping
    # Purpose:
    #   Test escaping in string literals.
    Given One or more data sources containing string literals with embedded single quotation ( ' ) and/or BELL, and/or BACKSPACE, and/or HORIZONTAL TAB, and/or NEWLINE, and/or VERTICAL TAB, and/or FORM FEED, and/or CARRIAGE RETURN characters.
    When Decode each string literal.
    Then assert that the escaped embedded characters have been correctly recovered.
