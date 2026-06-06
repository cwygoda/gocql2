# gocql2 - OGC CQL2 parser with SQL generation

[![codecov][codecov-badge]][codecov]

gocql2 is a Go library for parsing [OGC Common Query Language 2 (CQL2)][cql2]
filters and, when needed, compiling them into safe parameterized SQL fragments.

Use it when you accept CQL2 from API clients, such as OGC API Features `filter`
parameters, and need to validate the filter against your queryable fields before applying it to a
datastore.

## Install

```sh
go get github.com/cwygoda/gocql2
```

## Quick start: parse CQL2 Text

```go
package main

import (
    "fmt"
    "log"

    gocql2 "github.com/cwygoda/gocql2"
    "github.com/cwygoda/gocql2/api"
)

func main() {
    expr, err := gocql2.NewParser().
        WithAllowedProperties(
            api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString},
            api.PropertyDefinition{Name: "height", Type: api.PropertyTypeNumber},
        ).
        ParseText("name = 'Oak' AND height >= 10")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("%T\n", expr) // *api.LogicalExpression
}
```

For parsing without schema validation, still create an explicit parser with
`gocql2.NewParser()` and then call `ParseText`, `ParseJSON`, or `Parse`.

## Parse CQL2 JSON

```go
expr, err := gocql2.NewParser().
    WithAllowedProperties(
        api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString},
        api.PropertyDefinition{Name: "height", Type: api.PropertyTypeNumber},
    ).
    ParseJSON([]byte(`{
        "op": "and",
        "args": [
            {"op": "=", "args": [{"property": "name"}, "Oak"]},
            {"op": ">=", "args": [{"property": "height"}, 10]}
        ]
    }`))
```

## Serialize CQL2

Parsed ASTs can be serialized back to CQL2 Text or CQL2 JSON. Output is canonicalized for safe
round-tripping; it is structurally equivalent, not byte-for-byte identical to the original input.

```go
text, err := gocql2.SerializeText(expr)
jsonBytes, err := gocql2.SerializeJSON(expr)
```

## Compile CQL2 to SQL

The `sql` package turns a parsed AST into a parameterized SQL expression. Property mappings are
fail-closed by default: every CQL2 property must be explicitly mapped to trusted
application-authored SQL.

```go
package main

import (
    "fmt"
    "log"

    gocql2 "github.com/cwygoda/gocql2"
    "github.com/cwygoda/gocql2/api"
    cql2sql "github.com/cwygoda/gocql2/sql"
)

func main() {
    props := []cql2sql.Property{
        {Name: "name", Type: api.PropertyTypeString, Expr: cql2sql.Column("assets", "name")},
        {Name: "height", Type: api.PropertyTypeNumber, Expr: cql2sql.Column("assets", "height")},
    }

    expr, err := gocql2.NewParser().
        WithConformance(
            api.ConformanceAdvancedComparisonOperators,
            api.ConformanceCaseInsensitiveComparison,
        ).
        WithAllowedProperties(cql2sql.PropertyDefinitions(props...)...).
        ParseText("CASEI(name) LIKE casei('oak%') AND height >= 10")
    if err != nil {
        log.Fatal(err)
    }

    where, err := cql2sql.ToSQL(
        expr,
        cql2sql.PostGISDialect(),
        cql2sql.WithSQLProperties(props...),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(where.Text)
    fmt.Printf("%#v\n", where.Args)
}
```

Output:

```text
(((lower("assets"."name") LIKE lower($1))) AND (("assets"."height" >= CAST($2 AS numeric))))
[]interface {}{"oak%", "10"}
```

You can then compose `where.Text` into your query and pass `where.Args` to your database driver.

## Validate queryables and functions

A reusable parser can be configured before concurrent use:

- `WithAllowedProperties` rejects unknown properties and validates property types in scalar,
  comparison, temporal, spatial, array, and function contexts.
- `WithAllowedFunctions` rejects unknown functions and validates function signatures.
- `WithConformance` records CQL2 conformance classes and enables standard CQL2 functions implied
  by those classes, such as `CASEI`, spatial predicates, temporal predicates, and array predicates.
- `WithMaxDepth` limits recursive parse depth for defensive parsing.

## SQL dialects

gocql2 includes:

- `cql2sql.BaseDialect` for ANSI-style placeholders and identifier quoting.
- `cql2sql.PostGISDialect` for PostgreSQL/PostGIS placeholders, case/accent functions, spatial
  predicates, temporal predicates, array predicates, and geometry literals.

Implement `cql2sql.Dialect` or embed `cql2sql.BaseDialect` to customize database-specific
rendering.

## Error handling

Parser errors are returned as `*api.ParseError` and include source language plus either text
position or JSON path information.

```go
_, err := gocql2.NewParser().ParseText("name =")
if err != nil {
    var parseErr *api.ParseError
    if errors.As(err, &parseErr) {
        log.Printf(
            "bad CQL2 at line %d, column %d",
            parseErr.Location.Line,
            parseErr.Location.Column,
        )
    }
}
```

SQL generation errors are regular Go errors, for example when a property has no SQL mapping or a
dialect does not support a requested function.

## Supported input and features

- CQL2 Text and CQL2 JSON parsing.
- Logical expressions, comparisons, `LIKE`, `BETWEEN`, `IN`, `IS NULL`, arithmetic, and
  boolean/null/string/number literals.
- Standard CQL2 spatial, temporal, and array predicates when enabled by conformance.
- Typed public AST in the `api` package.
- Parameterized SQL fragment generation with explicit property mapping.

See [REFERENCES.md](./REFERENCES.md) for CQL2 references and [DEVELOPMENT.md](./DEVELOPMENT.md)
for contributor setup.

[codecov]: https://codecov.io/github/cwygoda/gocql2
[codecov-badge]: https://codecov.io/github/cwygoda/gocql2/graph/badge.svg?token=18FRBD1HD4
[cql2]: https://docs.ogc.org/is/21-065r2/21-065r2.html
