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
		Level:      observability.LevelInfo,
		Stage:      "wire.xml",
		Direction:  observability.DirectionOutbound,
		Attributes: observability.Attributes{CommandID: "cmd-wire", Method: "SetParameterValues", CWMPID: "cwmp-7", TraceID: "trace-7"},
		Payload:    payload,
	})

	if string(payload) != string(original) {
		t.Fatalf("event payload mutated: got %q want %q", payload, original)
	}
	var record model.CommandXML
	if err := db.First(&record, "command_id = ?", "cmd-wire").Error; err != nil {
		t.Fatalf("load command XML: %v", err)
	}
	if record.Direction != "outbound" || record.Method != "SetParameterValues" || record.CWMPID != "cwmp-7" {
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
		CWMPID:     "cwmp-correlated-wire",
		CreatedAt:  time.Now(),
	}
	if err := db.Create(&command).Error; err != nil {
		t.Fatal(err)
	}

	sink := NewCommandXMLSink(service.NewCommandStore(db), func() time.Duration { return time.Hour })
	for _, event := range []observability.Event{
		{
			Level: observability.LevelInfo, Stage: "wire.xml", Direction: observability.DirectionOutbound,
			Attributes: observability.Attributes{CWMPID: command.CWMPID, Method: "Reboot"},
			Payload:    []byte(`<Envelope><Reboot><CommandKey>rpc-key</CommandKey></Reboot></Envelope>`),
		},
		{
			Level: observability.LevelInfo, Stage: "wire.xml", Direction: observability.DirectionInbound,
			Attributes: observability.Attributes{CWMPID: command.CWMPID, Method: "RebootResponse"},
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

func TestCommandXMLSinkCorrelatesAsyncInboundXMLWithoutReusingRequestCWMPID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(new(model.Command), new(model.CommandXML)); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	rebootKey := "reboot-command-key"
	transferKey := "transfer-command-key"
	commands := []model.Command{
		{
			CommandID: "cmd-reboot-async", DeviceID: 18, DeviceKey: "001122-REBOOT",
			Operation: "Reboot", ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusWaitingReboot,
			CommandKey: &rebootKey, CWMPID: "acs-reboot-cwmp", QueuedAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Minute),
		},
		{
			CommandID: "cmd-transfer-async", DeviceID: 19, DeviceKey: "001122-TRANSFER",
			Operation: "Download", ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusWaitingTransfer,
			CommandKey: &transferKey, CWMPID: "acs-transfer-cwmp", QueuedAt: now, CreatedAt: now,
		},
	}
	if err := db.Create(&commands).Error; err != nil {
		t.Fatal(err)
	}

	sink := NewCommandXMLSink(service.NewCommandStore(db), func() time.Duration { return time.Hour })
	sink.Emit(context.Background(), observability.Event{
		Level: observability.LevelInfo, Stage: "wire.xml", Direction: observability.DirectionInbound,
		Attributes: observability.Attributes{CWMPID: "device-reboot-cwmp", Method: "Inform", DeviceKey: commands[0].DeviceKey},
		EventCodes: []string{"1 BOOT"},
		Payload:    []byte(`<Envelope><ID>device-reboot-cwmp</ID><Inform><EventCode>1 BOOT</EventCode></Inform></Envelope>`),
	})
	sink.Emit(context.Background(), observability.Event{
		Level: observability.LevelInfo, Stage: "wire.xml", Direction: observability.DirectionInbound,
		Attributes: observability.Attributes{CWMPID: "device-transfer-cwmp", Method: "TransferComplete", DeviceKey: commands[1].DeviceKey},
		CommandKey: transferKey,
		Payload:    []byte(`<Envelope><ID>device-transfer-cwmp</ID><TransferComplete><CommandKey>transfer-command-key</CommandKey></TransferComplete></Envelope>`),
	})

	var records []model.CommandXML
	if err := db.Order("id ASC").Find(&records).Error; err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("stored rows = %d, want Reboot Inform and TransferComplete XML", len(records))
	}
	if records[0].CommandID != commands[0].CommandID || records[0].CWMPID != "device-reboot-cwmp" {
		t.Fatalf("Reboot Inform record = %#v", records[0])
	}
	if records[1].CommandID != commands[1].CommandID || records[1].CWMPID != "device-transfer-cwmp" {
		t.Fatalf("TransferComplete record = %#v", records[1])
	}
}

func TestCommandXMLSinkDoesNotAttachUnrelatedInformToWaitingReboot(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(new(model.Command), new(model.CommandXML)); err != nil {
		t.Fatal(err)
	}
	command := model.Command{
		CommandID: "cmd-waiting-reboot", DeviceID: 20, DeviceKey: "001122-PERIODIC",
		Operation: "Reboot", ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusWaitingReboot,
		CWMPID: "acs-reboot-cwmp", QueuedAt: time.Now(), CreatedAt: time.Now(),
	}
	if err := db.Create(&command).Error; err != nil {
		t.Fatal(err)
	}
	sink := NewCommandXMLSink(service.NewCommandStore(db), func() time.Duration { return time.Hour })
	sink.Emit(context.Background(), observability.Event{
		Level: observability.LevelInfo, Stage: "wire.xml", Direction: observability.DirectionInbound,
		Attributes: observability.Attributes{CWMPID: "periodic-cwmp", Method: "Inform", DeviceKey: command.DeviceKey},
		EventCodes: []string{"2 PERIODIC"}, Payload: []byte(`<Envelope><Inform/></Envelope>`),
	})
	var count int64
	if err := db.Model(new(model.CommandXML)).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("stored rows = %d, want unrelated Inform ignored", count)
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
