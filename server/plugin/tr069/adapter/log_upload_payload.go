package adapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"gorm.io/gorm"
)

const (
	logUploadUsernamePlaceholder = "__GVA_TR069_LOG_UPLOAD_USERNAME__"
	logUploadPasswordPlaceholder = "__GVA_TR069_LOG_UPLOAD_PASSWORD__"
)

type LogUploadPayloadCodec struct{}

func (LogUploadPayloadCodec) Protect(_ context.Context, _ *gorm.DB, _ uint, operation, _ string, encoded []byte) ([]byte, error) {
	if operation != "Upload" {
		return append([]byte(nil), encoded...), nil
	}
	decoded, err := service.DecodeRPCRequest(operation, encoded)
	if err != nil {
		return nil, err
	}
	upload, ok := decoded.(request.UploadRequest)
	if !ok {
		return nil, fmt.Errorf("unexpected Upload request type %T", decoded)
	}
	upload.Username = logUploadUsernamePlaceholder
	upload.Password = logUploadPasswordPlaceholder
	return service.EncodeRPCRequest(operation, upload)
}

func (LogUploadPayloadCodec) Hydrate(_ context.Context, _ uint, operation string, params map[string]interface{}) error {
	if operation != "Upload" {
		return nil
	}
	if params == nil {
		return errors.New("Upload params are required")
	}
	runtime := config.CurrentRuntime()
	settings := runtime.Settings.FileIngress
	channel, ok := settings.Channels["log"]
	if !settings.Enabled || !ok || !channel.Enabled || strings.TrimSpace(settings.PublicBaseURL) == "" ||
		settings.Authentication.Username == "" || settings.Authentication.Password == "" {
		return errors.New("LOG file ingress is not configured")
	}
	params["url"] = strings.TrimRight(settings.PublicBaseURL, "/") + channel.Path
	params["username"] = settings.Authentication.Username
	params["password"] = settings.Authentication.Password
	return nil
}

type CompositeCommandPayloadCodec struct {
	protectors []service.CommandPayloadProtector
	hydrators  []CommandPayloadHydrator
}

func NewCompositeCommandPayloadCodec(components ...any) *CompositeCommandPayloadCodec {
	codec := new(CompositeCommandPayloadCodec)
	for _, component := range components {
		if protector, ok := component.(service.CommandPayloadProtector); ok && protector != nil {
			codec.protectors = append(codec.protectors, protector)
		}
		if hydrator, ok := component.(CommandPayloadHydrator); ok && hydrator != nil {
			codec.hydrators = append(codec.hydrators, hydrator)
		}
	}
	return codec
}

func (c *CompositeCommandPayloadCodec) Protect(ctx context.Context, tx *gorm.DB, deviceID uint, operation, origin string, encoded []byte) ([]byte, error) {
	protected := append([]byte(nil), encoded...)
	if c == nil {
		return protected, nil
	}
	for _, protector := range c.protectors {
		var err error
		protected, err = protector.Protect(ctx, tx, deviceID, operation, origin, protected)
		if err != nil {
			return nil, err
		}
	}
	return protected, nil
}

func (c *CompositeCommandPayloadCodec) Hydrate(ctx context.Context, deviceID uint, operation string, params map[string]interface{}) error {
	if c == nil {
		return nil
	}
	for _, hydrator := range c.hydrators {
		if err := hydrator.Hydrate(ctx, deviceID, operation, params); err != nil {
			return err
		}
	}
	return nil
}

var _ service.CommandPayloadProtector = LogUploadPayloadCodec{}
var _ CommandPayloadHydrator = LogUploadPayloadCodec{}
var _ service.CommandPayloadProtector = (*CompositeCommandPayloadCodec)(nil)
var _ CommandPayloadHydrator = (*CompositeCommandPayloadCodec)(nil)
