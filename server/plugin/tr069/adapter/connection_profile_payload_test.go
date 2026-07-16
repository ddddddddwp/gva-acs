package adapter

import (
	"context"
	"strings"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"gorm.io/gorm"
)

func TestConnectionProfilePayloadIsProtectedAtRestAndHydratedInMemory(t *testing.T) {
	repository, db := newConnectionProfileRepositoryTest(t)
	device := createConnectionProfileDevice(t, db, "PROFILE-PAYLOAD")
	if _, err := repository.Collect(context.Background(), device.ID, map[string]string{
		connectionRequestURLName: "http://127.0.0.1:8400",
	}); err != nil {
		t.Fatalf("collect URL: %v", err)
	}
	protector := NewConnectionProfilePayloadProtector(repository)
	request := req.SetParameterValuesRequest{Parameters: []req.SetParameterValue{
		{Name: connectionRequestUsernameName, Type: "xsd:string", Value: "payload-user"},
		{Name: connectionRequestPasswordName, Type: "xsd:string", Value: "payload-secret"},
	}}
	encoded, err := service.EncodeRPCRequest("SetParameterValues", request)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	var protected []byte
	if err := db.Transaction(func(tx *gorm.DB) error {
		var protectErr error
		protected, protectErr = protector.Protect(context.Background(), tx, device.ID, "SetParameterValues", model.CommandOriginSystem, encoded)
		return protectErr
	}); err != nil {
		t.Fatalf("protect: %v", err)
	}
	if strings.Contains(string(protected), "payload-secret") || !strings.Contains(string(protected), connectionRequestPasswordPlaceholder) {
		t.Fatalf("protected payload=%s", protected)
	}

	params, err := service.DecodeRPCParams("SetParameterValues", protected)
	if err != nil {
		t.Fatalf("decode protected params: %v", err)
	}
	if err := protector.Hydrate(context.Background(), device.ID, "SetParameterValues", params); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	parameters, ok := params["parameters"].([]map[string]interface{})
	if !ok || len(parameters) != 2 {
		t.Fatalf("normalized parameters=%#v", params["parameters"])
	}
	if got := parameters[1]["value"]; got != "payload-secret" {
		t.Fatalf("hydrated password=%#v", got)
	}
	profile := loadConnectionProfile(t, db, device.ID)
	if profile.CredentialSource != model.ConnectionCredentialSourceAuto || profile.Username != "payload-user" {
		t.Fatalf("profile=%#v", profile)
	}
}
