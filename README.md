# gocql2

[![codecov][codecov-badge]][codecov]

gocql2 is a Go library for parsing [OGC Common Query Language 2 (CQL2)][cql2]
filters and compiling them into safe parameterized SQL fragments.

Use it when an API accepts CQL2 filters and must validate them against known queryable fields
before applying them to a datastore.

## Install

```sh
go get github.com/cwygoda/gocql2
```

## Quick Start

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

## Documentation

- [Usage Guide](./docs/usage.md): parse, serialize, validate, and compile CQL2.
- [Public API Reference](./docs/public-api-reference.md): exported packages and public AST.
- [Code and Repository Structure](./docs/code-and-repo-structure.md): architecture and internals.
- [Development Guide](./DEVELOPMENT.md): local tooling, tests, linting, and commits.
- [References](./REFERENCES.md): OGC CQL2 specification links.

## Supported Features

- CQL2 Text and CQL2 JSON parsing.
- Logical expressions, comparisons, `LIKE`, `BETWEEN`, `IN`, `IS NULL`, arithmetic, and literals.
- Standard CQL2 spatial, temporal, and array predicates when enabled by conformance.
- Typed public AST in `github.com/cwygoda/gocql2/api`.
- Parameterized SQL generation with explicit property mapping.

[codecov]: https://codecov.io/github/cwygoda/gocql2
[codecov-badge]: https://codecov.io/github/cwygoda/gocql2/graph/badge.svg?token=18FRBD1HD4
[cql2]: https://docs.ogc.org/is/21-065r2/21-065r2.html
