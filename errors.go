package common

import (
	"errors"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Errorf returns a common lingio error wrapping the provided error.
// If err is a lingio error, its status code and message will be used
// to initialize the returned error.
//
// The caller may pass a string to be used as error message and an int
// to be used as error status code. If multiple strings and/or ints are
// passed, the last ones will be set in the returned error.
// This function will consider the caller as error origin and the caller
// can change this using the Caller() method.
func Errorf(err error, args ...interface{}) (lerr *Error) {
	defer func() {
		lerr.Caller(1 + 1) // the defer func is considered one stack frame
		for _, arg := range args {
			switch x := arg.(type) {
			case string:
				lerr.Message = x
			case int:
				lerr.HttpStatusCode = x
			}
		}
	}()

	if lerr, ok := err.(*Error); ok && lerr != nil {
		return NewErrorE(lerr.HttpStatusCode, err).Msg(lerr.Message)
	} else if err != nil {
		return NewErrorE(599, err).Msg("unknown error")
	}
	return NewError(599).Msg("unknown error")
}

// Error describes a traced HTTP error message with contextual details.
type Error struct {
	Message        string
	HttpStatusCode int
	Trace          string
	Map            map[string]string
	err            error
}

func NewError(httpStatusCode int) *Error {
	return NewErrorE(httpStatusCode, nil)
}

func NewErrorE(httpStatusCode int, err error) *Error {
	return &Error{
		HttpStatusCode: httpStatusCode,
		Trace:          getErrorTrace(1),
		Map:            make(map[string]string, 0),
		err:            err,
	}
}

func (e *Error) FullTrace() string {
	var indent strings.Builder
	var str strings.Builder
	var mapstr strings.Builder

	dumpMap := func(m map[string]string, to *strings.Builder) {
		mapstr.Reset()
		to.WriteString("\n")
		to.WriteString(indent.String())
		to.WriteString("  ")
		to.WriteString("| [map] ")
		for k, v := range m {
			if mapstr.Len() > 0 {
				mapstr.WriteRune(' ')
			}
			mapstr.WriteString(k)
			mapstr.WriteString(":")
			mapstr.WriteString(v)
		}
		to.WriteString(mapstr.String())
	}

	dumpSummary := func(err *Error, to *strings.Builder) {
		to.WriteString(err.Trace)
		to.WriteString(" (")
		to.WriteString(strconv.Itoa(err.HttpStatusCode))
		to.WriteString("): \"")
		to.WriteString(err.Message)
		to.WriteString("\"")
	}

	str.WriteString("> ")
	dumpSummary(e, &str)
	if len(e.Map) > 0 {
		dumpMap(e.Map, &str)
	}

	var lasterr error
	err := error(e)
	for {
		lasterr = err
		err = errors.Unwrap(err)
		if err == nil {
			break
		}

		if !errors.Is(err, lasterr) {
			indent.WriteString("  ")
		}

		str.WriteString("\n")
		str.WriteString(indent.String())
		str.WriteString("\\ ")

		if lerr, ok := err.(*Error); ok {
			dumpSummary(lerr, &str)
			if len(lerr.Map) > 0 {
				dumpMap(lerr.Map, &str)
			}
		} else {
			str.WriteString("error: ")
			str.WriteString(err.Error())
		}

	}
	return str.String()
}

func (e *Error) Error() string {
	var str strings.Builder
	for k, v := range e.Map {
		str.WriteString(k)
		str.WriteString(": ")
		str.WriteString(v)
		str.WriteString(". ")
	}
	str.WriteString("statusCode: ")
	str.WriteString(strconv.Itoa(e.HttpStatusCode))
	str.WriteString(". trace: ")
	str.WriteString(e.Trace)
	str.WriteString(". message: ")
	str.WriteString(e.Message)
	str.WriteString(".")
	return str.String()
}

func (e *Error) Str(k string, v string) *Error {
	e.ensureMapNotNil()
	e.Map[k] = v
	return e
}

func (e *Error) Int(k string, v int) *Error {
	e.ensureMapNotNil()
	e.Map[k] = strconv.Itoa(v)
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

func (e *Error) Unwrap() error {
	return e.err
}

func (e *Error) Caller(skip int) *Error {
	e.Trace = getErrorTrace(skip + 1) // we are the +1
	return e
}

// Is implements the errors.Is interface.
func (e *Error) Is(target error) bool {
	if e.err == target {
		return true
	} else if err, ok := target.(*Error); ok {
		return err.HttpStatusCode == e.HttpStatusCode && err.Message == e.Message
	} else {
		return false
	}
}

func getErrorTrace(skip int) string {
	_, filename, line, ok := runtime.Caller(skip + 1)
	if ok == false {
		return "caller not identified"
	}
	filename = filepath.Base(filename)
	return filename + ":" + strconv.Itoa(line)
}
