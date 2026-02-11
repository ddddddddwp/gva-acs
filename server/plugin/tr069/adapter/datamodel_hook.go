package adapter

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

type DataModelHook struct {
	base     core.CorrelationHook
	inflight core.InflightRepo
	ingest   core.CommandIngest
	now      func() time.Time
}

type DataModelHookOption func(*DataModelHook)

func WithDataModelHookNow(now func() time.Time) DataModelHookOption {
	return func(h *DataModelHook) {
		h.now = now
	}
}

func NewDataModelHook(base core.CorrelationHook, inflight core.InflightRepo, ingest core.CommandIngest, opts ...DataModelHookOption) *DataModelHook {
	h := &DataModelHook{
		base:     base,
		inflight: inflight,
		ingest:   ingest,
		now:      time.Now,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *DataModelHook) OnRequestBuilt(ctx context.Context, session *core.Session, cmd *core.Command, req *tr069.Message) error {
	if h == nil {
		return nil
	}
	if h.base != nil {
		return h.base.OnRequestBuilt(ctx, session, cmd, req)
	}
	return nil
}

func (h *DataModelHook) OnResponse(ctx context.Context, session *core.Session, resp *tr069.Message) (bool, error) {
	if h == nil {
		return false, nil
	}
	req, found := h.lookupInflight(ctx, session, resp)
	if found {
		_ = h.handleDataModelResponse(ctx, session, req, resp)
	}
	if h.base != nil {
		return h.base.OnResponse(ctx, session, resp)
	}
	return found, nil
}

func (h *DataModelHook) OnFault(ctx context.Context, session *core.Session, fault *tr069.Message) (bool, error) {
	if h == nil {
		return false, nil
	}
	if h.base != nil {
		return h.base.OnFault(ctx, session, fault)
	}
	return false, nil
}

func (h *DataModelHook) lookupInflight(ctx context.Context, session *core.Session, resp *tr069.Message) (core.InflightRequest, bool) {
	if h == nil || h.inflight == nil || session == nil || resp == nil {
		return core.InflightRequest{}, false
	}
	if session.DeviceKey == "" || resp.ID == "" {
		return core.InflightRequest{}, false
	}
	req, found, err := h.inflight.GetByCwmpID(ctx, session.DeviceKey, resp.ID)
	if err != nil || !found {
		return core.InflightRequest{}, false
	}
	return req, true
}

func (h *DataModelHook) handleDataModelResponse(ctx context.Context, session *core.Session, req core.InflightRequest, resp *tr069.Message) error {
	if global.GVA_DB == nil || session == nil || resp == nil {
		return nil
	}
	deviceID, ok := h.resolveDeviceNumericID(ctx, session)
	if !ok || deviceID == 0 {
		return nil
	}

	switch req.RequestName {
	case tr069.MethodGetParameterValues:
		if resp.Method != tr069.MethodGetParameterValuesResponse {
			return nil
		}
		return h.persistGPV(ctx, deviceID, resp.Parameters)
	case tr069.MethodGetParameterNames:
		if resp.Method != tr069.MethodGetParameterNamesResponse {
			return nil
		}
		return h.persistAndExpandGPN(ctx, deviceID, session.DeviceKey, req.CommandID, resp.ParameterInfos)
	case tr069.MethodGetRPCMethods:
		if resp.Method != tr069.MethodGetRPCMethodsResponse {
			return nil
		}
		return h.persistRPCMethods(ctx, deviceID, resp.Methods)
	default:
		return nil
	}
}

func (h *DataModelHook) resolveDeviceNumericID(ctx context.Context, session *core.Session) (uint, bool) {
	if global.GVA_DB == nil || session == nil {
		return 0, false
	}
	if session.DeviceID != "" {
		var d model.Device
		if err := global.GVA_DB.WithContext(ctx).Select("id").Where("serial_number = ?", session.DeviceID).First(&d).Error; err == nil {
			return d.ID, true
		}
	}
	oui, sn, ok := strings.Cut(session.DeviceKey, "-")
	if !ok || sn == "" {
		return 0, false
	}
	var d model.Device
	if err := global.GVA_DB.WithContext(ctx).Select("id").Where("serial_number = ? AND oui = ?", sn, oui).First(&d).Error; err != nil {
		return 0, false
	}
	return d.ID, true
}

func (h *DataModelHook) persistGPV(ctx context.Context, deviceID uint, params []tr069.Parameter) error {
	now := h.now()
	for _, p := range params {
		if p.Name == "" {
			continue
		}

		valType := normalizeValueType(p.Type)
		val := castValue(valType, p.Value)

		b, err := json.Marshal(val)
		if err != nil {
			continue
		}
		rec := model.DataModelValue{
			DeviceID:        deviceID,
			Name:            p.Name,
			Writable:        false,
			ValueType:       valType,
			ValueJSON:       b,
			LastCollectedAt: now,
		}
		_ = global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "device_id"}, {Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"value_type", "value_json", "last_collected_at", "updated_at"}),
		}).Create(&rec).Error
	}
	return nil
}

