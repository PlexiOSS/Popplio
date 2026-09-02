// Copyright (C) 2026 NodeByte LTD

package resp

import (
	"net/http"

	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/ratelimit"
	"github.com/PlexiOSS/Keel/uapi"
)

func Err(reason string, err error, fields ...zap.Field) uapi.HttpResponse {
	state.Logger.Error(reason, append([]zap.Field{zap.Error(err)}, fields...)...)
	return uapi.DefaultResponse(http.StatusInternalServerError)
}

func ErrDetail(reason string, err error, fields ...zap.Field) uapi.HttpResponse {
	state.Logger.Error(reason, append([]zap.Field{zap.Error(err)}, fields...)...)
	return Status(http.StatusInternalServerError, reason+".")
}

func ErrBody(reason, message string, err error, fields ...zap.Field) uapi.HttpResponse {
	state.Logger.Error(reason, append([]zap.Field{zap.Error(err)}, fields...)...)
	return Status(http.StatusInternalServerError, message)
}

func Status(status int, message string) uapi.HttpResponse {
	return uapi.HttpResponse{
		Status: status,
		Json:   types.ApiError{Message: message},
	}
}

func BadRequest(message string) uapi.HttpResponse {
	return Status(http.StatusBadRequest, message)
}

func NotFound(message string) uapi.HttpResponse {
	return Status(http.StatusNotFound, message)
}

func Forbidden(message string) uapi.HttpResponse {
	return Status(http.StatusForbidden, message)
}

func Unauthorized(message string) uapi.HttpResponse {
	return Status(http.StatusUnauthorized, message)
}

func Conflict(message string) uapi.HttpResponse {
	return Status(http.StatusConflict, message)
}

func RateLimited(limit ratelimit.Limit) uapi.HttpResponse {
	return uapi.HttpResponse{
		Status:  http.StatusTooManyRequests,
		Json:    types.ApiError{Message: "You are being ratelimited. Please try again in " + limit.TimeToReset.String()},
		Headers: limit.Headers(),
	}
}

func WithHeaders(res uapi.HttpResponse, headers map[string]string) uapi.HttpResponse {
	if len(headers) == 0 {
		return res
	}

	merged := make(map[string]string, len(headers)+len(res.Headers))
	for k, v := range headers {
		merged[k] = v
	}
	for k, v := range res.Headers {
		merged[k] = v
	}

	res.Headers = merged
	return res
}

func OK(body any) uapi.HttpResponse {
	return uapi.HttpResponse{Json: body}
}

func Created(body any) uapi.HttpResponse {
	return uapi.HttpResponse{
		Status: http.StatusCreated,
		Json:   body,
	}
}

func NoContent() uapi.HttpResponse {
	return uapi.DefaultResponse(http.StatusNoContent)
}
