package engine

import (
	"context"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/tr069-core-only/factory"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/ddddddddwp/tr069-core-only/pkg/core/defaults"
)

func TestEngine_CPEOnboardingAndInflightCorrelation(t *testing.T) {
	parser := factory.NewParser()
	builder := factory.NewBuilder()

	store := defaults.NewMemorySessionStore(0)
	devRepo := defaults.NewMemoryDeviceRepo()
	cmdRepo := defaults.NewMemoryCommandRepo()
	inflight := adapter.NewMemoryInflightRepo(10 * time.Minute)
	hook := defaults.NewInflightCorrelationHook(inflight, cmdRepo)

	reg := defaults.NewExecutorRegistry()
	reg.Register("Reboot", &defaults.RebootExecutor{})

	queue := defaults.NewMemoryQueue()
	queue.Enqueue(&core.Command{
		ID:        "cmd-1",
		DeviceKey: "001122-1234567890",
		Operation: "Reboot",
		Params:    map[string]interface{}{"commandKey": "k"},
	})

	e, err := New(Deps{
		Parser:       parser,
		Builder:      builder,
		SessionStore: store,
		DeviceRepo:   devRepo,
		CommandRepo:  cmdRepo,
		CommandQueue: queue,
		ExecutorReg:  reg,
		InflightRepo: inflight,
		Hook:         hook,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	ctx := context.Background()
	ip := "203.0.113.10"

	_, err = e.Handle(ctx, &core.Request{
		ID:       "req-1",
		RemoteIP: ip,
		Headers:  map[string]string{},
		Body:     []byte(sampleInformXMLBoot2),
	})
	if err != nil {
		t.Fatalf("Handle Inform error: %v", err)
	}

	pollResp, err := e.Handle(ctx, &core.Request{
		ID:       "req-2",
		RemoteIP: ip,
		Headers:  map[string]string{},
		Body:     nil,
	})
	if err != nil {
		t.Fatalf("Handle poll error: %v", err)
	}
	msg, err := parser.ParseMessage(ctx, pollResp.Body)
	if err != nil {
		t.Fatalf("Parse poll response error: %v", err)
	}
	if msg.Method != tr069.MethodGetParameterValues {
		t.Fatalf("expected GetParameterValues first, got %s", msg.Method)
	}
	if msg.ID == "" {
		t.Fatalf("expected cwmp id")
	}

	gpvRespXML, err := builder.BuildMessage(ctx, &tr069.Message{
		Method: tr069.MethodGetParameterValuesResponse,
		ID:     msg.ID,
		Parameters: []tr069.Parameter{
			{Name: "Device.DeviceInfo.SerialNumber", Value: "1234567890", Type: "xsd:string"},
		},
	})
	if err != nil {
		t.Fatalf("build GPV response error: %v", err)
	}
	gpvHandleResp, err := e.Handle(ctx, &core.Request{
		ID:       "req-2-2",
		RemoteIP: ip,
		Headers:  map[string]string{},
		Body:     gpvRespXML,
	})
	if err != nil {
		t.Fatalf("Handle GPV response error: %v", err)
	}
	if gpvHandleResp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", gpvHandleResp.StatusCode)
	}
	msg2, err := parser.ParseMessage(ctx, gpvHandleResp.Body)
	if err != nil {
		t.Fatalf("Parse reboot request error: %v", err)
	}
	if msg2.Method != tr069.MethodReboot {
		t.Fatalf("expected Reboot, got %s", msg2.Method)
	}
	if msg2.ID == "" {
		t.Fatalf("expected cwmp id for reboot")
	}

	if _, ok, _ := inflight.GetByCwmpID(ctx, "001122-1234567890", msg2.ID); !ok {
		t.Fatalf("expected inflight record")
	}

	wrongRespXML, err := builder.BuildMessage(ctx, &tr069.Message{
		Method: tr069.MethodDownloadResponse,
		ID:     msg2.ID,
		Status: 0,
	})
	if err != nil {
		t.Fatalf("build wrong response error: %v", err)
	}
	_, err = e.Handle(ctx, &core.Request{
		ID:       "req-3",
		RemoteIP: ip,
		Headers:  map[string]string{},
		Body:     wrongRespXML,
	})
	if err != nil {
		t.Fatalf("Handle wrong response error: %v", err)
	}
	if _, ok, _ := inflight.GetByCwmpID(ctx, "001122-1234567890", msg2.ID); !ok {
		t.Fatalf("expected inflight still exists after wrong method")
	}

	okRespXML, err := builder.BuildMessage(ctx, &tr069.Message{
		Method: tr069.MethodRebootResponse,
		ID:     msg2.ID,
	})
	if err != nil {
		t.Fatalf("build reboot response error: %v", err)
	}
	_, err = e.Handle(ctx, &core.Request{
		ID:       "req-4",
		RemoteIP: ip,
		Headers:  map[string]string{},
		Body:     okRespXML,
	})
	if err != nil {
		t.Fatalf("Handle reboot response error: %v", err)
	}
	if _, ok, _ := inflight.GetByCwmpID(ctx, "001122-1234567890", msg2.ID); ok {
		t.Fatalf("expected inflight deleted after correct response")
	}
}

const sampleInformXMLBoot2 = `<?xml version="1.0" encoding="UTF-8"?>
<soap_env:Envelope xmlns:soap_env="http://schemas.xmlsoap.org/soap/envelope/" xmlns:soap_enc="http://schemas.xmlsoap.org/soap/encoding/" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap_env:Header>
    <cwmp:ID soap_env:mustUnderstand="1">abc</cwmp:ID>
  </soap_env:Header>
  <soap_env:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>Manufacturer</Manufacturer>
        <OUI>001122</OUI>
        <ProductClass>ProductClass</ProductClass>
        <SerialNumber>1234567890</SerialNumber>
      </DeviceId>
      <Event soap_enc:arrayType="cwmp:EventStruct[1]">
        <EventStruct>
          <EventCode>1 BOOT</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>2023-10-27T10:00:00Z</CurrentTime>
      <RetryCount>0</RetryCount>
      <ParameterList soap_enc:arrayType="cwmp:ParameterValueStruct[1]">
        <ParameterValueStruct>
          <Name>Device.RootDataModelVersion</Name>
          <Value xsi:type="xsd:string">2.0</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:Inform>
  </soap_env:Body>
</soap_env:Envelope>`
