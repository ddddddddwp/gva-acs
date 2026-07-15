# TR-069 Core RPC Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the independent `tr069-core-only` protocol layer needed by the GVA plugin: construct all 12 device RPCs, preserve complete inbound/outbound XML events, correlate every normal response, and expose `TransferComplete` without coupling core to GVA models.

**Architecture:** Keep CWMP types, XML codec, executors, correlation, and observability in the standalone core repository. Add an optional transfer-completion hook so existing integrations remain source-compatible. GVA supplies the lifecycle hook and event sink; core only emits protocol facts and invokes repository/hook contracts.

**Tech Stack:** Go, `encoding/xml`, existing `tr069-core-only` interfaces/defaults, table-driven Go tests.

## Global Constraints

- Work only in `/root/code/gva-acs/gva-acs/server/plugin/tr069/lib/tr069-core-only` for this plan.
- The directory is an independent Git repository on `feat/xml-observability`; never stage it from the parent GVA repository.
- Follow test-driven development: add one failing test, run it and confirm the expected failure, implement the minimum code, rerun the focused test, then run the package suite.
- Do not add JSON wire logging. INFO carries the exact XML bytes; DEBUG carries internal parser/builder stages; ERROR carries construction/parsing failures.
- Do not redact or rewrite XML payloads. Copy payload slices before emitting so sinks cannot observe later buffer mutation.
- Preserve existing exported interfaces where possible. New transfer handling is an optional companion interface, not a new required method on `CorrelationHook`.
- Do not push the nested repository until the GVA integration is verified against the resulting commit.

---

### Task 1: Model complete `SetParameterAttributes` semantics

**Files:**

- Modify: `interface/types.go`
- Modify: `internal/types/rpc_more.go`
- Modify: `internal/builder/builder.go`
- Modify: `internal/parser/parser.go`
- Test: `internal/builder/builder_new_methods_test.go`
- Test: `internal/parser/parser_new_methods_test.go`

- [ ] Add a failing builder test proving both change flags are serialized, including explicit `false` values.

```go
func TestBuilder_BuildMessage_SetParameterAttributesChangeFlags(t *testing.T) {
	b := New()
	out, err := b.BuildMessage(context.Background(), &tr069.Message{
		Method: tr069.MethodSetParameterAttributes,
		ParameterAttributes: []tr069.ParameterAttribute{{
			Name:               "Device.ManagementServer.PeriodicInformEnable",
			NotificationChange: true,
			Notification:       2,
			AccessListChange:   false,
			AccessList:         []string{"Subscriber"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	xmlText := string(out)
	for _, want := range []string{
		"<NotificationChange>true</NotificationChange>",
		"<Notification>2</Notification>",
		"<AccessListChange>false</AccessListChange>",
		"<string>Subscriber</string>",
	} {
		if !strings.Contains(xmlText, want) {
			t.Fatalf("missing %q in %s", want, xmlText)
		}
	}
}
```

- [ ] Run the test and verify it fails because `ParameterAttribute` has no change-flag fields.

Run: `go test ./internal/builder -run TestBuilder_BuildMessage_SetParameterAttributesChangeFlags -count=1`

- [ ] Split the protocol-facing getter and setter XML shapes while keeping one public value type.

```go
type ParameterAttribute struct {
	Name               string   `json:"name"`
	NotificationChange bool     `json:"notificationChange"`
	Notification       int      `json:"notification"`
	AccessListChange   bool     `json:"accessListChange"`
	AccessList         []string `json:"accessList"`
}
```

In `internal/types/rpc_more.go`, keep `ParameterAttributeStruct` for `GetParameterAttributesResponse` and add the exact request shape:

```go
type SetParameterAttributeStruct struct {
	XMLName           xml.Name   `xml:"SetParameterAttributesStruct"`
	Name              string     `xml:"Name"`
	NotificationChange bool      `xml:"NotificationChange"`
	Notification      int        `xml:"Notification"`
	AccessListChange  bool       `xml:"AccessListChange"`
	AccessList        AccessList `xml:"AccessList"`
}

type SetParameterAttributeList struct {
	XMLName    xml.Name                      `xml:"ParameterList"`
	ArrayType  string                        `xml:"SOAP-ENC:arrayType,attr,omitempty"`
	Parameters []SetParameterAttributeStruct `xml:"SetParameterAttributesStruct"`
}

type SetParameterAttributes struct {
	XMLName       xml.Name                     `xml:"SetParameterAttributes"`
	ParameterList SetParameterAttributeList    `xml:"ParameterList"`
}
```

