// Package builder implements the TR069 message builder.
package builder

import (
 "context"
 "testing"
 
 "github.com/root/demo/tr069/interfaces"
)

func TestBuilder_BuildMessage(t *testing.T) {
 builder := New()
 
 msg := &interfaces.Message{
  Method: interfaces.MethodInform,
  Parameters: []interfaces.Parameter{
   {
    Name:  "Device.DeviceInfo.Manufacturer",
    Value: "ExampleCorp",
    Type:  "string",
   },
  },
 }
 
 data, err := builder.BuildMessage(context.Background(), msg)
 if err != nil {
  t.Fatalf("Failed to build message: %v", err)
 }
 
 if len(data) == 0 {
  t.Error("Expected non-empty data")
 }
 
 // We expect an InformResponse for MethodInform
 expected := `<?xml version="1.0" encoding="UTF-8"?>
<Envelope><Header ID=""></Header><Body><InformResponse><MaxEnvelopes>1</MaxEnvelopes></InformResponse></Body></Envelope>`
 if string(data) != expected {
  t.Errorf("Unexpected message content: %s", string(data))
 }
}

func TestBuilder_BuildFault(t *testing.T) {
 builder := New()
 
 data, err := builder.BuildFault(context.Background(), 9001, "Invalid parameter")
 if err != nil {
  t.Fatalf("Failed to build fault: %v", err)
 }
 
 if len(data) == 0 {
  t.Error("Expected non-empty data")
 }
 
 expected := `<?xml version="1.0" encoding="UTF-8"?>
<Envelope><Header ID=""></Header><Body><Fault><FaultCode>9001</FaultCode><FaultString>Invalid parameter</FaultString></Fault></Body></Envelope>`
 if string(data) != expected {
  t.Errorf("Unexpected fault content: %s", string(data))
 }
}

func TestBuilder_SetPrettyPrint(t *testing.T) {
 builder := New()
 
 builder.SetPrettyPrint(true)
 if !builder.GetPrettyPrint() {
  t.Error("Expected pretty print to be true")
 }
 
 builder.SetPrettyPrint(false)
 if builder.GetPrettyPrint() {
  t.Error("Expected pretty print to be false")
 }
}