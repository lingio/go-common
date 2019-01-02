package logicerr

import (
	"fmt"
	"net/http"
)

type Error struct {
	Message        string
	HttpStatusCode int
}

func (e *Error) Error() string {
	return e.Message
}

func NewInternalError(msg string) *Error {
	return &Error{Message: msg, HttpStatusCode: http.StatusInternalServerError}
}

func NewInternalError2(contextMsg string, err error) *Error {
	return &Error{Message: fmt.Sprintf("%s: %s", contextMsg, err.Error()), HttpStatusCode: http.StatusInternalServerError}
}

func NewNotFoundError(typeName string, key string) *Error {
	return &Error{Message: fmt.Sprintf("%s %s not found", typeName, key), HttpStatusCode: http.StatusNotFound}
}

func NewRemoteNotFoundError(url string) *Error {
	return &Error{Message: fmt.Sprintf("Remote call to %s returned Not Found", url), HttpStatusCode: http.StatusNotFound}
}

func NewBadGatewayError(url string) *Error {
	return &Error{Message: fmt.Sprintf("Failed call to %s", url), HttpStatusCode: http.StatusBadGateway}
}

func NewBadRequestError(msg string) *Error {
	return &Error{Message: msg, HttpStatusCode: http.StatusBadRequest}
}
