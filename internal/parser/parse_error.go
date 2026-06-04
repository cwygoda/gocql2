package parser

import "github.com/cwygoda/cql2/api"

func parseError(source api.Language, loc api.Location, message string, expected ...string) *api.ParseError {
	return &api.ParseError{Source: source, Location: loc, Message: message, Expected: expected}
}
