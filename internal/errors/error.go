package errors

import (
	"fmt"
	"net/http"
)

type Error struct {
	Message  string `json:"message"`
	Layer    string `json:"layer,omitempty"`
	Code     int    `json:"-"`
	Internal error  `json:"-"`
}

func (e *Error) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Layer, e.Message, e.Internal)
	}
	return fmt.Sprintf("[%s] %s", e.Layer, e.Message)
}

func New(code int, message, layer string, err error) *Error {
	return &Error{
		Code:     code,
		Message:  message,
		Layer:    layer,
		Internal: err,
	}
}

func NewBadRequest(message, layer string, err error) *Error {
	return New(http.StatusBadRequest, message, layer, err)
}

func NewNotFound(message, layer string, err error) *Error {
	return New(http.StatusNotFound, message, layer, err)
}

func NewInternal(layer string, err error) *Error {
	return New(http.StatusInternalServerError, "Internal Server Error", layer, err)
}