- [ ] Update the builder to map both booleans and set `ArrayType` deterministically to `cwmp:SetParameterAttributesStruct[N]`.

- [ ] Add a failing parser test with `NotificationChange=false` and `AccessListChange=true`, then map both fields in `internal/parser/parser.go`.

- [ ] Run focused codec tests.

Run: `go test ./internal/builder ./internal/parser -run 'ParameterAttributes' -count=1`

- [ ] Commit the completed type/codec slice.

```bash
git add interface/types.go internal/types/rpc_more.go internal/builder/builder.go internal/parser/parser.go internal/builder/builder_new_methods_test.go internal/parser/parser_new_methods_test.go
git commit -m "feat: complete parameter attribute XML semantics"
```

### Task 2: Construct all 12 advertised device RPCs

**Files:**

- Modify: `pkg/core/defaults/executors.go`
- Modify: `pkg/core/defaults/executors_extra.go`
- Test: `pkg/core/defaults/executors_test.go`

- [ ] Add table-driven failing tests for `AddObject`, `DeleteObject`, and `FactoryReset`.

```go
func TestObjectAndFactoryExecutors(t *testing.T) {
	tests := []struct {
		name string
		exec core.Executor
		params map[string]interface{}
		method string
		assert func(*testing.T, *tr069.Message)
	}{
		{
			name: "add object",
			exec: &AddObjectExecutor{},
			params: map[string]interface{}{"objectName": "Device.WiFi.SSID.", "parameterKey": "add-1"},
			method: tr069.MethodAddObject,
			assert: func(t *testing.T, msg *tr069.Message) {
				if msg.ObjectName != "Device.WiFi.SSID." || msg.ParameterKey != "add-1" { t.Fatalf("unexpected message: %#v", msg) }
			},
		},
		{
			name: "delete object",
			exec: &DeleteObjectExecutor{},
			params: map[string]interface{}{"objectName": "Device.WiFi.SSID.7.", "parameterKey": "del-1"},
			method: tr069.MethodDeleteObject,
			assert: func(t *testing.T, msg *tr069.Message) {
				if msg.ObjectName != "Device.WiFi.SSID.7." || msg.ParameterKey != "del-1" { t.Fatalf("unexpected message: %#v", msg) }
			},
		},
		{name: "factory reset", exec: &FactoryResetExecutor{}, params: map[string]interface{}{}, method: tr069.MethodFactoryReset, assert: func(*testing.T, *tr069.Message) {}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := tt.exec.BuildRequest(context.Background(), &core.Session{}, &core.Command{Params: tt.params})
			if err != nil { t.Fatal(err) }
			msg, ok := raw.(*tr069.Message)
			if !ok || msg.Method != tt.method { t.Fatalf("unexpected request: %#v", raw) }
			tt.assert(t, msg)
		})
	}
}
```

- [ ] Run the focused test and confirm the three executor types are undefined.

Run: `go test ./pkg/core/defaults -run TestObjectAndFactoryExecutors -count=1`

- [ ] Implement the three executors using the same parameter keys GVA persists.

```go
type AddObjectExecutor struct{}

func (*AddObjectExecutor) BuildRequest(_ context.Context, _ *core.Session, cmd *core.Command) (interface{}, error) {
	params := commandParams(cmd)
	return &tr069.Message{
		Method:       tr069.MethodAddObject,
		ObjectName:   asString(params["objectName"]),
		ParameterKey: asString(params["parameterKey"]),
	}, nil
}

type DeleteObjectExecutor struct{}

func (*DeleteObjectExecutor) BuildRequest(_ context.Context, _ *core.Session, cmd *core.Command) (interface{}, error) {
	params := commandParams(cmd)
	return &tr069.Message{
		Method:       tr069.MethodDeleteObject,
		ObjectName:   asString(params["objectName"]),
		ParameterKey: asString(params["parameterKey"]),
	}, nil
}

type FactoryResetExecutor struct{}

func (*FactoryResetExecutor) BuildRequest(context.Context, *core.Session, *core.Command) (interface{}, error) {
	return &tr069.Message{Method: tr069.MethodFactoryReset}, nil
}

func commandParams(cmd *core.Command) map[string]interface{} {
	if cmd == nil || cmd.Params == nil { return map[string]interface{}{} }
	return cmd.Params
}
```

