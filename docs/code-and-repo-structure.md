# Code and Repository Structure

This document explains how gocql2 is organized. It is a maintainer guide, not an API promise.
The code itself is the source of truth.

## Architectural Shape

gocql2 follows a small ports-and-adapters shape:

1. `api` is the inner model. It defines filters, scalar values, metadata, and parse errors.
2. `gocql2` is the parser and serializer facade. It exposes the stable entry points.
3. `internal/parser` adapts CQL2 Text and CQL2 JSON into the `api` model.
4. `internal/serializer` adapts the `api` model back into CQL2 Text and CQL2 JSON.
5. `sql` adapts the `api` model into parameterized SQL fragments.

The dependency direction is intentional. Internal packages and adapters depend on `api`. The `api`
package does not depend on parser, serializer, SQL, database, HTTP, or CLI code.

## Public Packages

### Root Package

Files: `doc.go`, `parse.go`, `serialize.go`.

The root package is the small public facade. It exports `Parser`, `NewParser`, parser setup
methods, parser introspection methods, parse methods, and serialization functions.

The facade keeps callers away from `internal/parser` and `internal/serializer`. This gives the
maintainer room to change parse algorithms while preserving the imported API.

### API Package

Files: `api/ast.go`, `api/property.go`, `api/function.go`, `api/conformance.go`,
`api/errors.go`.

The `api` package defines the abstract filter representation and parse-time metadata. It should
stay free of storage concerns. If a future feature needs database behavior, put that behavior in
an adapter package and keep only the neutral shape in `api`.

The most important file is `api/ast.go`. It defines:

- `Node`, `Expression`, and `ScalarExpression` interfaces.
- Concrete expression structs for logical, comparison, spatial, temporal, array, and function
  forms.
- Literal, property, geometry, temporal, and array value structs.
- Operation constants that encode CQL2 operator names.

The schema files define property types, function signatures, conformance classes, and source
locations.

### SQL Package

Files: `sql/doc.go`, `sql/sql.go`.

The `sql` package is an outbound adapter. It accepts `api.Expression` and emits `sql.SQL`, which
contains fragment text and bind arguments.

The package has three public extension seams:

1. `PropertyResolver` maps CQL2 property references to trusted SQL expressions.
2. `Dialect` renders placeholders, identifiers, literals, functions, and predicate families.
3. `Option` values configure the compiler for each `ToSQL` call.

The built-in `PostGISDialect` is an adapter for PostgreSQL and PostGIS. `BaseDialect` is the
starting point for custom dialects.

## Internal Packages

### Internal Parser

Directory: `internal/parser`.

The parser owns both concrete CQL2 syntaxes. `text_lexer.go` tokenizes CQL2 Text. `text_parser.go`
parses tokens into public `api` nodes. `json_parser.go` parses CQL2 JSON objects into the same
public nodes.

Support files keep concerns separate:

- `parse.go` holds parser configuration and reusable parser methods.
- `property.go` validates property registration, property use, and operand compatibility.
- `function.go` validates function registration, argument count, argument type, and return type.
- `conformance.go` tracks which optional CQL2 features are enabled.
- `spatial.go`, `temporal.go`, and `array.go` parse feature-specific literal and predicate forms.
- `number.go` preserves numeric literal text to avoid float rounding during parse.
- `parse_error.go` builds `api.ParseError` values with text or JSON locations.

Keep parse-time validation here when the rule is about valid CQL2 syntax or valid CQL2 semantics.
Move policy validation to an application layer if the rule is product-specific.

### Internal Serializer

Directory: `internal/serializer`.

The serializer walks the public AST and emits CQL2 Text or CQL2 JSON. It is intentionally internal
because callers should depend on `gocql2.SerializeText` and `gocql2.SerializeJSON`.

When adding a new public AST node, update both serializer targets or return a clear error for the
unsupported target.

### Internal ATS Tests

Directory: `internal/ats`.

This package contains CQL2 Abstract Test Suite inspired fixture tests and PostGIS-backed SQL tests.
The feature files live under `internal/ats/features/ats`.

Unit tests close to parser, API, serializer, and SQL code cover focused behavior. ATS tests cover
cross-feature conformance and fixture-backed flows.

## Main Data Flow

A normal integration has this flow:

```text
client CQL2 input
    -> gocql2.Parser
    -> internal/parser
    -> api.Expression
    -> application service or domain filter value
    -> sql.ToSQL in a repository adapter
    -> database driver with Text and Args
```

Serialization follows a shorter path:

```text
api.Expression
    -> gocql2.SerializeText or gocql2.SerializeJSON
    -> internal/serializer
    -> CQL2 output
```

## Public AST as a Hexagonal Boundary

Use `api.Expression` as a boundary value when CQL2 is part of the application contract. For
example, an HTTP adapter can parse a `filter` query parameter and pass an `api.Expression` into an
application service. The application service can combine that value with authorization rules and
call a repository port.

Do not let `sql.SQL`, `sql.Dialect`, or `sql.PropertyResolver` cross inward into the domain model.
Those types are persistence adapter details.

For domain-rich filters, translate the AST near the application boundary:

```text
api.Expression -> SearchCriteria -> repository port -> SQL adapter
```

Use this translation when domain language matters more than CQL2 operator fidelity.

## Change Guide

Use these paths for common changes:

| Change | Edit First | Then Check |
| --- | --- | --- |
| New public node | `api/ast.go` | Parser, serializer, SQL, tests |
| New property type | `api/property.go` | `internal/parser/property.go`, SQL tests |
| New function type | `api/function.go` | `internal/parser/function.go`, serializer |
| New conformance class | `api/conformance.go` | `internal/parser/conformance.go`, ATS tests |
| New text syntax | `internal/parser/text_parser.go` | Text parser tests, serializer |
| New JSON syntax | `internal/parser/json_parser.go` | JSON parser tests, serializer |
| New SQL feature | `sql/sql.go` | SQL unit tests and ATS SQL tests |

## Testing Layout

Run fast unit coverage with `task test:unit`. This covers package tests outside the full ATS suite.

Run CQL2 fixture coverage with `task test:ats`. This executes the ATS-inspired Gherkin suite.

Run the full check before publishing a change with `task check`. The task combines Go lint,
Markdown lint, GitHub Actions lint, unit tests, and ATS tests through `Taskfile.yml`.

## Documentation Style

Documentation should follow the agent-style project rules at
<https://github.com/yzhao062/agent-style>. In practice, write concrete sentences, keep factual
claims tied to code paths, avoid casual em dashes, use title case headings, and avoid filler
transitions.