func (h *DataModelHook) persistAndExpandGPN(ctx context.Context, deviceID uint, deviceKey string, commandID string, infos []tr069.ParameterInfo) error {
	now := h.now()
	for _, info := range infos {
		if info.Name == "" {
			continue
		}
		// Skip saving objects (paths ending with .)
		if strings.HasSuffix(info.Name, ".") {
			continue
		}
		rec := model.DataModelValue{
			DeviceID:        deviceID,
			Name:            info.Name,
			Writable:        info.Writable,
			ValueType:       "",
			ValueJSON:       []byte("null"),
			LastCollectedAt: now,
		}
		_ = global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "device_id"}, {Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"writable", "last_collected_at", "updated_at"}),
		}).Create(&rec).Error
	}

	cmdPath := ""
	depth := 0
	maxDepth := 0
	if commandID != "" {
		var cmd model.Command
		if err := global.GVA_DB.WithContext(ctx).Select("params_json").Where("command_id = ?", commandID).First(&cmd).Error; err == nil {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(cmd.ParamsJSON), &m); err == nil {
				if v, ok := m["parameterPath"].(string); ok {
					cmdPath = v
				}
				if v, ok := m["depth"].(float64); ok {
					depth = int(v)
				}
				if v, ok := m["maxDepth"].(float64); ok {
					maxDepth = int(v)
				}
			}
		}
	}
	if maxDepth > 0 && depth >= maxDepth {
		return nil
	}
	if cmdPath == "" {
		cmdPath = "Device."
	}

	objects := make([]string, 0, len(infos))
	leaves := make([]string, 0, len(infos))
	for _, info := range infos {
		if info.Name == "" {
			continue
		}
		if strings.HasSuffix(info.Name, ".") {
			objects = append(objects, info.Name)
		} else {
			leaves = append(leaves, info.Name)
		}
	}

	for _, obj := range objects {
		_ = h.enqueue(ctx, deviceKey, "GetParameterNames", map[string]interface{}{
			"parameterPath": obj,
			"nextLevel":     true,
			"depth":         depth + 1,
			"maxDepth":      maxDepth,
		}, dedupKey("gpn", deviceKey, obj))
	}

	for _, chunk := range chunkStrings(leaves, 50) {
		_ = h.enqueue(ctx, deviceKey, "GetParameterValues", map[string]interface{}{
			"paths": chunk,
		}, dedupKey("gpv", deviceKey, strings.Join(chunk, "\x1f")))
	}
	return nil
}

func (h *DataModelHook) persistRPCMethods(ctx context.Context, deviceID uint, methods []string) error {
	b, err := json.Marshal(methods)
	if err != nil {
		return nil
	}
	rec := model.DeviceRPCMethods{
		DeviceID:    deviceID,
		MethodsJSON: b,
	}
	return global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"methods_json", "updated_at"}),
	}).Create(&rec).Error
}

func (h *DataModelHook) enqueue(ctx context.Context, deviceKey string, op string, params map[string]interface{}, dedupKey string) error {
	if h == nil || h.ingest == nil || deviceKey == "" || op == "" {
		return nil
	}
	cmdID := uuid.NewString()
	paramsJSON := ""
	if params != nil {
		if b, err := json.Marshal(params); err == nil {
			paramsJSON = string(b)
		}
	}
	if global.GVA_DB != nil {
		_ = global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "command_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"device_key",
				"operation",
				"params_json",
				"dedup_key",
				"status",
				"updated_at",
			}),
		}).Create(&model.Command{
			CommandID:  cmdID,
			DeviceKey:  deviceKey,
			Operation:  op,
			ParamsJSON: paramsJSON,
			DedupKey:   dedupKey,
			Status:     "PENDING",
			CreatedAt:  h.now(),
			UpdatedAt:  h.now(),
		}).Error
	}
	return h.ingest.Enqueue(ctx, &core.Command{
		ID:        cmdID,
		DeviceKey: deviceKey,
		Operation: op,
		Params:    params,
		DedupKey:  dedupKey,
		CreatedAt: h.now(),
	})
}

func chunkStrings(in []string, n int) [][]string {
	if n <= 0 || len(in) == 0 {
		return nil
	}
	out := make([][]string, 0, (len(in)+n-1)/n)
	for i := 0; i < len(in); i += n {
		j := i + n
		if j > len(in) {
			j = len(in)
		}
		out = append(out, append([]string(nil), in[i:j]...))
	}
	return out
}

func dedupKey(kind string, deviceKey string, seed string) string {
	sum := sha1.Sum([]byte(kind + ":" + deviceKey + ":" + seed))
	return "dm:" + kind + ":" + deviceKey + ":" + hex.EncodeToString(sum[:8])
}

func castValue(valType string, val interface{}) interface{} {
	var s string
	switch v := val.(type) {
	case nil:
		s = ""
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return v
	}

	s = strings.TrimSpace(s)

	switch valType {
	case "boolean":
		if s == "" {
			return nil
		}
		return s == "1" || strings.ToLower(s) == "true"
	case "int", "integer", "long":
		if s == "" {
			return nil
		}
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return v
		}
	case "unsignedInt", "unsignedLong":
		if s == "" {
			return nil
		}
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			return v
		}
	}

	if valType == "string" {
		if s == "" {
			return nil
		}
		return s
	}
	if s == "" {
		return nil
	}
	return s
}

func normalizeValueType(valType string) string {
	valType = strings.TrimSpace(valType)
	if valType == "" {
		return ""
	}
	if i := strings.IndexByte(valType, ':'); i >= 0 {
		return valType[i+1:]
	}
	return valType
}

var _ core.CorrelationHook = (*DataModelHook)(nil)
