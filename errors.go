package go_common

import "fmt"

type Error struct {
	Message        string
	HttpStatusCode int
}

func (e *Error) Error() string {
	return e.Message
}

func NewInternalError(msg string) *Error {
	return &Error{ Message: msg, HttpStatusCode: 500}
}

func NewNotFoundError(typeName string, key string) *Error {
	return &Error{ Message: fmt.Sprintf("%s %s not found", typeName, key), HttpStatusCode: 404}
}

func NewBadGatewayError(url string) *Error {
	return &Error{ Message: fmt.Sprintf("Failed call to %s", url), HttpStatusCode: 502}
}
