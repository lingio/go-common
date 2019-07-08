package middleware

// FIMXE: This should be moved to go-commons/util or something similar

import (
	"go.opencensus.io/trace"
)

func HTTPStatusCodeToGoogleAPIErrorCode(httpCode int) int {
	if httpCode >= 200 && httpCode < 300 {
		return trace.StatusCodeOK
	}
	switch httpCode {
	case 400:
		return trace.StatusCodeOutOfRange
	case 401:
		return trace.StatusCodeUnauthenticated
	case 403:
		return trace.StatusCodePermissionDenied
	case 404:
		return trace.StatusCodeNotFound
	case 409:
		return trace.StatusCodeAlreadyExists
		// NOTE: StatusCodeAborted also maps to the HTTP status code "409 Conflict"
		// We choose to only return AlreadyExists as it more closely matches
	case 429:
		return trace.StatusCodeResourceExhausted
	case 499:
		return trace.StatusCodeCancelled
	case 500:
		return trace.StatusCodeInternal
	case 501:
		return trace.StatusCodeUnimplemented
	case 503:
		return trace.StatusCodeUnavailable
	case 504:
		return trace.StatusCodeDeadlineExceeded
	default:
		// If this code doesn't map we just don't map it
		return httpCode
	}
}
