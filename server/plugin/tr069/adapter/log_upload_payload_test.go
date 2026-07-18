package adapter

import (
	"context"
	"strings"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
)

func TestLogUploadPayloadCodecProtectsAtRestAndHydratesCurrentRuntime(t *testing.T) {
	previous := config.CurrentRuntime()
	t.Cleanup(func() { config.StoreRuntime(previous.Settings) })
	storeLogUploadRuntime("shared-log-user", "shared-log-password")
	codec := new(LogUploadPayloadCodec)
	encoded, err := service.EncodeRPCRequest("Upload", req.UploadRequest{
		FileType: "Vendor Log File", URL: "http://old/acs/log", Username: "shared-log-user", Password: "shared-log-password", DelaySeconds: 5,
	})
	if err != nil {
		t.Fatalf("encode Upload: %v", err)
	}
	protected, err := codec.Protect(context.Background(), nil, 1, "Upload", model.CommandOriginUser, encoded)
	if err != nil {
		t.Fatalf("protect: %v", err)
	}
	if strings.Contains(string(protected), "shared-log-user") || strings.Contains(string(protected), "shared-log-password") ||
		!strings.Contains(string(protected), logUploadUsernamePlaceholder) || !strings.Contains(string(protected), logUploadPasswordPlaceholder) {
		t.Fatalf("protected payload=%s", protected)
	}
	params, err := service.DecodeRPCParams("Upload", protected)
	if err != nil {
		t.Fatalf("decode protected params: %v", err)
	}
	storeLogUploadRuntime("rotated-user", "rotated-password")
	if err := codec.Hydrate(context.Background(), 1, "Upload", params); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if params["url"] != "http://gva:7458/acs/log" || params["username"] != "rotated-user" || params["password"] != "rotated-password" {
		t.Fatalf("hydrated params=%#v", params)
	}
}

func storeLogUploadRuntime(username, password string) {
	config.StoreRuntime(config.TR069Config{FileIngress: config.FileIngressConfig{
		Enabled: true, PublicBaseURL: "http://gva:7458",
		Authentication: config.FileIngressAuthConfig{Username: username, Password: password, Realm: "GVA", Schemes: []string{"basic"}},
		Channels:       map[string]config.TransferChannelConfig{"log": {Enabled: true, Path: "/acs/log"}},
	}})
}
