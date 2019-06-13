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
func NewInternalError(msg string) *Error {
	return &Error{
		Message:        msg,
		HTTPStatusCode: http.StatusInternalServerError,
		Trace:          getErrorTrace(),
		InfoMap         map[string]string{},
	}
}

// NewInternalErrorE returns a new Error with the InternalServerError http error code
func NewInternalErrorE(contextMsg string, err error) *Error {
	return &Error{
		Message:        fmt.Sprintf("%s: %s", contextMsg, err.Error()),
		HTTPStatusCode: http.StatusInternalServerError,
		Trace:          getErrorTrace(),
		InfoMap:        map[string]string{"error": fmt.Sprintf("%v", err)},
	}
}

// NewInternalError1 returns a new Error with the InternalServerError http error code
func NewInternalError1(msg string, key string, value string) *Error {
	return &Error{
		Message:        msg,
		HTTPStatusCode: http.StatusInternalServerError,
		Trace:          getErrorTrace(),
		InfoMap:        map[string]string{key: value},
	}
}

// NewInternalErrorE1 returns a new Error with the InternalServerError http error code
func NewInternalErrorE1(contextMsg string, err error, key string, value string) *Error {
	return &Error{
		Message:        fmt.Sprintf("%s: %s", contextMsg, err.Error()),
		HTTPStatusCode: http.StatusInternalServerError,
		Trace:          getErrorTrace(),
		InfoMap:        map[string]string{key: value, "error": fmt.Sprintf("%v", err)},
	}
}

// NewNotFoundError returns a new Error with the StatusNotFound http error code
func NewNotFoundError(typeName string, thingNotFound string) *Error {
	return &Error{
		Message:        fmt.Sprintf("%s %s not found", typeName, thingNotFound),
		HTTPStatusCode: http.StatusNotFound,
		Trace:          getErrorTrace(),
		InfoMap:        map[string]string{"notFound": thingNotFound},
	}
}

// NewNotFoundError1 returns a new Error with the StatusNotFound http error code
func NewNotFoundError1(typeName string, thingNotFound string, key string, value string) *Error {
	return &Error{
		Message:        fmt.Sprintf("%s %s not found", typeName, thingNotFound),
		HTTPStatusCode: http.StatusNotFound,
		Trace:          getErrorTrace(),
		InfoMap:        map[string]string{"notFound": thingNotFound, key: value},
	}
}

// NewNotFoundErrorE returns a new Error with the StatusNotFound http error code
func NewNotFoundErrorE(typeName string, thingNotFound string, err error) *Error {
	return &Error{
		Message:        fmt.Sprintf("%s %s not found got error: %v", typeName, thingNotFound, err),
		HTTPStatusCode: http.StatusNotFound,
		Trace:          getErrorTrace(),
		InfoMap:        map[string]string{"error": fmt.Sprintf("%v", err)},
	}
}

// NewRemoteNotFoundError returns a new Error with the StatusNotFound http error code
func NewRemoteNotFoundError(url string) *Error {
	return &Error{
		Message:        fmt.Sprintf("Remote call to %s returned Not Found", url),
		HTTPStatusCode: http.StatusNotFound,
		Trace:          getErrorTrace(),
		InfoMap:        map[string]string{"url": url},
	}
}

// NewBadGatewayError returns a new Error with the StatusBadGateway http error code
func NewBadGatewayError(url string) *Error {
	return &Error{
		Message:        fmt.Sprintf("Failed call to %s", url),
		HTTPStatusCode: http.StatusBadGateway,
		Trace:          getErrorTrace(),
		InfoMap:        map[string]string{"url": url},
	}
}

// NewBadRequestError returns a new Error with the StatusBadRequest http error code
func NewBadRequestError(msg string, err error) *Error {
	return &Error{
		Message:        msg,
		HTTPStatusCode: http.StatusBadRequest,
		Trace:          getErrorTrace(),
		InfoMap:        map[string]string{"error":fmt.Sprintf("%v", err)},
	}
}

func getErrorTrace() string {
	_, filename, line, ok := runtime.Caller(2)
	if ok == false {
		return ""
	}
	filename = filepath.Base(filename)
	return fmt.Sprintf("%v:%v", filename, line)
}