- [ ] Extend `SetParameterAttributesExecutor` to accept JSON-decoded `[]interface{}` and numeric `float64`, and populate both change flags. Add `asBool` and reuse `asInt`.

- [ ] Add a single table test covering method selection for the complete supported set:

```go
[]string{
	tr069.MethodGetRPCMethods,
	tr069.MethodGetParameterValues,
	tr069.MethodGetParameterNames,
	tr069.MethodGetParameterAttributes,
	tr069.MethodSetParameterValues,
	tr069.MethodSetParameterAttributes,
	tr069.MethodAddObject,
	tr069.MethodDeleteObject,
	tr069.MethodDownload,
	tr069.MethodUpload,
	tr069.MethodReboot,
	tr069.MethodFactoryReset,
}
```

- [ ] Run the executor and XML suites.

Run: `go test ./pkg/core/defaults ./internal/builder -count=1`

- [ ] Commit the executor slice.

```bash
git add pkg/core/defaults/executors.go pkg/core/defaults/executors_extra.go pkg/core/defaults/executors_test.go
git commit -m "feat: construct complete device RPC set"
```

### Task 3: Complete normal-response and transfer correlation

**Files:**

- Modify: `pkg/core/correlation_hook.go`
- Modify: `pkg/core/inflight.go`
- Modify: `pkg/core/machine.go`
- Modify: `pkg/core/defaults/inflight_correlation_hook.go`
- Test: `pkg/core/inflight_test.go`
- Test: `pkg/core/machine_transfer_test.go`

- [ ] Add failing correlation cases for `GetParameterAttributesResponse` and `SetParameterAttributesResponse` to `pkg/core/inflight_test.go`.

```go
tests := []struct{ request, response string }{
	{tr069.MethodGetParameterAttributes, tr069.MethodGetParameterAttributesResponse},
	{tr069.MethodSetParameterAttributes, tr069.MethodSetParameterAttributesResponse},
}
```

- [ ] Add both mappings to `isExpectedResponse` and run the focused test.

Run: `go test ./pkg/core -run 'Correlation' -count=1`

- [ ] Add an optional transfer hook; do not alter the required `CorrelationHook` method set.

```go
type TransferCompleteHook interface {
	OnTransferComplete(ctx context.Context, session *Session, complete *tr069.Message) (handled bool, err error)
}
```

- [ ] Add a fake-hook test proving `Machine.handleTransferComplete` invokes the optional hook and always returns `TransferCompleteResponse` with the incoming CWMP ID.

- [ ] Invoke the optional hook before constructing the acknowledgement.

```go
func (m *Machine) handleTransferComplete(ctx context.Context, session *Session, msg *tr069.Message) (*tr069.Message, error) {
	if hook, ok := m.conf.CorrelationHook.(TransferCompleteHook); ok {
		if _, err := hook.OnTransferComplete(ctx, session, msg); err != nil {
			return nil, err
		}
	}
	return &tr069.Message{Method: tr069.MethodTransferCompleteResponse, ID: msg.ID}, nil
}
```

- [ ] Change the default inflight hook so `DownloadResponse` and `UploadResponse` with `Status == 1` are not marked successful. Delete the request/CWMP inflight entry after the response, because later association uses `CommandKey`, not CWMP ID.

- [ ] Add tests for immediate transfer completion (`Status == 0`) and deferred transfer completion (`Status == 1`). The default hook only withholds success for deferred transfers; GVA's hook owns the durable `WAITING_TRANSFER` transition.

- [ ] Run the core state-machine suite.

Run: `go test ./pkg/core/... -count=1`

- [ ] Commit the correlation slice.

```bash
git add pkg/core/correlation_hook.go pkg/core/inflight.go pkg/core/machine.go pkg/core/defaults/inflight_correlation_hook.go pkg/core/inflight_test.go pkg/core/machine_transfer_test.go
git commit -m "feat: expose deferred transfer completion lifecycle"
```

### Task 4: Emit complete inbound and outbound XML at INFO

**Files:**

- Modify: `parser/auto.go`
- Modify: `internal/builder/builder.go`
- Test: `parser/auto_observability_test.go`
- Test: `internal/builder/builder_observability_test.go`

- [ ] Add a failing parser observability test that expects one exact inbound XML event in addition to DEBUG stage events.

