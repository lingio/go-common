package common

import (
	"errors"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
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
	} else if errors.Is(err, ErrObjectNotFound) {
		return NewErrorE(http.StatusNotFound, err).Msg("object not found")
	} else if err != nil {
		return NewErrorE(http.StatusInternalServerError, err).Msg("unknown error")
	}
	return NewError(http.StatusInternalServerError).Msg("unknown error")
}

// Error describes a traced HTTP error message with contextual details.
type Error struct {
	Message        string
	HttpStatusCode int
	Trace          string
	Map            map[string]string
	err            error
}

// Note (Axel): Ideally we make NewError/E internal in favor of Errorf:
//   - most calls use http.StatusCodeInternalServerError which doesn't add any valuable information
//   - extract error details from context object instead of explicitely passing them

func NewError(httpStatusCode int) *Error {
	return &Error{
		HttpStatusCode: httpStatusCode,
		Trace:          getErrorTrace(1),
		Map:            make(map[string]string, 0),
		err:            nil,
	}
}

func NewErrorE(httpStatusCode int, err error) *Error {
	return &Error{
		HttpStatusCode: httpStatusCode,
		Trace:          getErrorTrace(1),
		Map:            make(map[string]string, 0),
		err:            err,
	}
}

func FullErrorTrace(e error) string {
	var str strings.Builder
	var indent strings.Builder

	dumpMap := func(m map[string]string, to *strings.Builder) {
		to.WriteString("\n")
		to.WriteString(indent.String())
		to.WriteString("  ")
		to.WriteString("| [map]")
		for k, v := range m {
			to.WriteRune(' ')
			to.WriteString(k)
			to.WriteString(":")
			to.WriteString(v)
		}
	}

	dumpError := func(err error, to *strings.Builder) {
		if lerr, ok := err.(*Error); ok {
			to.WriteString(lerr.Trace)
			to.WriteString(" (")
			to.WriteString(strconv.Itoa(lerr.HttpStatusCode))
			to.WriteString("): \"")
			to.WriteString(lerr.Message)
			to.WriteString("\"")
			if len(lerr.Map) > 0 {
				dumpMap(lerr.Map, to)
			}
		} else {
			to.WriteString("error: ")
			to.WriteString(err.Error())
		}
	}

	if e == nil {
		return ""
	}

	str.WriteString("> ")
	dumpError(e, &str)
	indent.WriteString("  ") // always indent child errors at least one step

	var lasterr error
	err := e
	for {
		lasterr = err

		var lingioErr *Error
		if errors.As(err, &lingioErr) {
			err = lingioErr.err
		} else {
			err = errors.Unwrap(err)
		}

		if err == nil {
			break
		}

		if !errors.Is(err, lasterr) && lasterr != e {
			indent.WriteString("  ")
		}

		str.WriteString("\n")
		str.WriteString(indent.String())
		str.WriteString("\\ ")

		dumpError(err, &str)
	}
	return str.String()
}

// Error constructs a string: message (code) [trace] { k:v }: parent_error
func (e *Error) Error() string {
	var str strings.Builder
	str.WriteString(e.Message)
	str.WriteString(" (")
	str.WriteString(strconv.Itoa(e.HttpStatusCode))
	str.WriteString(") [")
	str.WriteString(e.Trace)
	str.WriteString("]")
	if len(e.Map) > 0 {
		str.WriteString(" {")
		for k, v := range e.Map {
			str.WriteRune(' ')
			str.WriteString(k)
			str.WriteRune(':')
			str.WriteString(v)
		}
		str.WriteString(" }")
	}
	if e.err != nil {
		str.WriteString(": ")
		str.WriteString(e.err.Error())
	}
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

func (e *Error) Unwrap() []error {
	return []error{
		// multierror-style so echo is happy
		&echo.HTTPError{
			Code:     e.HttpStatusCode,
			Message:  e.Message,
			Internal: nil,
		},
		e.err,
	}
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
