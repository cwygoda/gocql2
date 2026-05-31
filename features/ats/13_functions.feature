# Generated from OGC 21-065r2 Annex A: Abstract Test Suite (Normative).
# Source: https://docs.ogc.org/is/21-065r2/21-065r2.html#ats
@cql2-ats @functions
Feature: A.13 Functions abstract conformance tests
  The scenarios mirror the normative CQL2 Abstract Test Suite test methods directly.
  The expected-fail tag marks ATS entries not yet covered by regular package tests.

  @expected-fail @test-51
  Scenario: A.13.1. Conformance Test 51 /conf/functions/functions
    # ATS section: A.13.1
    # ATS id: /conf/functions/functions
    # Requirements:
    #   /req/functions/functions
    # Purpose:
    #   Test predicates with functions
    Given The list of functions with arguments and return type supported by the implementation under test.
    When For each function construct multiple valid filter expressions involving different operators.
    Then assert successful execution of the evaluation.
