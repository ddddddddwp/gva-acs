package response

import (
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/redact"
)

type CommandRecordSummary struct {
	CommandID       string     `json:"commandId"`
	DeviceID        uint       `json:"deviceId"`
	DeviceKey       string     `json:"deviceKey"`
	Operation       string     `json:"operation"`
	RetryOf         string     `json:"retryOf"`
	CommandKey      *string    `json:"commandKey"`
	Status          string     `json:"status"`
	CWMPID          string     `json:"cwmpId"`
	PhaseDeadlineAt *time.Time `json:"phaseDeadlineAt"`
	QueuedAt        time.Time  `json:"queuedAt"`
	WaitingAt       *time.Time `json:"waitingAt"`
	BuildingAt      *time.Time `json:"buildingAt"`
	SentAt          *time.Time `json:"sentAt"`
	FinishedAt      *time.Time `json:"finishedAt"`
	FailureStage    string     `json:"failureStage"`
	FaultCode       int        `json:"faultCode"`
	FaultString     string     `json:"faultString"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func NewCommandRecordSummary(command model.Command) CommandRecordSummary {
	return CommandRecordSummary{
		CommandID: redact.CommandText(command.CommandID), DeviceID: command.DeviceID, DeviceKey: redact.CommandText(command.DeviceKey),
		Operation: redact.CommandText(command.Operation), RetryOf: redact.CommandText(command.RetryOf), CommandKey: sanitizedCommandKey(command.CommandKey),
		Status: redact.CommandText(command.Status), CWMPID: redact.CommandText(command.CWMPID), PhaseDeadlineAt: command.PhaseDeadlineAt,
		QueuedAt: command.QueuedAt, WaitingAt: command.WaitingAt, BuildingAt: command.BuildingAt,
		SentAt: command.SentAt, FinishedAt: command.FinishedAt, FailureStage: redact.CommandText(command.FailureStage),
		FaultCode: command.FaultCode, FaultString: redact.CommandText(command.FaultString),
		CreatedAt: command.CreatedAt, UpdatedAt: command.UpdatedAt,
	}
}

type CommandXMLResponse struct {
	ID        uint64    `json:"id"`
	Direction string    `json:"direction"`
	Method    string    `json:"method"`
	CWMPID    string    `json:"cwmpId"`
	XML       string    `json:"xml"`
	CreatedAt time.Time `json:"createdAt"`
}

type CommandRecordDetail struct {
	Command model.Command        `json:"command"`
	Events  []model.CommandEvent `json:"events"`
	XML     []CommandXMLResponse `json:"xml"`
}

func CommandRecordDetailFrom(command model.Command, events []model.CommandEvent, xmlRecords []model.CommandXML) CommandRecordDetail {
	sanitizeCommandStrings(&command)
	command.ParamsJSON = sanitizedCommandJSON(command.ParamsJSON)
	command.ResultJSON = sanitizedCommandJSON(command.ResultJSON)
	sanitizedEvents := append([]model.CommandEvent(nil), events...)
	for index := range sanitizedEvents {
		sanitizeCommandEventStrings(&sanitizedEvents[index])
		sanitizedEvents[index].PayloadJSON = sanitizedCommandJSON(sanitizedEvents[index].PayloadJSON)
	}
	xml := make([]CommandXMLResponse, 0, len(xmlRecords))
	for _, record := range xmlRecords {
		sanitized, err := redact.CWMPXML(append([]byte(nil), record.Payload...))
		if err != nil {
			sanitized = nil
		}
		xml = append(xml, CommandXMLResponse{
			ID: record.ID, Direction: redact.CommandText(record.Direction), Method: redact.CommandText(record.Method),
			CWMPID: redact.CommandText(record.CWMPID),
			XML:    string(sanitized), CreatedAt: record.CreatedAt,
		})
	}
	return CommandRecordDetail{Command: command, Events: sanitizedEvents, XML: xml}
}

func sanitizeCommandStrings(command *model.Command) {
	command.CommandID = redact.CommandText(command.CommandID)
	command.DeviceKey = redact.CommandText(command.DeviceKey)
	command.Operation = redact.CommandText(command.Operation)
	command.Origin = redact.CommandText(command.Origin)
	command.RetryOf = redact.CommandText(command.RetryOf)
	command.DedupKey = redact.CommandText(command.DedupKey)
	command.CommandKey = sanitizedCommandKey(command.CommandKey)
	command.Status = redact.CommandText(command.Status)
	command.CWMPID = redact.CommandText(command.CWMPID)
	command.FailureStage = redact.CommandText(command.FailureStage)
	command.FaultString = redact.CommandText(command.FaultString)
}

func sanitizeCommandEventStrings(event *model.CommandEvent) {
	event.CommandID = redact.CommandText(event.CommandID)
	event.EventType = redact.CommandText(event.EventType)
	event.FromStatus = redact.CommandText(event.FromStatus)
	event.ToStatus = redact.CommandText(event.ToStatus)
	event.Stage = redact.CommandText(event.Stage)
	event.Message = redact.CommandText(event.Message)
}

func sanitizedCommandKey(value *string) *string {
	if value == nil {
		return nil
	}
	sanitized := redact.CommandText(*value)
	return &sanitized
}

func sanitizedCommandJSON(value model.LongTextJSON) model.LongTextJSON {
	if len(value) == 0 {
		return nil
	}
	sanitized, err := redact.CommandJSON(append([]byte(nil), value...))
	if err != nil {
		return nil
	}
	return model.LongTextJSON(sanitized)
}
