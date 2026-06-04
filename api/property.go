package api

// PropertyType identifies the advertised data type for a queryable property.
type PropertyType string

// Supported property data types. The values align with the CQL2/OGC API
// queryables type names, lower-cased for API consistency.
const (
	// PropertyTypeAny permits the property in any scalar context. It is used by
	// name-only property allow-lists where no type information is available.
	PropertyTypeAny PropertyType = ""

	PropertyTypeString             PropertyType = "string"
	PropertyTypeNumber             PropertyType = "number"
	PropertyTypeInteger            PropertyType = "integer"
	PropertyTypeBoolean            PropertyType = "boolean"
	PropertyTypeDate               PropertyType = "date"
	PropertyTypeTimestamp          PropertyType = "timestamp"
	PropertyTypeInterval           PropertyType = "interval"
	PropertyTypePoint              PropertyType = "point"
	PropertyTypeMultiPoint         PropertyType = "multipoint"
	PropertyTypeLineString         PropertyType = "linestring"
	PropertyTypeMultiLineString    PropertyType = "multilinestring"
	PropertyTypePolygon            PropertyType = "polygon"
	PropertyTypeMultiPolygon       PropertyType = "multipolygon"
	PropertyTypeGeometry           PropertyType = "geometry"
	PropertyTypeGeometryCollection PropertyType = "geometrycollection"
	PropertyTypeArray              PropertyType = "array"
)

// PropertyDefinition describes one allowed queryable property.
type PropertyDefinition struct {
	Name string
	Type PropertyType
}
