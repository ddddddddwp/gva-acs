package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func newCommandStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Command), new(model.CommandEvent), new(model.CommandXML)); err != nil {
		t.Fatalf("migrate command models: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	return db
}

func TestCommandStoreCreateCommitsCommandAndCreatedEvent(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	command := &model.Command{
		CommandID:  "cmd-create",
		DeviceID:   7,
		DeviceKey:  "001122-SN7",
		Operation:  "GetRPCMethods",
		ParamsJSON: model.LongTextJSON(`{}`),
		Status:     model.CommandStatusQueued,
		QueuedAt:   now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := store.Create(context.Background(), command); err != nil {
		t.Fatalf("create command: %v", err)
	}

	var gotCommand model.Command
	if err := db.First(&gotCommand, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load command: %v", err)
	}
	if gotCommand.Status != model.CommandStatusQueued {
		t.Fatalf("status = %q, want %q", gotCommand.Status, model.CommandStatusQueued)
	}

	var events []model.CommandEvent
	if err := db.Where("command_id = ?", command.CommandID).Order("id ASC").Find(&events).Error; err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].EventType != model.CommandEventCreated {
		t.Fatalf("event type = %q, want %q", events[0].EventType, model.CommandEventCreated)
	}
	if events[0].FromStatus != "" || events[0].ToStatus != model.CommandStatusQueued {
		t.Fatalf("event transition = %q -> %q, want empty -> %q", events[0].FromStatus, events[0].ToStatus, model.CommandStatusQueued)
	}
}

func TestCommandStoreCreateRollsBackCommandWhenCreatedEventFails(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	injected := errors.New("injected command event failure")
	if err := db.Callback().Create().Before("gorm:create").Register("test:fail_command_event", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Name == "CommandEvent" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}

	command := &model.Command{
		CommandID:  "cmd-rollback",
		DeviceID:   8,
		DeviceKey:  "001122-SN8",
		Operation:  "Reboot",
		ParamsJSON: model.LongTextJSON(`{}`),
		Status:     model.CommandStatusQueued,
		QueuedAt:   time.Now(),
	}
	if err := store.Create(context.Background(), command); !errors.Is(err, injected) {
		t.Fatalf("create error = %v, want %v", err, injected)
	}

	var commandCount int64
	if err := db.Model(new(model.Command)).Where("command_id = ?", command.CommandID).Count(&commandCount).Error; err != nil {
		t.Fatalf("count commands: %v", err)
	}
	var eventCount int64
	if err := db.Model(new(model.CommandEvent)).Where("command_id = ?", command.CommandID).Count(&eventCount).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if commandCount != 0 || eventCount != 0 {
		t.Fatalf("persisted rows after rollback: commands=%d events=%d", commandCount, eventCount)
	}
}

func TestCommandTransitionMapAndTerminalHelpers(t *testing.T) {
	accepted := []string{
		model.CommandStatusQueued,
		model.CommandStatusWaitingDevice,
		model.CommandStatusBuilding,
		model.CommandStatusSent,
		model.CommandStatusWaitingTransfer,
		model.CommandStatusWaitingReboot,
		model.CommandStatusCompleted,
		model.CommandStatusFailed,
		model.CommandStatusTimeout,
	}
	for _, status := range accepted {
		if !model.IsCommandStatus(status) {
			t.Errorf("status %q is not accepted", status)
		}
		if model.IsTerminalCommandStatus(status) == model.IsNonTerminalCommandStatus(status) {
			t.Errorf("status %q must be exactly one of terminal/non-terminal", status)
		}
	}

	allowed := [][2]string{
		{model.CommandStatusQueued, model.CommandStatusWaitingDevice},
		{model.CommandStatusQueued, model.CommandStatusFailed},
		{model.CommandStatusWaitingDevice, model.CommandStatusBuilding},
		{model.CommandStatusWaitingDevice, model.CommandStatusTimeout},
		{model.CommandStatusBuilding, model.CommandStatusWaitingDevice},
		{model.CommandStatusBuilding, model.CommandStatusSent},
		{model.CommandStatusSent, model.CommandStatusWaitingTransfer},
		{model.CommandStatusSent, model.CommandStatusWaitingReboot},
		{model.CommandStatusSent, model.CommandStatusCompleted},
		{model.CommandStatusWaitingTransfer, model.CommandStatusCompleted},
		{model.CommandStatusWaitingReboot, model.CommandStatusCompleted},
		{model.CommandStatusWaitingReboot, model.CommandStatusFailed},
		{model.CommandStatusWaitingReboot, model.CommandStatusTimeout},
	}
	for _, transition := range allowed {
		if !model.CanTransitionCommand(transition[0], transition[1]) {
			t.Errorf("transition %s -> %s should be allowed", transition[0], transition[1])
		}
	}

	for _, from := range accepted {
		if from != model.CommandStatusSent && model.CanTransitionCommand(from, model.CommandStatusWaitingTransfer) {
			t.Errorf("WAITING_TRANSFER must not follow %s", from)
		}
		if from != model.CommandStatusSent && model.CanTransitionCommand(from, model.CommandStatusWaitingReboot) {
			t.Errorf("WAITING_REBOOT must not follow %s", from)
		}
	}
	for _, terminal := range []string{model.CommandStatusCompleted, model.CommandStatusFailed, model.CommandStatusTimeout} {
		for _, to := range accepted {
			if model.CanTransitionCommand(terminal, to) {
				t.Errorf("terminal transition %s -> %s must be rejected", terminal, to)
			}
		}
	}
}

func TestCommandStoreTransitionConditionallyUpdatesVersionAndAppendsEvent(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	now := time.Date(2026, 7, 16, 11, 0, 0, 0, time.UTC)
	command := &model.Command{
		CommandID:  "cmd-transition",
		DeviceID:   9,
		DeviceKey:  "001122-SN9",
		Operation:  "GetRPCMethods",
		ParamsJSON: model.LongTextJSON(`{}`),
		Status:     model.CommandStatusSent,
		QueuedAt:   now.Add(-time.Minute),
		SentAt:     pointerTo(now.Add(-time.Second)),
		CreatedAt:  now.Add(-time.Minute),
		UpdatedAt:  now.Add(-time.Second),
	}
	if err := store.Create(context.Background(), command); err != nil {
		t.Fatalf("seed command: %v", err)
	}

	got, err := store.Transition(context.Background(), CommandTransition{
		CommandID:       command.CommandID,
		FromStatuses:    []string{model.CommandStatusSent},
		ToStatus:        model.CommandStatusCompleted,
		ExpectedVersion: 0,
		EventType:       "RESPONSE_COMPLETED",
		Stage:           "response",
		Message:         "device response received",
		PayloadJSON:     datatypes.JSON(`{"status":0}`),
		Updates: map[string]any{
			"finished_at": now,
			"result_json": datatypes.JSON(`{"status":0}`),
		},
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if got.Status != model.CommandStatusCompleted || got.Version != 1 {
		t.Fatalf("transitioned command status/version = %s/%d", got.Status, got.Version)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(now) {
		t.Fatalf("finishedAt = %v, want %v", got.FinishedAt, now)
	}

	var events []model.CommandEvent
	if err := db.Where("command_id = ?", command.CommandID).Order("id ASC").Find(&events).Error; err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d, want CREATED plus transition", len(events))
	}
	event := events[1]
	if event.EventType != "RESPONSE_COMPLETED" || event.FromStatus != model.CommandStatusSent || event.ToStatus != model.CommandStatusCompleted {
		t.Fatalf("transition event = %#v", event)
	}
	if event.Stage != "response" || event.Message != "device response received" || string(event.PayloadJSON) != `{"status":0}` {
		t.Fatalf("transition event metadata = %#v", event)
	}
}

func TestCommandStoreTransitionConcurrentTerminalCompletionIsIdempotent(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	command := &model.Command{
		CommandID:  "cmd-concurrent-completion",
		DeviceID:   10,
		DeviceKey:  "001122-SN10",
		Operation:  "GetParameterValues",
		ParamsJSON: model.LongTextJSON(`{"paths":["Device."]}`),
		Status:     model.CommandStatusSent,
		QueuedAt:   time.Now().Add(-time.Minute),
		CreatedAt:  time.Now().Add(-time.Minute),
	}
	if err := store.Create(context.Background(), command); err != nil {
		t.Fatalf("seed command: %v", err)
	}

	start := make(chan struct{})
	results := make([]model.Command, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			results[index], errs[index] = store.Transition(context.Background(), CommandTransition{
				CommandID:       command.CommandID,
				FromStatuses:    []string{model.CommandStatusSent},
				ToStatus:        model.CommandStatusCompleted,
				ExpectedVersion: 0,
				EventType:       "RESPONSE_COMPLETED",
			})
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("transition %d returned error: %v", i, err)
		}
		if results[i].Status != model.CommandStatusCompleted || results[i].Version != 1 {
			t.Errorf("transition %d result = %s/%d, want COMPLETED/1", i, results[i].Status, results[i].Version)
		}
	}

	var completionEvents int64
	if err := db.Model(new(model.CommandEvent)).
		Where("command_id = ? AND event_type = ?", command.CommandID, "RESPONSE_COMPLETED").
		Count(&completionEvents).Error; err != nil {
		t.Fatalf("count completion events: %v", err)
	}
	if completionEvents != 1 {
		t.Fatalf("completion events = %d, want 1", completionEvents)
	}
}

func TestCommandStoreCreateRejectsInvalidSerializedJSON(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	command := &model.Command{
		CommandID:  "cmd-invalid-json",
		DeviceID:   11,
		DeviceKey:  "001122-SN11",
		Operation:  "Reboot",
		ParamsJSON: model.LongTextJSON(`{"broken"`),
		Status:     model.CommandStatusQueued,
		QueuedAt:   time.Now(),
	}

	if err := store.Create(context.Background(), command); !errors.Is(err, ErrInvalidCommandJSON) {
		t.Fatalf("create error = %v, want ErrInvalidCommandJSON", err)
	}

	var commandCount int64
	if err := db.Model(new(model.Command)).Where("command_id = ?", command.CommandID).Count(&commandCount).Error; err != nil {
		t.Fatalf("count commands: %v", err)
	}
	if commandCount != 0 {
		t.Fatalf("commands = %d, want 0", commandCount)
	}
}

func TestCommandStoreCreateRejectsUnknownStatus(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	command := &model.Command{
		CommandID: "cmd-invalid-status",
		Status:    "PENDING",
		QueuedAt:  time.Now(),
	}

	if err := store.Create(context.Background(), command); !errors.Is(err, ErrInvalidCommandStatus) {
		t.Fatalf("create error = %v, want ErrInvalidCommandStatus", err)
	}
}

func TestCommandStoreAppendEventAndSaveXMLPreservePayloads(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	now := time.Now().UTC().Truncate(time.Second)
	command := &model.Command{
		CommandID: "cmd-payloads",
		DeviceID:  12,
		DeviceKey: "001122-SN12",
		Operation: "Download",
		Status:    model.CommandStatusSent,
		QueuedAt:  now.Add(-time.Minute),
	}
	if err := store.Create(context.Background(), command); err != nil {
		t.Fatalf("seed command: %v", err)
	}

	eventPayload := model.LongTextJSON(`{"attempt":1,"accepted":true}`)
	if err := store.AppendEvent(context.Background(), &model.CommandEvent{
		CommandID:   command.CommandID,
		EventType:   "CORE_STAGE",
		Stage:       "builder.completed",
		PayloadJSON: eventPayload,
		CreatedAt:   now,
	}); err != nil {
		t.Fatalf("append event: %v", err)
	}

	xmlPayload := []byte("<Download><Password>secret</Password></Download>")
	if err := store.SaveXML(context.Background(), &model.CommandXML{
		CommandID: command.CommandID,
		Direction: "outbound",
		Method:    "Download",
		CWMPID:    "cwmp-12",
		Payload:   xmlPayload,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("save XML: %v", err)
	}

	var gotEvent model.CommandEvent
	if err := db.Where("command_id = ? AND event_type = ?", command.CommandID, "CORE_STAGE").First(&gotEvent).Error; err != nil {
		t.Fatalf("load appended event: %v", err)
	}
	if string(gotEvent.PayloadJSON) != string(eventPayload) {
		t.Fatalf("event payload = %q, want %q", gotEvent.PayloadJSON, eventPayload)
	}
	var gotXML model.CommandXML
	if err := db.Where("command_id = ?", command.CommandID).First(&gotXML).Error; err != nil {
		t.Fatalf("load XML: %v", err)
	}
	if string(gotXML.Payload) != string(xmlPayload) {
		t.Fatalf("XML payload changed: got %q want %q", gotXML.Payload, xmlPayload)
	}
}

func TestCommandStoreSaveXMLSanitizesCopyBeforePersistence(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	payload := []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>database-secret</Value></ParameterValueStruct><Password>download-secret</Password></Envelope>`)
	original := append([]byte(nil), payload...)
	record := &model.CommandXML{CommandID: "cmd-redact", Payload: payload, CreatedAt: time.Now()}

	if err := store.SaveXML(context.Background(), record); err != nil {
		t.Fatalf("save XML: %v", err)
	}
	if string(record.Payload) != string(original) {
		t.Fatalf("caller payload mutated: got %q want %q", record.Payload, original)
	}
	var persisted model.CommandXML
	if err := db.First(&persisted, "command_id = ?", record.CommandID).Error; err != nil {
		t.Fatalf("load XML: %v", err)
	}
	if strings.Contains(string(persisted.Payload), "database-secret") || !strings.Contains(string(persisted.Payload), "******") {
		t.Fatalf("persisted XML was not sanitized: %s", persisted.Payload)
	}
	if !strings.Contains(string(persisted.Payload), "download-secret") {
		t.Fatalf("unrelated password changed: %s", persisted.Payload)
	}
}

func TestCommandStoreSaveXMLRejectsMalformedPayloadWithoutCreatingRow(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	record := &model.CommandXML{
		CommandID: "cmd-malformed",
		Payload:   []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>database-secret</Envelope>`),
	}

	if err := store.SaveXML(context.Background(), record); !errors.Is(err, ErrInvalidCommandXML) {
		t.Fatalf("save error = %v, want ErrInvalidCommandXML", err)
	}
	var count int64
	if err := db.Model(new(model.CommandXML)).Where("command_id = ?", record.CommandID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("persisted rows = %d, want 0", count)
	}
}

func TestCommandStoreTransitionRejectsInvalidEventJSONWithoutChangingState(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	command := &model.Command{
		CommandID: "cmd-invalid-transition-json",
		DeviceID:  13,
		DeviceKey: "001122-SN13",
		Operation: "Reboot",
		Status:    model.CommandStatusSent,
		QueuedAt:  time.Now(),
	}
	if err := store.Create(context.Background(), command); err != nil {
		t.Fatalf("seed command: %v", err)
	}

	_, err := store.Transition(context.Background(), CommandTransition{
		CommandID:       command.CommandID,
		FromStatuses:    []string{model.CommandStatusSent},
		ToStatus:        model.CommandStatusCompleted,
		ExpectedVersion: 0,
		EventType:       "RESPONSE_COMPLETED",
		PayloadJSON:     datatypes.JSON(`{"broken"`),
	})
	if !errors.Is(err, ErrInvalidCommandJSON) {
		t.Fatalf("transition error = %v, want ErrInvalidCommandJSON", err)
	}

	var got model.Command
	if err := db.First(&got, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load command: %v", err)
	}
	if got.Status != model.CommandStatusSent || got.Version != 0 {
		t.Fatalf("state after rejected transition = %s/%d, want SENT/0", got.Status, got.Version)
	}
}

func TestCommandStoreTransitionRejectsInvalidResultJSONWithoutChangingState(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	command := &model.Command{
		CommandID: "cmd-invalid-result-json",
		DeviceID:  14,
		DeviceKey: "001122-SN14",
		Operation: "GetRPCMethods",
		Status:    model.CommandStatusSent,
		QueuedAt:  time.Now(),
	}
	if err := store.Create(context.Background(), command); err != nil {
		t.Fatalf("seed command: %v", err)
	}

	_, err := store.Transition(context.Background(), CommandTransition{
		CommandID:       command.CommandID,
		FromStatuses:    []string{model.CommandStatusSent},
		ToStatus:        model.CommandStatusCompleted,
		ExpectedVersion: 0,
		Updates: map[string]any{
			"result_json": datatypes.JSON(`{"broken"`),
		},
	})
	if !errors.Is(err, ErrInvalidCommandJSON) {
		t.Fatalf("transition error = %v, want ErrInvalidCommandJSON", err)
	}

	var got model.Command
	if err := db.First(&got, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load command: %v", err)
	}
	if got.Status != model.CommandStatusSent || got.Version != 0 {
		t.Fatalf("state after rejected transition = %s/%d, want SENT/0", got.Status, got.Version)
	}
}

func TestCommandStoreHeadForDeviceReturnsEarliestNonTerminalCommand(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	base := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	commands := []model.Command{
		{CommandID: "terminal-first", DeviceID: 21, Status: model.CommandStatusCompleted, CreatedAt: base},
		{CommandID: "other-device", DeviceID: 22, Status: model.CommandStatusWaitingDevice, CreatedAt: base.Add(time.Second)},
		{CommandID: "expected-head", DeviceID: 21, Status: model.CommandStatusQueued, CreatedAt: base.Add(2 * time.Second)},
		{CommandID: "later", DeviceID: 21, Status: model.CommandStatusWaitingDevice, CreatedAt: base.Add(3 * time.Second)},
	}
	if err := db.Create(&commands).Error; err != nil {
		t.Fatalf("seed commands: %v", err)
	}

	head, err := store.HeadForDevice(context.Background(), 21)
	if err != nil {
		t.Fatalf("head for device: %v", err)
	}
	if head.CommandID != "expected-head" {
		t.Fatalf("head command = %q, want expected-head", head.CommandID)
	}
}

func TestCommandStoreWaitingRebootBlocksFIFO(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	base := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	commands := []model.Command{
		{CommandID: "waiting-reboot-head", DeviceID: 23, Status: model.CommandStatusWaitingReboot, CreatedAt: base},
		{CommandID: "queued-behind-reboot", DeviceID: 23, Status: model.CommandStatusQueued, CreatedAt: base.Add(time.Second)},
	}
	if err := db.Create(&commands).Error; err != nil {
		t.Fatalf("seed commands: %v", err)
	}

	head, err := store.HeadForDevice(context.Background(), 23)
	if err != nil {
		t.Fatalf("head for device: %v", err)
	}
	if head.CommandID != "waiting-reboot-head" {
		t.Fatalf("head command = %q, want waiting-reboot-head", head.CommandID)
	}
}

func TestCommandStoreListFiltersAndPaginatesCommands(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	base := time.Date(2026, 7, 16, 13, 0, 0, 0, time.UTC)
	commands := []model.Command{
		{CommandID: "cmd-list-download-old", DeviceID: 31, DeviceKey: "001122-SN31", Operation: "Download", Status: model.CommandStatusCompleted, CreatedAt: base},
		{CommandID: "cmd-list-download-new", DeviceID: 31, DeviceKey: "001122-SN31", Operation: "Download", Status: model.CommandStatusCompleted, CreatedAt: base.Add(time.Minute)},
		{CommandID: "cmd-list-reboot", DeviceID: 31, DeviceKey: "001122-SN31", Operation: "Reboot", Status: model.CommandStatusFailed, CreatedAt: base.Add(2 * time.Minute)},
		{CommandID: "cmd-list-other-device", DeviceID: 32, DeviceKey: "001122-SN32", Operation: "Download", Status: model.CommandStatusCompleted, CreatedAt: base.Add(3 * time.Minute)},
	}
	if err := db.Create(&commands).Error; err != nil {
		t.Fatalf("seed commands: %v", err)
	}

	from := base.Add(-time.Second)
	to := base.Add(90 * time.Second)
	list, total, err := store.List(context.Background(), CommandListFilter{
		DeviceID:     31,
		DeviceSerial: "SN31",
		Operation:    "Download",
		Status:       model.CommandStatusCompleted,
		CommandID:    "cmd-list-download",
		CreatedFrom:  &from,
		CreatedTo:    &to,
		Offset:       1,
		Limit:        1,
	})
	if err != nil {
		t.Fatalf("list commands: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(list) != 1 || list[0].CommandID != "cmd-list-download-old" {
		t.Fatalf("paginated list = %#v, want older matching command", list)
	}
}

func TestCommandStoreDetailReturnsOrderedEventsAndXML(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	base := time.Date(2026, 7, 16, 14, 0, 0, 0, time.UTC)
	command := model.Command{
		CommandID: "cmd-detail",
		DeviceID:  41,
		DeviceKey: "001122-SN41",
		Operation: "Upload",
		Status:    model.CommandStatusWaitingTransfer,
		CreatedAt: base,
	}
	if err := db.Create(&command).Error; err != nil {
		t.Fatalf("seed command: %v", err)
	}
	events := []model.CommandEvent{
		{CommandID: command.CommandID, EventType: "SECOND", CreatedAt: base.Add(2 * time.Second)},
		{CommandID: command.CommandID, EventType: "FIRST", CreatedAt: base.Add(time.Second)},
	}
	if err := db.Create(&events).Error; err != nil {
		t.Fatalf("seed events: %v", err)
	}
	xmlRecords := []model.CommandXML{
		{CommandID: command.CommandID, Direction: "inbound", Payload: []byte("response"), CreatedAt: base.Add(4 * time.Second)},
		{CommandID: command.CommandID, Direction: "outbound", Payload: []byte("request"), CreatedAt: base.Add(3 * time.Second)},
	}
	if err := db.Create(&xmlRecords).Error; err != nil {
		t.Fatalf("seed XML: %v", err)
	}

	detail, err := store.Detail(context.Background(), command.CommandID)
	if err != nil {
		t.Fatalf("command detail: %v", err)
	}
	if detail.Command.CommandID != command.CommandID {
		t.Fatalf("detail command ID = %q", detail.Command.CommandID)
	}
	if len(detail.Events) != 2 || detail.Events[0].EventType != "FIRST" || detail.Events[1].EventType != "SECOND" {
		t.Fatalf("ordered events = %#v", detail.Events)
	}
	if len(detail.XML) != 2 || detail.XML[0].Direction != "outbound" || detail.XML[1].Direction != "inbound" {
		t.Fatalf("ordered XML = %#v", detail.XML)
	}
}

func TestCommandStoreDetailMergesCommandKeyWithoutMutatingPersistedParams(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	commandKey := "rpc-detail-command-key"
	command := model.Command{
		CommandID:  "cmd-detail-command-key",
		DeviceID:   42,
		DeviceKey:  "001122-SN42",
		Operation:  "Reboot",
		ParamsJSON: model.LongTextJSON(`{}`),
		CommandKey: &commandKey,
		Status:     model.CommandStatusSent,
		CreatedAt:  time.Now(),
	}
	if err := db.Create(&command).Error; err != nil {
		t.Fatalf("seed command: %v", err)
	}

	detail, err := store.Detail(context.Background(), command.CommandID)
	if err != nil {
		t.Fatalf("command detail: %v", err)
	}
	if got := string(detail.Command.ParamsJSON); got != `{"commandKey":"rpc-detail-command-key"}` {
		t.Fatalf("detail params = %s, want merged commandKey", got)
	}

	var persisted model.Command
	if err := db.First(&persisted, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("reload command: %v", err)
	}
	if got := string(persisted.ParamsJSON); got != `{}` {
		t.Fatalf("persisted params = %s, want original {}", got)
	}
}

func TestCommandStoreTransitionRollsBackStateWhenEventAppendFails(t *testing.T) {
	db := newCommandStoreTestDB(t)
	store := NewCommandStore(db)
	command := &model.Command{
		CommandID: "cmd-transition-rollback",
		DeviceID:  51,
		Status:    model.CommandStatusSent,
		QueuedAt:  time.Now(),
	}
	if err := store.Create(context.Background(), command); err != nil {
		t.Fatalf("seed command: %v", err)
	}
	injected := errors.New("injected transition event failure")
	if err := db.Callback().Create().Before("gorm:create").Register("test:fail_transition_event", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Name == "CommandEvent" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, err := store.Transition(context.Background(), CommandTransition{
		CommandID:       command.CommandID,
		FromStatuses:    []string{model.CommandStatusSent},
		ToStatus:        model.CommandStatusCompleted,
		ExpectedVersion: 0,
		EventType:       "RESPONSE_COMPLETED",
	})
	if !errors.Is(err, injected) {
		t.Fatalf("transition error = %v, want injected failure", err)
	}
	var got model.Command
	if err := db.First(&got, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load command: %v", err)
	}
	if got.Status != model.CommandStatusSent || got.Version != 0 {
		t.Fatalf("state after rollback = %s/%d, want SENT/0", got.Status, got.Version)
	}
	var eventCount int64
	if err := db.Model(new(model.CommandEvent)).Where("command_id = ?", command.CommandID).Count(&eventCount).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("events = %d, want only CREATED", eventCount)
	}
}

func TestCommandModelsUseCompatibleColumnTypesAndNullableUniqueCommandKey(t *testing.T) {
	db := newCommandStoreTestDB(t)
	assertDatabaseColumnType(t, db, new(model.Command), "params_json", "LONGTEXT")
	assertDatabaseColumnType(t, db, new(model.Command), "result_json", "LONGTEXT")
	assertDatabaseColumnType(t, db, new(model.CommandEvent), "payload_json", "LONGTEXT")
	assertDatabaseColumnType(t, db, new(model.CommandXML), "payload", "LONGBLOB")

	commandsWithoutKeys := []model.Command{
		{CommandID: "cmd-null-key-1", Status: model.CommandStatusQueued},
		{CommandID: "cmd-null-key-2", Status: model.CommandStatusQueued},
	}
	if err := db.Create(&commandsWithoutKeys).Error; err != nil {
		t.Fatalf("multiple NULL command keys must be allowed: %v", err)
	}
	commandKey := "rpc-unique-key"
	if err := db.Create(&model.Command{CommandID: "cmd-key-1", CommandKey: &commandKey, Status: model.CommandStatusQueued}).Error; err != nil {
		t.Fatalf("create first unique command key: %v", err)
	}
	if err := db.Create(&model.Command{CommandID: "cmd-key-2", CommandKey: &commandKey, Status: model.CommandStatusQueued}).Error; err == nil {
		t.Fatal("duplicate non-NULL command key was accepted")
	}
}

func assertDatabaseColumnType(t *testing.T, db *gorm.DB, table any, columnName, wantType string) {
	t.Helper()
	columns, err := db.Migrator().ColumnTypes(table)
	if err != nil {
		t.Fatalf("column types for %T: %v", table, err)
	}
	for _, column := range columns {
		if column.Name() == columnName {
			if got := strings.ToUpper(column.DatabaseTypeName()); got != wantType {
				t.Fatalf("%T.%s database type = %s, want %s", table, columnName, got, wantType)
			}
			return
		}
	}
	t.Fatalf("column %T.%s not found", table, columnName)
}

func pointerTo[T any](value T) *T {
	return &value
}
