package logicerr

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
)

// Error describes a HTTP error message
type Error struct {
	Message        string
	HTTPStatusCode int
	Trace          string
	InfoMap        map[string]string
}

func (e *Error) Error() string {
	return e.Message
}

// NewInternalError returns a new Error with the InternalServerError http error code
func NewInternalError(msg string, m map[string]string) *Error {
	m = ensureMapNotNil(m)
	return &Error{
		Message:        msg,
		HTTPStatusCode: http.StatusInternalServerError,
		Trace:          getErrorTrace(),
		InfoMap:        m,
	}
}

// NewInternalErrorE returns a new Error with the InternalServerError http error code
func NewInternalErrorE(contextMsg string, err error, m map[string]string) *Error {
	m = ensureMapNotNil(m)
	m["error"] = fmt.Sprintf("%v", err)
	return &Error{
		Message:        fmt.Sprintf("%s: %s", contextMsg, err.Error()),
		HTTPStatusCode: http.StatusInternalServerError,
		Trace:          getErrorTrace(),
		InfoMap:        m,
	}
}

// NewNotFoundError returns a new Error with the StatusNotFound http error code
func NewNotFoundError(typeName string, thingNotFound string, m map[string]string) *Error {
	m = ensureMapNotNil(m)
	m["notFound"] = thingNotFound
	return &Error{
		Message:        fmt.Sprintf("%s '%s' not found", typeName, thingNotFound),
		HTTPStatusCode: http.StatusNotFound,
		Trace:          getErrorTrace(),
		InfoMap:        m,
	}
}

// NewNotFoundErrorE returns a new Error with the StatusNotFound http error code
func NewNotFoundErrorE(typeName string, thingNotFound string, err error, m map[string]string) *Error {
	m = ensureMapNotNil(m)
	m["error"] = fmt.Sprintf("%v", err)
	return &Error{
		Message:        fmt.Sprintf("%s '%s' not found, got error: %v", typeName, thingNotFound, err),
		HTTPStatusCode: http.StatusNotFound,
		Trace:          getErrorTrace(),
		InfoMap:        m,
	}
}

// NewRemoteNotFoundError returns a new Error with the StatusNotFound http error code
func NewRemoteNotFoundError(url string, m map[string]string) *Error {
	m = ensureMapNotNil(m)
	m["url"] = url
	return &Error{
		Message:        fmt.Sprintf("Remote call to %s returned Not Found", url),
		HTTPStatusCode: http.StatusNotFound,
		Trace:          getErrorTrace(),
		InfoMap:        m,
	}
}

// NewBadGatewayError returns a new Error with the StatusBadGateway http error code
func NewBadGatewayError(url string, m map[string]string) *Error {
	m = ensureMapNotNil(m)
	m["url"] = url
	return &Error{
		Message:        fmt.Sprintf("Failed call to %s", url),
		HTTPStatusCode: http.StatusBadGateway,
		Trace:          getErrorTrace(),
		InfoMap:        m,
	}
}

// NewBadRequestError returns a new Error with the StatusBadRequest http error code
func NewBadRequestError(msg string, err error, m map[string]string) *Error {
	m = ensureMapNotNil(m)
	m["error"] = fmt.Sprintf("%v", err)
	return &Error{
		Message:        msg,
		HTTPStatusCode: http.StatusBadRequest,
		Trace:          getErrorTrace(),
		InfoMap:        m,
	}
}

func ensureMapNotNil(m map[string]string) map[string]string {
	if m == nil {
		m = map[string]string{}
	}
	return m
}

func getErrorTrace() string {
	_, filename, line, ok := runtime.Caller(2)
	if ok == false {
		return ""
	}
	filename = filepath.Base(filename)
	return fmt.Sprintf("%v:%v", filename, line)
}
