// Package errors defines the error types for the TR069 library.
package errors

import (
 "testing"
)

func TestTR069Error_Error(t *testing.T) {
 err := &TR069Error{
  Code:    9001,
  Message: "Invalid parameter",
 }
 
 expected := "TR069 Error 9001: Invalid parameter"
 if err.Error() != expected {
  t.Errorf("Expected error message %s, got %s", expected, err.Error())
 }
}

func TestNew(t *testing.T) {
 err := New(9002, "Invalid value")
 
 if err.Code != 9002 {
  t.Errorf("Expected code 9002, got %d", err.Code)
 }
 
 if err.Message != "Invalid value" {
  t.Errorf("Expected message 'Invalid value', got %s", err.Message)
 }
}

func TestCommonErrors(t *testing.T) {
 testCases := []struct {
  name     string
  err      *TR069Error
  code     int
  contains string
 }{
  {"ErrInvalidMessage", ErrInvalidMessage, ErrCodeInvalidMessage, "Invalid message format"},
  {"ErrInvalidParameter", ErrInvalidParameter, ErrCodeInvalidParameter, "Invalid parameter"},
  {"ErrInvalidRPCMethod", ErrInvalidRPCMethod, ErrCodeInvalidRPCMethod, "Invalid RPC method"},
  {"ErrInvalidParameterValue", ErrInvalidParameterValue, ErrCodeInvalidParameterValue, "Invalid parameter value"},
  {"ErrInvalidParameterType", ErrInvalidParameterType, ErrCodeInvalidParameterType, "Invalid parameter type"},
  {"ErrInvalidArguments", ErrInvalidArguments, ErrCodeInvalidArguments, "Invalid arguments"},
  {"ErrInvalidSessionID", ErrInvalidSessionID, ErrCodeInvalidSessionID, "Invalid session ID"},
  {"ErrInvalidXMLFormat", ErrInvalidXMLFormat, ErrCodeInvalidXMLFormat, "Invalid XML format"},
  {"ErrInvalidSOAPEnvelope", ErrInvalidSOAPEnvelope, ErrCodeInvalidSOAPEnvelope, "Invalid SOAP envelope"},
  {"ErrRequestDenied", ErrRequestDenied, ErrCodeRequestDenied, "Request denied"},
 }
 
 for _, tc := range testCases {
  t.Run(tc.name, func(t *testing.T) {
   if tc.err.Code != tc.code {
    t.Errorf("Expected code %d, got %d", tc.code, tc.err.Code)
   }
   
   if tc.err.Message != tc.contains {
    t.Errorf("Expected message containing '%s', got '%s'", tc.contains, tc.err.Message)
   }
  })
 }
}