// Copyright (C) 2026 NodeByte LTD

package constants

const (
	ResourceNotFound    = "{\"message\":\"The requested resource could not be found.\"}"
	EndpointNotFound    = "{\"message\":\"This endpoint does not exist. Check the request path and try again.\"}"
	BadRequest          = "{\"message\":\"The request was malformed or invalid.\"}"
	Forbidden           = "{\"message\":\"You do not have permission to perform this action.\"}"
	Unauthorized        = "{\"message\":\"Authentication is required. Check that a valid API token was provided.\"}"
	InternalServerError = "{\"message\":\"An unexpected error occurred while processing the request. Please try again later.\"}"
	MethodNotAllowed    = "{\"message\":\"This HTTP method is not supported for this endpoint.\"}"
	BodyRequired        = "{\"message\":\"A request body is required for this endpoint.\"}"
	BackTick            = "`"
	DoubleBackTick      = "``"
)