```go
func TestAutoParserEmitsExactInboundXML(t *testing.T) {
	sink := newRecordingSink(observability.LevelInfo)
	p := factory.NewParser(tr069.WithEventSink(sink))
	raw := []byte(sampleInformXML)
	if _, err := p.ParseMessage(context.Background(), raw); err != nil { t.Fatal(err) }
	var event observability.Event
	for _, candidate := range sink.Events() {
		if candidate.Stage == "wire.xml" { event = candidate }
	}
	if event.Stage == "" { t.Fatal("wire.xml event not emitted") }
	if event.Level != observability.LevelInfo || event.Direction != observability.DirectionInbound {
		t.Fatalf("unexpected event: %#v", event)
	}
	if !bytes.Equal(event.Payload, raw) { t.Fatalf("payload changed") }
}
```

- [ ] Emit `wire.xml` from `AutoParser.ParseMessage` after parsing so CWMP ID and method are known. Emit even when parsing fails, with empty method attributes and the exact payload.

```go
func (a *AutoParser) ParseMessage(ctx context.Context, data []byte) (*tr069.Message, error) {
	msg, err := a.ParseMessageReader(ctx, bytes.NewReader(data))
	if a.sink.Enabled(observability.LevelInfo) {
		attrs := observability.Attributes{}
		if msg != nil { attrs.CWMPID, attrs.Method = msg.ID, msg.Method }
		a.sink.Emit(ctx, observability.Enrich(ctx, observability.Event{
			Level: observability.LevelInfo, Stage: "wire.xml", Direction: observability.DirectionInbound,
			Attributes: attrs, Payload: append([]byte(nil), data...),
		}))
	}
	return msg, err
}
```

- [ ] Add a failing builder test expecting exact outbound XML bytes from the `wire.xml` event.

- [ ] Refactor the builder's final return into `result = append(...)`, emit after the final declaration-prefixed result exists, and return `result, nil`.

```go
result = append([]byte(`<?xml version="1.0" encoding="UTF-8"?>`), result...)
if b.sink.Enabled(observability.LevelInfo) {
	b.sink.Emit(ctx, observability.Enrich(ctx, observability.Event{
		Level: observability.LevelInfo, Stage: "wire.xml", Direction: observability.DirectionOutbound,
		Attributes: attributes, Payload: append([]byte(nil), result...),
	}))
}
return result, nil
```

- [ ] Verify INFO emits only the wire event, DEBUG emits wire plus internal stages, and ERROR still records a failed stage without fabricating outbound XML.

Run: `go test ./parser ./internal/builder ./observability -count=1`

- [ ] Commit the wire-observability slice.

```bash
git add parser/auto.go parser/auto_observability_test.go internal/builder/builder.go internal/builder/builder_observability_test.go
git commit -m "feat: emit exact CWMP XML wire events"
```

### Task 5: Fail commands explicitly when core cannot construct them

**Files:**

- Modify: `pkg/core/machine.go`
- Test: `pkg/core/machine_command_failure_test.go`

- [ ] Add fake `CommandRepo`, `CommandSource`, and executor tests for an unknown operation and a `BuildRequest` error. Both commands must call `MarkFail`; neither may produce an outbound RPC.

- [ ] Use CWMP fault code `9003` (`Invalid arguments`) for executor validation/construction errors and code `9000` (`Method not supported`) when no executor is registered. Include the operation and original error in `FaultString`.

- [ ] Release the Redis/device lock through `ack` after the durable failure is recorded. If `MarkFail` itself fails, call `nack` so the command remains recoverable.

- [ ] Treat `CorrelationHook.OnRequestBuilt` failure as a construction-stage failure: do not send the request, mark the command failed, and release the queue lock.

- [ ] Run focused tests and the full core suite.

Run: `go test ./pkg/core -run 'Command.*Fail|RequestBuilt' -count=1`

Run: `go test ./... -count=1`

- [ ] Commit and record the exact core revision for the GVA plan.

```bash
git add pkg/core/machine.go pkg/core/machine_command_failure_test.go
git commit -m "fix: terminate commands that cannot be constructed"
git status --short
git rev-parse HEAD
```

Expected final state: `git status --short` prints nothing. Copy the `git rev-parse HEAD` value into the GVA integration commit message or handoff notes; the parent repository must continue to ignore this directory.
