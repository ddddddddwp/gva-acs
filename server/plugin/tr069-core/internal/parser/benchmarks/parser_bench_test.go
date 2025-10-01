package benchmarks

import (
"context"
"testing"

"github.com/root/demo/tr069/interfaces"
"github.com/root/demo/tr069/internal/parser"
)

var benchmarkResult *interfaces.Message

func BenchmarkParseInformMessage(b *testing.B) {
p := parser.New()

// Sample TR-069 Inform message
xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<soap-env:Envelope
xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"
xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
xmlns:xsd="http://www.w3.org/2001/XMLSchema"
xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
<soap-env:Header>
<cwmp:ID soap-env:mustUnderstand="1">1234567890</cwmp:ID>
</soap-env:Header>
<soap-env:Body>
<cwmp:Inform>
<DeviceId>
<Manufacturer>ExampleCorp</Manufacturer>
<OUI>001122</OUI>
<ProductClass>ExampleModel</ProductClass>
<SerialNumber>1234567890</SerialNumber>
</DeviceId>
<Event soap-enc:arrayType="cwmp:EventStruct[1]">
<EventStruct>
<EventCode>0 BOOTSTRAP</EventCode>
<CommandKey></CommandKey>
</EventStruct>
</Event>
<MaxEnvelopes>1</MaxEnvelopes>
<CurrentTime>2023-01-01T00:00:00Z</CurrentTime>
<RetryCount>0</RetryCount>
<ParameterList soap-enc:arrayType="cwmp:ParameterValueStruct[2]">
<ParameterValueStruct>
<Name>Device.DeviceInfo.Manufacturer</Name>
<Value xsi:type="xsd:string">ExampleCorp</Value>
</ParameterValueStruct>
<ParameterValueStruct>
<Name>Device.DeviceInfo.ModelName</Name>
<Value xsi:type="xsd:string">Model-123</Value>
</ParameterValueStruct>
</ParameterList>
</cwmp:Inform>
</soap-env:Body>
</soap-env:Envelope>`)

b.ResetTimer()
b.ReportAllocs()

var result *interfaces.Message
for i := 0; i < b.N; i++ {
result, _ = p.ParseMessage(context.Background(), xmlData)
}
benchmarkResult = result
}
