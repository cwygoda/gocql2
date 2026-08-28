# Usage Guide

This guide shows the common integration paths for gocql2: parse CQL2, validate queryables,
serialize parsed filters, and compile filters to SQL.

## Install

```sh
go get github.com/cwygoda/gocql2
```

## Parse CQL2 Text

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
round-tripping. It is structurally equivalent to the original input, but not byte-for-byte
identical.

```go
text, err := gocql2.SerializeText(expr)
jsonBytes, err := gocql2.SerializeJSON(expr)
```

## Compile CQL2 to SQL

The `sql` package turns a parsed AST into a parameterized SQL expression. Property mappings fail
closed by default. Every CQL2 property must be explicitly mapped to trusted application-authored
SQL.

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
(((lower("assets"."name") LIKE lower($1))) AND
(("assets"."height" >= CAST($2 AS numeric))))
[]interface {}{"oak%", "10"}
```

Compose `where.Text` into your query and pass `where.Args` to your database driver.

## Validate Queryables and Functions

A reusable parser can be configured before concurrent use:

- `WithAllowedProperties` rejects unknown properties and validates property types in scalar,
  comparison, temporal, spatial, array, and function contexts.
- `WithAllowedFunctions` rejects unknown functions and validates function signatures.
- `WithConformance` records CQL2 conformance classes and enables standard CQL2 functions implied
  by those classes, such as `CASEI`, spatial predicates, temporal predicates, and array predicates.
- `WithMaxDepth` limits recursive parse depth for defensive parsing.

## SQL Dialects

gocql2 includes:

- `cql2sql.BaseDialect` for ANSI-style placeholders and identifier quoting.
- `cql2sql.PostGISDialect` for PostgreSQL and PostGIS placeholders, case and accent functions,
  spatial predicates, temporal predicates, array predicates, and geometry literals.

Implement `cql2sql.Dialect` or embed `cql2sql.BaseDialect` to customize database-specific
rendering.

## Error Handling

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

SQL generation errors are regular Go errors. For example, SQL generation fails when a property has
no SQL mapping or a dialect does not support a requested function.
