# Public API Reference

This document is a starter reference for the public surface of gocql2. It describes what callers
can import, what each package owns, and how the public AST can fit into a hexagonal architecture.
The concrete source for this reference is the exported Go code in `parse.go`, `serialize.go`,
`api/*.go`, and `sql/*.go`.

## Package Map

`github.com/cwygoda/gocql2` is the entry package. It owns parsing and serialization entry points.
It depends on the internal parser and serializer packages, but those internals do not leak through
its exported types.

`github.com/cwygoda/gocql2/api` owns the public CQL2 model. Its types describe filters,
properties, functions, conformance classes, source locations, and parse errors.

`github.com/cwygoda/gocql2/sql` owns SQL generation. It accepts an `api.Expression`, a `Dialect`,
and trusted property mappings. It returns a parameterized SQL fragment.

## Parser API

Create parsers with `gocql2.NewParser()`.

```go
parser := gocql2.NewParser().
    WithAllowedProperties(
        api.PropertyDefinition{Name: "name", Type: api.PropertyTypeString},
        api.PropertyDefinition{Name: "height", Type: api.PropertyTypeNumber},
    ).
    WithConformance(api.ConformanceAdvancedComparisonOperators)
```

Parser setup methods mutate the parser and return it for chaining. Configure a parser before
sharing it between goroutines.

The parser exposes these setup methods:

| Method | Purpose |
| --- | --- |
| `WithMaxDepth(n int)` | Sets a recursion limit for defensive parsing. |
| `WithAllowedProperties(defs ...api.PropertyDefinition)` | Rejects unknown properties. |
| `WithAllowedFunctions(defs ...api.FunctionDefinition)` | Rejects unknown functions. |
| `WithConformanceClasses(classes ...string)` | Records advertised class IDs only. |
| `WithConformance(classes ...string)` | Records classes and adds standard functions. |

The parser exposes these read methods:

| Method | Purpose |
| --- | --- |
| `SupportedProperties()` | Returns advertised property names. |
| `SupportedPropertyDefinitions()` | Returns configured property definitions. |
| `SupportedFunctions()` | Returns advertised function names. |
| `SupportedFunctionDefinitions()` | Returns configured function definitions. |
| `ConformanceClasses()` | Returns advertised conformance class IDs. |

Use these parse methods for input:

| Method | Input | Result |
| --- | --- | --- |
| `ParseText(input string)` | CQL2 Text | `api.Expression` |
| `ParseJSON(input []byte)` | CQL2 JSON | `api.Expression` |
| `Parse(input []byte, lang api.Language)` | Text or JSON | `api.Expression` |

## Serialization API

`SerializeText(expr api.Expression)` emits canonical CQL2 Text. `SerializeJSON(expr
api.Expression)` emits canonical CQL2 JSON. The serializer accepts the public AST, so callers can
parse, inspect, transform, and serialize without depending on internals.

Serialization returns an error when a node cannot be represented in the target encoding. The
serializer code in `internal/serializer/serializer.go` contains the exact supported node cases.

## Public Abstract Filter Representation

The public filter representation is the AST in `api/ast.go`. The root type for a filter is
`api.Expression`. The root type for a scalar value is `api.ScalarExpression`. Both interfaces embed
`api.Node`, which exposes `Span() api.Span`.

The concrete public nodes are plain Go structs:

| Family | Types |
| --- | --- |
| Boolean filters | `LogicalExpression`, `ComparisonExpression`, `LikeExpression` |
| More predicates | `BetweenExpression`, `InExpression`, `IsNullExpression` |
| Spatial predicates | `SpatialPredicateExpression`, `GeometryLiteral`, `Coordinate` |
| Temporal predicates | `TemporalPredicateExpression`, `TemporalInstant`, `TemporalInterval` |
| Array predicates | `ArrayPredicateExpression`, `ArrayLiteral` |
| Scalar values | `PropertyRef`, `Literal`, `FunctionCall`, `ArithmeticExpression` |

The representation is public in two senses. Application code can read it after parsing, and tests
or adapters can construct it directly. The marker methods on `Expression` and `ScalarExpression`
keep external packages from adding new implementations of those interfaces. External code can
compose the exported concrete nodes that already implement the interfaces.

## Domain Model Guidance

