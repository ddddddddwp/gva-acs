package adapter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type channelWakeTokenSource struct {
	tokens chan string
}

func newChannelWakeTokenSource(size int) *channelWakeTokenSource {
	return &channelWakeTokenSource{tokens: make(chan string, size)}
}

func (s *channelWakeTokenSource) Push(deviceKey string) {
	s.tokens <- deviceKey
}

func (s *channelWakeTokenSource) Pop(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case token := <-s.tokens:
		return token, nil
	}
}

func newCommandWakeConsumerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.Command), new(model.CommandEvent)); err != nil {
		t.Fatalf("migrate wake consumer models: %v", err)
	}
	return db
}

func TestCommandWakeConsumerConsumesSubmitTokenAndTriggersConnectionRequest(t *testing.T) {
	db := newCommandWakeConsumerTestDB(t)
	now := time.Now()
	device := model.Device{OUI: "001122", SerialNumber: "WAKE-CONSUMER", LastInform: now.Add(-time.Second)}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	source := newChannelWakeTokenSource(4)
	triggered := make(chan uint, 1)
	consumer := newCommandWakeConsumer(db, source, func(_ context.Context, deviceID uint, _ ConnectionRequestConfig) ConnectionRequestResult {
		triggered <- deviceID
		return ConnectionRequestResult{StatusCode: 200}
	}, CommandWakeConsumerConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		consumer.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	manager := service.NewCommandManager(db, func(_ context.Context, deviceKey string) error {
		source.Push(deviceKey)
		return nil
	})
	result, err := manager.Submit(context.Background(), device.ID, "GetRPCMethods", nil)
	if err != nil {
		t.Fatalf("Submit() error: %v", err)
	}
	select {
	case gotDeviceID := <-triggered:
		if gotDeviceID != device.ID {
			t.Fatalf("triggered device ID = %d, want %d", gotDeviceID, device.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("wake token was not consumed")
	}
	if len(source.tokens) != 0 {
		t.Fatalf("wake queue length = %d, want drained", len(source.tokens))
	}
	var command model.Command
	if err := db.First(&command, "command_id = ?", result.CommandID).Error; err != nil {
		t.Fatalf("load command: %v", err)
	}
	if command.Status != model.CommandStatusWaitingDevice {
		t.Fatalf("consumer changed DB-authoritative head to %s", command.Status)
	}
	deadline := time.Now().Add(time.Second)
	for {
		var event model.CommandEvent
		err := db.Where("command_id = ? AND event_type = ?", result.CommandID, "WAKE_TRIGGERED").First(&event).Error
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("load WAKE_TRIGGERED event: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestCommandWakeConsumerRecordsBoundedConnectionRequestFailure(t *testing.T) {
	db := newCommandWakeConsumerTestDB(t)
	now := time.Now()
	device := model.Device{OUI: "001122", SerialNumber: "WAKE-FAIL", LastInform: now.Add(-time.Second)}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	command := model.Command{
		CommandID: "wake-failure", DeviceID: device.ID, DeviceKey: "001122-WAKE-FAIL", Operation: "GetRPCMethods",
		ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusWaitingDevice, QueuedAt: now, WaitingAt: &now, CreatedAt: now,
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed waiting command: %v", err)
	}
	source := newChannelWakeTokenSource(2)
	source.Push(command.DeviceKey)
	injected := errors.New("connection refused")
	var mu sync.Mutex
	calls := 0
	consumer := newCommandWakeConsumer(db, source, func(_ context.Context, _ uint, cfg ConnectionRequestConfig) ConnectionRequestResult {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if cfg.Retries != 1 {
			t.Errorf("connection request retries = %d, want 1", cfg.Retries)
		}
		return ConnectionRequestResult{Err: injected}
	}, CommandWakeConsumerConfig{FailureBackoff: 10 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		consumer.Run(ctx)
	}()

	deadline := time.Now().Add(time.Second)
	for {
		var event model.CommandEvent
		err := db.Where("command_id = ? AND event_type = ?", command.CommandID, "WAKE_FAILED").First(&event).Error
		if err == nil {
			if event.Stage != "connection_request" || event.Message != injected.Error() {
				t.Fatalf("WAKE_FAILED event = %#v", event)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("WAKE_FAILED event not persisted: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("consumer trigger calls = %d, want one bounded attempt per token", calls)
	}
	if len(source.tokens) != 0 {
		t.Fatalf("failed wake token was not drained: queue length %d", len(source.tokens))
	}
}

func TestCommandWakeConsumerToleratesDuplicateDeviceTokens(t *testing.T) {
	db := newCommandWakeConsumerTestDB(t)
	now := time.Now()
	device := model.Device{OUI: "001122", SerialNumber: "WAKE-DUPLICATE", LastInform: now.Add(-time.Second)}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	command := model.Command{
		CommandID: "wake-duplicate", DeviceID: device.ID, DeviceKey: "001122-WAKE-DUPLICATE", Operation: "GetRPCMethods",
		ParamsJSON: model.LongTextJSON(`{}`), Status: model.CommandStatusWaitingDevice, QueuedAt: now, WaitingAt: &now, CreatedAt: now,
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed waiting command: %v", err)
	}
	source := newChannelWakeTokenSource(2)
	source.Push(command.DeviceKey)
	source.Push(command.DeviceKey)
	var mu sync.Mutex
	triggerCalls := 0
	consumer := newCommandWakeConsumer(db, source, func(context.Context, uint, ConnectionRequestConfig) ConnectionRequestResult {
		mu.Lock()
		triggerCalls++
		mu.Unlock()
		return ConnectionRequestResult{StatusCode: 200}
	}, CommandWakeConsumerConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		consumer.Run(ctx)
	}()

	deadline := time.Now().Add(time.Second)
	for {
		var count int64
		if err := db.Model(new(model.CommandEvent)).
			Where("command_id = ? AND event_type = ?", command.CommandID, "WAKE_TRIGGERED").
			Count(&count).Error; err != nil {
			t.Fatalf("count duplicate wake events: %v", err)
		}
		if count == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("duplicate wake tokens produced %d events, want 2", count)
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done
	mu.Lock()
	defer mu.Unlock()
	if triggerCalls != 2 || len(source.tokens) != 0 {
		t.Fatalf("duplicate token calls/remaining = %d/%d, want 2/0", triggerCalls, len(source.tokens))
	}
	var current model.Command
	if err := db.First(&current, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load command: %v", err)
	}
	if current.Status != model.CommandStatusWaitingDevice {
		t.Fatalf("duplicate wake tokens changed durable status to %s", current.Status)
	}
}
