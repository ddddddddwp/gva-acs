package adapter

import (
	"context"
	"testing"
	"time"

	appGlobal "github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type captureIngest struct {
	cmds []*core.Command
}

func (i *captureIngest) Enqueue(ctx context.Context, cmd *core.Command) error {
	_ = ctx
	if cmd != nil {
		i.cmds = append(i.cmds, cmd)
	}
	return nil
}

func TestDataModelHook_PersistsGPVAndExpandsGPN(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	appGlobal.GVA_DB = db
	t.Cleanup(func() { appGlobal.GVA_DB = nil })

	if err := db.AutoMigrate(new(model.Device), new(model.Command), new(model.DataModelValue), new(model.DeviceRPCMethods)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	dev := model.Device{OUI: "001122", SerialNumber: "1234567890"}
	if err := db.Create(&dev).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	inflight := NewMemoryInflightRepo(10 * time.Minute)
	ingest := &captureIngest{}
	hook := NewDataModelHook(nil, inflight, ingest, WithDataModelHookNow(func() time.Time { return time.Unix(10, 0) }))

	ctx := context.Background()
	session := &core.Session{
		DeviceKey: "001122-1234567890",
		DeviceID:  "1234567890",
	}

	if err := inflight.Save(ctx, core.InflightRequest{
		DeviceKey:   session.DeviceKey,
		CommandID:   "cmd-gpv",
		CwmpID:      "cwmp-gpv",
		RequestName: tr069.MethodGetParameterValues,
		SentAt:      time.Now(),
	}); err != nil {
		t.Fatalf("save inflight: %v", err)
	}

	handled, err := hook.OnResponse(ctx, session, &tr069.Message{
		Method: tr069.MethodGetParameterValuesResponse,
		ID:     "cwmp-gpv",
		Parameters: []tr069.Parameter{
			{Name: "Device.DeviceInfo.SerialNumber", Value: "1234567890", Type: "xsd:string"},
		},
	})
	if err != nil {
		t.Fatalf("OnResponse error: %v", err)
	}
	if !handled {
		t.Fatalf("expected handled=true for correlated response")
	}

	var got model.DataModelValue
	if err := db.Where("device_id = ? AND name = ?", dev.ID, "Device.DeviceInfo.SerialNumber").First(&got).Error; err != nil {
		t.Fatalf("expected datamodel value persisted: %v", err)
	}

	if err := db.Create(&model.Command{
		CommandID:  "cmd-gpn",
		DeviceKey:  session.DeviceKey,
		Operation:  "GetParameterNames",
		ParamsJSON: `{"parameterPath":"Device.","depth":0,"maxDepth":2}`,
		Status:     "SENDING",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}).Error; err != nil {
		t.Fatalf("create cmd: %v", err)
	}

	if err := inflight.Save(ctx, core.InflightRequest{
		DeviceKey:   session.DeviceKey,
		CommandID:   "cmd-gpn",
		CwmpID:      "cwmp-gpn",
		RequestName: tr069.MethodGetParameterNames,
		SentAt:      time.Now(),
	}); err != nil {
		t.Fatalf("save inflight gpn: %v", err)
	}

	handled, err = hook.OnResponse(ctx, session, &tr069.Message{
		Method: tr069.MethodGetParameterNamesResponse,
		ID:     "cwmp-gpn",
		ParameterInfos: []tr069.ParameterInfo{
			{Name: "Device.Services.", Writable: false},
			{Name: "Device.DeviceInfo.SerialNumber", Writable: false},
		},
	})
	if err != nil {
		t.Fatalf("OnResponse gpn error: %v", err)
	}
	if !handled {
		t.Fatalf("expected handled=true for correlated gpn response")
	}
	if len(ingest.cmds) == 0 {
		t.Fatalf("expected expansion enqueued commands")
	}
	foundGPN := false
	foundGPV := false
	for _, c := range ingest.cmds {
		if c.Operation == "GetParameterNames" {
			foundGPN = true
		}
		if c.Operation == "GetParameterValues" {
			foundGPV = true
		}
	}
	if !foundGPN || !foundGPV {
		t.Fatalf("expected both GetParameterNames and GetParameterValues enqueued, got=%v", ingest.cmds)
	}
}

