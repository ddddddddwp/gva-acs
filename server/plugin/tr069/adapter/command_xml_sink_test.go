package adapter

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/ddddddddwp/tr069-core-only/observability"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCommandXMLSinkPersistsSanitizedWireEventWithoutMutatingPayload(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(new(model.CommandXML)); err != nil {
		t.Fatal(err)
	}
	retention := 48 * time.Hour
	sink := NewCommandXMLSink(service.NewCommandStore(db), func() time.Duration { return retention })
	payload := []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>wire-secret</Value></ParameterValueStruct><Password>download-secret</Password></Envelope>`)
	original := append([]byte(nil), payload...)

	if !sink.Enabled(observability.LevelInfo) || sink.Enabled(observability.LevelDebug) || sink.Enabled(observability.LevelError) {
		t.Fatal("sink must enable only INFO events")
	}
	sink.Emit(context.Background(), observability.Event{
		Level:     observability.LevelInfo,
		Stage:     "wire.xml",
		Direction: observability.DirectionOutbound,
		Attributes: observability.Attributes{
			CommandID: "cmd-wire", Method: "SetParameterValues", CWMPID: "cwmp-7", RequestID: "request-7",
		},
		Payload: payload,
	})

	if string(payload) != string(original) {
		t.Fatalf("event payload mutated: got %q want %q", payload, original)
	}
	var record model.CommandXML
	if err := db.First(&record, "command_id = ?", "cmd-wire").Error; err != nil {
		t.Fatalf("load command XML: %v", err)
	}
	if record.Direction != "outbound" || record.Method != "SetParameterValues" || record.CWMPID != "cwmp-7" || record.RequestID != "request-7" {
		t.Fatalf("metadata = %#v", record)
	}
	if strings.Contains(string(record.Payload), "wire-secret") || !strings.Contains(string(record.Payload), "******") {
		t.Fatalf("stored wire XML was not sanitized: %s", record.Payload)
	}
	if !strings.Contains(string(record.Payload), "download-secret") {
		t.Fatalf("unrelated password changed: %s", record.Payload)
	}
	if got := record.ExpiresAt.Sub(record.CreatedAt); got != retention {
		t.Fatalf("retention = %s, want %s", got, retention)
	}
}

func TestCommandXMLSinkCorrelatesBidirectionalWireEventsByCWMPID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(new(model.Command), new(model.CommandXML)); err != nil {
		t.Fatal(err)
	}
	command := model.Command{
		CommandID:  "cmd-correlated-wire",
		DeviceID:   8,
		DeviceKey:  "001122-WIRE",
		Operation:  "Reboot",
		ParamsJSON: model.LongTextJSON(`{}`),
		Status:     model.CommandStatusSent,
		RequestID:  "cwmp-correlated-wire",
		CreatedAt:  time.Now(),
	}
	if err := db.Create(&command).Error; err != nil {
		t.Fatal(err)
	}

	sink := NewCommandXMLSink(service.NewCommandStore(db), func() time.Duration { return time.Hour })
	for _, event := range []observability.Event{
		{
			Level: observability.LevelInfo, Stage: "wire.xml", Direction: observability.DirectionOutbound,
			Attributes: observability.Attributes{CWMPID: command.RequestID, Method: "Reboot"},
			Payload:    []byte(`<Envelope><Reboot><CommandKey>rpc-key</CommandKey></Reboot></Envelope>`),
		},
		{
			Level: observability.LevelInfo, Stage: "wire.xml", Direction: observability.DirectionInbound,
			Attributes: observability.Attributes{CWMPID: command.RequestID, Method: "RebootResponse"},
			Payload:    []byte(`<Envelope><RebootResponse/></Envelope>`),
		},
	} {
		sink.Emit(context.Background(), event)
	}

	var records []model.CommandXML
	if err := db.Order("id ASC").Find(&records).Error; err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("stored rows = %d, want outbound and inbound XML", len(records))
	}
	if records[0].CommandID != command.CommandID || records[0].Direction != string(observability.DirectionOutbound) {
		t.Fatalf("outbound record = %#v", records[0])
	}
	if records[1].CommandID != command.CommandID || records[1].Direction != string(observability.DirectionInbound) {
		t.Fatalf("inbound record = %#v", records[1])
	}
}

func TestCommandXMLSinkFiltersIncompleteOrNonWireEvents(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(new(model.Command), new(model.CommandXML)); err != nil {
		t.Fatal(err)
	}
	sink := NewCommandXMLSink(service.NewCommandStore(db), func() time.Duration { return time.Hour })
	for _, event := range []observability.Event{
		{Level: observability.LevelInfo, Stage: "parser.completed", Attributes: observability.Attributes{CommandID: "cmd"}, Payload: []byte(`<Envelope/>`)},
		{Level: observability.LevelInfo, Stage: "wire.xml", Payload: []byte(`<Envelope/>`)},
		{Level: observability.LevelInfo, Stage: "wire.xml", Attributes: observability.Attributes{CWMPID: "cwmp-missing"}, Payload: []byte(`<Envelope/>`)},
		{Level: observability.LevelInfo, Stage: "wire.xml", Attributes: observability.Attributes{CommandID: "cmd"}},
	} {
		sink.Emit(context.Background(), event)
	}
	var count int64
	if err := db.Model(new(model.CommandXML)).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("stored rows = %d, want 0", count)
	}
}

func TestCommandXMLSinkMalformedPayloadFailsClosed(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(new(model.CommandXML)); err != nil {
		t.Fatal(err)
	}
	sink := NewCommandXMLSink(service.NewCommandStore(db), func() time.Duration { return time.Hour })
	sink.Emit(context.Background(), observability.Event{
		Level: observability.LevelInfo, Stage: "wire.xml",
		Attributes: observability.Attributes{CommandID: "cmd-malformed"},
		Payload:    []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>wire-secret</Envelope>`),
	})
	var count int64
	if err := db.Model(new(model.CommandXML)).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("stored rows = %d, want 0", count)
	}
}

func TestCommandXMLSinkErrorReasonIsSafeAndActionable(t *testing.T) {
	if got := commandXMLSinkErrorReason(service.ErrInvalidCommandXML); got != "invalid_xml" {
		t.Fatalf("invalid XML reason = %q", got)
	}
	if got := commandXMLSinkErrorReason(context.DeadlineExceeded); got != "storage" {
		t.Fatalf("storage reason = %q", got)
	}
}
