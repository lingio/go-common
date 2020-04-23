package common

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

// Error describes a HTTP error message
type Error struct {
	Message        string
	HttpStatusCode int
	Trace          string
	Map            map[string]string
}

func (e *Error) Error() string {
	return e.Message
}

func NewError(httpStatusCode int) *Error {
	return &Error{
		HttpStatusCode: httpStatusCode,
		Trace:          getErrorTrace(),
		Map:            make(map[string]string, 0),
	}
}

func NewErrorE(httpStatusCode int, err error) *Error {
	m := make(map[string]string, 0)
	m["error"] = err.Error()
	return &Error{
		HttpStatusCode: httpStatusCode,
		Trace:          getErrorTrace(),
		Map:            m,
	}
}

func (e *Error) Str(k string, v string) *Error {
	e.ensureMapNotNil()
	e.Map[k] = v
	return e
}

func (e *Error) Int(k string, v int) *Error {
	e.ensureMapNotNil()
	e.Map[k] = fmt.Sprintf("%d", v)
	return e
}

func (e *Error) Msg(msg string) *Error {
	e.Message = msg
	return e
}

func (e *Error) Datetime(k string, v time.Time) *Error {
	e.ensureMapNotNil()
	e.Map[k] = v.String()
	return e
}

func (e *Error) ensureMapNotNil() {
	if e.Map == nil {
		e.Map = make(map[string]string, 0)
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
