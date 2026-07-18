package response

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
)

func TestCommandRecordDTOUsesCWMPIDWithoutRequestID(t *testing.T) {
	detail := CommandRecordDetailFrom(
		model.Command{CommandID: "cmd-1", CWMPID: "cwmp-command-1"},
		nil,
		[]model.CommandXML{{CommandID: "cmd-1", CWMPID: "cwmp-wire-1", Payload: []byte(`<Envelope/>`)}},
	)
	payload, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal command detail: %v", err)
	}
	encoded := string(payload)
	if strings.Contains(encoded, `"requestId"`) {
		t.Fatalf("legacy requestId leaked into DTO: %s", encoded)
	}
	if !strings.Contains(encoded, `"cwmpId":"cwmp-command-1"`) || !strings.Contains(encoded, `"cwmpId":"cwmp-wire-1"`) {
		t.Fatalf("CWMP IDs missing from DTO: %s", encoded)
	}
}
