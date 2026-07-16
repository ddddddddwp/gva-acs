package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"gorm.io/gorm"
)

const connectionRequestPasswordPlaceholder = "__GVA_TR069_CONNECTION_REQUEST_PASSWORD__"

type CommandPayloadHydrator interface {
	Hydrate(ctx context.Context, deviceID uint, operation string, params map[string]interface{}) error
}

type ConnectionProfilePayloadProtector struct {
	repository *ConnectionProfileRepository
}

func NewConnectionProfilePayloadProtector(repository *ConnectionProfileRepository) *ConnectionProfilePayloadProtector {
	return &ConnectionProfilePayloadProtector{repository: repository}
}

func (p *ConnectionProfilePayloadProtector) Protect(ctx context.Context, tx *gorm.DB, deviceID uint, operation, origin string, encoded []byte) ([]byte, error) {
	if operation != "SetParameterValues" {
		return append([]byte(nil), encoded...), nil
	}
	if p == nil || p.repository == nil {
		return nil, errors.New("connection profile payload protector is not initialized")
	}
	decoded, err := service.DecodeRPCRequest(operation, encoded)
	if err != nil {
		return nil, err
	}
	request, ok := decoded.(req.SetParameterValuesRequest)
	if !ok {
		return nil, fmt.Errorf("unexpected SetParameterValues request type %T", decoded)
	}
	username := ""
	passwordIndex := -1
	password := ""
	for index, parameter := range request.Parameters {
		switch parameter.Name {
		case connectionRequestUsernameName:
			username = fmt.Sprint(parameter.Value)
		case connectionRequestPasswordName:
			value := fmt.Sprint(parameter.Value)
			if value != connectionRequestPasswordPlaceholder {
				passwordIndex = index
				password = value
			}
		}
	}
	if passwordIndex < 0 {
		return append([]byte(nil), encoded...), nil
	}
	if username == "" {
		existingUsername, _, err := p.repository.LoadCredential(ctx, deviceID)
		if err != nil {
			return nil, errors.New("connection request username is required when setting its password")
		}
		username = existingUsername
	}
	source := model.ConnectionCredentialSourceManual
	if origin == model.CommandOriginSystem {
		source = model.ConnectionCredentialSourceAuto
	}
	if err := p.repository.StoreCredential(ctx, tx, deviceID, username, password, source); err != nil {
		return nil, err
	}
	request.Parameters[passwordIndex].Value = connectionRequestPasswordPlaceholder
	return service.EncodeRPCRequest(operation, request)
}

func (p *ConnectionProfilePayloadProtector) Hydrate(ctx context.Context, deviceID uint, operation string, params map[string]interface{}) error {
	if operation != "SetParameterValues" || params == nil {
		return nil
	}
	if p == nil || p.repository == nil {
		return errors.New("connection profile payload hydrator is not initialized")
	}
	parameters, ok := params["parameters"].([]map[string]interface{})
	if !ok {
		return errors.New("SetParameterValues parameters have an unexpected shape")
	}
	needsCredential := false
	for _, parameter := range parameters {
		if parameter["name"] == connectionRequestPasswordName && parameter["value"] == connectionRequestPasswordPlaceholder {
			needsCredential = true
			break
		}
	}
	if !needsCredential {
		return nil
	}
	_, password, err := p.repository.LoadCredential(ctx, deviceID)
	if err != nil {
		return err
	}
	for _, parameter := range parameters {
		if parameter["name"] == connectionRequestPasswordName && parameter["value"] == connectionRequestPasswordPlaceholder {
			parameter["value"] = password
		}
	}
	return nil
}

var _ service.CommandPayloadProtector = (*ConnectionProfilePayloadProtector)(nil)
var _ CommandPayloadHydrator = (*ConnectionProfilePayloadProtector)(nil)
