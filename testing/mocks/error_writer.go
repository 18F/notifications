package mocks

import (
	"net/http"
	"github.com/ryanmoran/stack"
)

type ErrorWriter struct {
	WriteCall struct {
		Receives struct {
			Writer http.ResponseWriter
			Error  error
			Context stack.Context
		}
	}
}

func NewErrorWriter() *ErrorWriter {
	return &ErrorWriter{}
}

func (ew *ErrorWriter) Write(writer http.ResponseWriter, err error, context stack.Context) {
	ew.WriteCall.Receives.Writer = writer
	ew.WriteCall.Receives.Error = err
	ew.WriteCall.Receives.Context = context
}
