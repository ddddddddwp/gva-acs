package adapter

import (
	"context"
	"errors"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/ddddddddwp/tr069-core-only/observability"
	"go.uber.org/zap"
)

// CommandXMLSink stores core wire events through CommandStore so XML is
// sanitized before persistence.
type CommandXMLSink struct {
	store     *service.CommandStore
	retention func() time.Duration
}

func NewCommandXMLSink(store *service.CommandStore, retention func() time.Duration) *CommandXMLSink {
	if retention == nil {
		retention = func() time.Duration { return config.CurrentRuntime().RPCXMLRetention }
	}
	return &CommandXMLSink{store: store, retention: retention}
}

func (s *CommandXMLSink) Enabled(level observability.Level) bool {
	return level == observability.LevelInfo
}

func (s *CommandXMLSink) Emit(ctx context.Context, event observability.Event) {
	if s == nil || s.store == nil || event.Stage != "wire.xml" || event.CommandID == "" || len(event.Payload) == 0 {
		return
	}
	now := time.Now()
	retention := time.Duration(0)
	if s.retention != nil {
		retention = s.retention()
	}
	err := s.store.SaveXML(ctx, &model.CommandXML{
		CommandID: event.CommandID,
		Direction: string(event.Direction),
		Method:    event.Method,
		CWMPID:    event.CWMPID,
		RequestID: event.RequestID,
		Payload:   append([]byte(nil), event.Payload...),
		ExpiresAt: now.Add(retention),
		CreatedAt: now,
	})
	if err != nil && global.GVA_LOG != nil {
		global.GVA_LOG.Error("TR069 command XML persistence failed",
			zap.String("stage", event.Stage),
			zap.String("commandId", event.CommandID),
			zap.String("reason", commandXMLSinkErrorReason(err)),
		)
	}
}

func commandXMLSinkErrorReason(err error) string {
	if errors.Is(err, service.ErrInvalidCommandXML) {
		return "invalid_xml"
	}
	return "storage"
}

var _ observability.EventSink = (*CommandXMLSink)(nil)