In hexagonal architecture terms, `api.Expression` works well as an application-level filter value
when the product accepts CQL2 as its filter language. It is independent of HTTP, SQL, filesystems,
and databases. That makes it safe to pass through an application port such as:

```go
type AssetSearchFilter struct {
    CQL api.Expression
}
```

Keep `api.Expression` out of persistence-specific code until an adapter boundary. A repository or
query adapter can compile it with `sql.ToSQL`, while the application service stays tied to the
abstract filter rather than a SQL string.

Translate `api.Expression` into your own domain type when the domain has terms that CQL2 does not
express. For example, a policy rule named `VisibleToTenant` should be a domain concept, not a raw
property comparison that every caller must remember.

The AST structs are mutable. Treat them as request-scoped values or copy them before rewriting.
Do not store global mutable AST values and share them across requests.

## Schema and Function Metadata

`api.PropertyDefinition` describes one allowed property. `PropertyTypeAny` permits use in any
scalar context when only a name allow-list exists. Specific property types let the parser reject
invalid comparisons, scalar use, spatial use, temporal use, array use, and function arguments.

`api.FunctionDefinition` describes one allowed function. Each definition has a normalized name,
positional arguments, optional variadic final argument, and at least one return type.

Use helpers for standard functions:

```go
api.CaseIFunction()
api.AccentiFunction()
api.StandardTextFunctions()
api.StandardFunctionsForConformance(classes...)
```

## Conformance Classes

The `api` package exports CQL2 conformance class constants, such as
`ConformanceAdvancedComparisonOperators`, `ConformanceSpatialFunctions`, and
`ConformanceCQL2JSON`.

Use `api.CanonicalConformanceClasses(classes...)` to normalize class IDs. It accepts full CQL2
conformance URIs, requirements URIs, `/conf/<class>` fragments, `/req/<class>` fragments, and class
slugs.

Use parser `WithConformance(classes...)` when standard functions should become valid during parse.
Use `WithConformanceClasses(classes...)` when the parser should advertise classes without changing
its function registry.

## SQL Generation API

`sql.ToSQL(expr, dialect, opts...)` compiles an `api.Expression` into:

```go
type SQL struct {
    Text string
    Args []any
}
```

The generated `Text` is a SQL expression fragment. The generated `Args` slice contains bind values
in placeholder order.

The safest setup maps every CQL2 property to trusted application-authored SQL:

```go
props := []cql2sql.Property{
    {Name: "name", Type: api.PropertyTypeString, Expr: cql2sql.Column("assets", "name")},
}

where, err := cql2sql.ToSQL(
    expr,
    cql2sql.PostGISDialect(),
    cql2sql.WithSQLProperties(props...),
)
```

`sql.Column(parts...)` builds a dialect-quoted identifier. `sql.RawSQL(text)` accepts a trusted SQL
expression. Never build `RawSQL` from CQL2 input or other untrusted user input.

SQL generation fails closed by default. If no resolver maps a property, `ToSQL` returns an error
wrapping `sql.ErrNoSQLMapping`. `WithDefaultColumnMapping()` changes that behavior and maps
unresolved properties to quoted columns with the same name.

## SQL Extension Points

Implement `sql.Dialect` to customize database rendering. Embed `sql.BaseDialect` when only a few
constructs differ from the conservative defaults.

Implement `sql.PropertyResolver` to map properties dynamically. Return `sql.ErrNoSQLMapping` when
the resolver has no mapping and should allow fallback behavior configured by
`WithDefaultColumnMapping()`.

Use `sql.PostGISDialect()` for PostgreSQL and PostGIS. It provides `$1` placeholders, numeric and
temporal casts, text functions, spatial predicates, temporal predicates, array predicates, and
geometry literal rendering. `sql.WithSRID(srid)` changes the SRID for geometry literals.

## Error Types

CQL2 syntax and semantic parse failures return `*api.ParseError`. It records the source language,
human message, optional wrapped cause, expected tokens or values, and source location.

Text locations use line, column, byte offset, and character offset. JSON semantic locations use
`api.JSONPath`, which formats as strings such as `$`, `$.args[0]`, or `$["odd-key"]`.

Use `errors.As` to inspect parse errors:

```go
var parseErr *api.ParseError
if errors.As(err, &parseErr) {
    // read parseErr.Source and parseErr.Location
}
```
