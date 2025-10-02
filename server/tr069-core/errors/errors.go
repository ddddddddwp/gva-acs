// Package errors defines the error types for the TR069 library.
package errors

import (
 "fmt"
)

// Error codes
const (
 ErrCodeInvalidMessage      = 9000
 ErrCodeInvalidParameter    = 9001
 ErrCodeInvalidRPCMethod    = 9002
 ErrCodeInvalidParameterValue = 9003
 ErrCodeInvalidParameterType  = 9004
 ErrCodeInvalidArguments    = 9005
 ErrCodeInvalidSessionID    = 9006
 ErrCodeInvalidXMLFormat    = 9007
 ErrCodeInvalidSOAPEnvelope = 9008
 ErrCodeRequestDenied       = 9009
)

// TR069Error represents a TR069 protocol error.
type TR069Error struct {
 Code    int
 Message string
}

// Error returns the error message.
func (e *TR069Error) Error() string {
 return fmt.Sprintf("TR069 Error %d: %s", e.Code, e.Message)
}

// New creates a new TR069Error.
func New(code int, message string) *TR069Error {
 return &TR069Error{
  Code:    code,
  Message: message,
 }
}

// Common TR069 errors
var (
 ErrInvalidMessage      = New(ErrCodeInvalidMessage, "Invalid message format")
 ErrInvalidParameter    = New(ErrCodeInvalidParameter, "Invalid parameter")
 ErrInvalidRPCMethod    = New(ErrCodeInvalidRPCMethod, "Invalid RPC method")
 ErrInvalidParameterValue = New(ErrCodeInvalidParameterValue, "Invalid parameter value")
 ErrInvalidParameterType  = New(ErrCodeInvalidParameterType, "Invalid parameter type")
 ErrInvalidArguments    = New(ErrCodeInvalidArguments, "Invalid arguments")
 ErrInvalidSessionID    = New(ErrCodeInvalidSessionID, "Invalid session ID")
 ErrInvalidXMLFormat    = New(ErrCodeInvalidXMLFormat, "Invalid XML format")
 ErrInvalidSOAPEnvelope = New(ErrCodeInvalidSOAPEnvelope, "Invalid SOAP envelope")
 ErrRequestDenied       = New(ErrCodeRequestDenied, "Request denied")
)