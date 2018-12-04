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

func NewNotFoundError(typeName string, key string) *Error {
	return &Error{Message: fmt.Sprintf("%s %s not found", typeName, key), HttpStatusCode: http.StatusNotFound}
}

func NewBadGatewayError(url string) *Error {
	return &Error{Message: fmt.Sprintf("Failed call to %s", url), HttpStatusCode: http.StatusBadGateway}
}

func NewBadRequestError(msg string) *Error {
	return &Error{Message: msg, HttpStatusCode: http.StatusBadRequest}
}
