package logicerr

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// Error describes a HTTP error message
type Error struct {
	Message        string
	HttpStatusCode int
	Trace          string
	InfoMap        map[string]string
}

func (e *Error) Error() string {
	return e.Message
}

func NewError(msg string, httpStatusCode int) *Error {
	return &Error{
		Message:        msg,
		HttpStatusCode: httpStatusCode,
		Trace:          getErrorTrace(),
	}
}

func NewErrorE(msg string, httpStatusCode int, err error) *Error {
	return &Error{
		Message:        msg,
		HttpStatusCode: httpStatusCode,
		Trace:          getErrorTrace(),
	}
}

func NewErrorMap(msg string, httpStatusCode int, m map[string]string) *Error {
	m = ensureMapNotNil(m)
	return &Error{
		Message:        msg,
		HttpStatusCode: httpStatusCode,
		Trace:          getErrorTrace(),
		InfoMap:        m,
	}
}

func NewErrorEMap(msg string, httpStatusCode int, err error, m map[string]string) *Error {
	m = ensureMapNotNil(m)
	m["error"] = fmt.Sprintf("%v", err)
	return &Error{
		Message:        msg,
		HttpStatusCode: httpStatusCode,
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
