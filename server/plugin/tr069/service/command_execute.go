package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
)

type CommandExecuteResult struct {
	CommandID string                 `json:"commandId"`
	Status    string                 `json:"status"`
	Values    map[string]interface{} `json:"values,omitempty"`
}

func (s *CommandService) ExecuteGetParameterValues(deviceID uint, in req.GetParameterValuesRequest, timeout time.Duration) (CommandExecuteResult, error) {
	if len(in.Paths) == 0 {
		return CommandExecuteResult{}, errors.New("paths is empty")
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	cmdID, err := s.EnqueueGetParameterValues(deviceID, in)
	if err != nil {
		return CommandExecuteResult{}, err
	}
	status, err := waitCommandFinal(context.Background(), cmdID, timeout)
	if err != nil {
		return CommandExecuteResult{CommandID: cmdID, Status: "TIMEOUT"}, nil
	}
	if status != "SUCCESS" {
		return CommandExecuteResult{CommandID: cmdID, Status: status}, nil
	}
	values, _ := readDataModelValues(deviceID, in.Paths)
	return CommandExecuteResult{CommandID: cmdID, Status: status, Values: values}, nil
}

func waitCommandFinal(ctx context.Context, commandID string, timeout time.Duration) (string, error) {
	if global.GVA_DB == nil {
		return "", errors.New("db not initialized")
	}
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return "", errors.New("timeout")
		}
		var cmd model.Command
		if err := global.GVA_DB.WithContext(ctx).Select("command_id", "status").Where("command_id = ?", commandID).First(&cmd).Error; err == nil {
			switch cmd.Status {
			case "SUCCESS", "FAIL":
				return cmd.Status, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func readDataModelValues(deviceID uint, paths []string) (map[string]interface{}, error) {
	if global.GVA_DB == nil {
		return nil, errors.New("db not initialized")
	}
	if len(paths) == 0 {
		return map[string]interface{}{}, nil
	}
	var rows []model.DataModelValue
	if err := global.GVA_DB.Select("name", "value_json").Where("device_id = ? AND name IN ?", deviceID, paths).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]interface{}, len(rows))
	for _, r := range rows {
		var v interface{}
		if len(r.ValueJSON) > 0 {
			_ = json.Unmarshal(r.ValueJSON, &v)
		}
		out[r.Name] = v
	}
	return out, nil
}
