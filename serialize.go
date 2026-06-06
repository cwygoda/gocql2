package gocql2

import (
	"github.com/cwygoda/gocql2/api"
	"github.com/cwygoda/gocql2/internal/serializer"
)

// SerializeText serializes a CQL2 AST to CQL2 Text.
func SerializeText(expr api.Expression) (string, error) { return serializer.ToText(expr) }

// SerializeJSON serializes a CQL2 AST to CQL2 JSON.
func SerializeJSON(expr api.Expression) ([]byte, error) { return serializer.ToJSON(expr) }
