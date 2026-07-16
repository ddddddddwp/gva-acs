package response

import (
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
)

type CommandRecordSummary struct {
	CommandID       string     `json:"commandId"`
	DeviceID        uint       `json:"deviceId"`
	DeviceKey       string     `json:"deviceKey"`
	Operation       string     `json:"operation"`
	RetryOf         string     `json:"retryOf"`
	CommandKey      *string    `json:"commandKey"`
	Status          string     `json:"status"`
	RequestID       string     `json:"requestId"`
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
		CommandID: command.CommandID, DeviceID: command.DeviceID, DeviceKey: command.DeviceKey,
		Operation: command.Operation, RetryOf: command.RetryOf, CommandKey: command.CommandKey,
		Status: command.Status, RequestID: command.RequestID, PhaseDeadlineAt: command.PhaseDeadlineAt,
		QueuedAt: command.QueuedAt, WaitingAt: command.WaitingAt, BuildingAt: command.BuildingAt,
		SentAt: command.SentAt, FinishedAt: command.FinishedAt, FailureStage: command.FailureStage,
		FaultCode: command.FaultCode, FaultString: command.FaultString,
		CreatedAt: command.CreatedAt, UpdatedAt: command.UpdatedAt,
	}
}

type CommandXMLResponse struct {
	ID        uint64    `json:"id"`
	Direction string    `json:"direction"`
	Method    string    `json:"method"`
	CWMPID    string    `json:"cwmpId"`
	RequestID string    `json:"requestId"`
	XML       string    `json:"xml"`
	CreatedAt time.Time `json:"createdAt"`
}

type CommandRecordDetail struct {
	Command model.Command        `json:"command"`
	Events  []model.CommandEvent `json:"events"`
	XML     []CommandXMLResponse `json:"xml"`
}

func CommandRecordDetailFrom(command model.Command, events []model.CommandEvent, xmlRecords []model.CommandXML) CommandRecordDetail {
	xml := make([]CommandXMLResponse, 0, len(xmlRecords))
	for _, record := range xmlRecords {
		xml = append(xml, CommandXMLResponse{
			ID: record.ID, Direction: record.Direction, Method: record.Method,
			CWMPID: record.CWMPID, RequestID: record.RequestID,
			XML: string(record.Payload), CreatedAt: record.CreatedAt,
		})
	}
	return CommandRecordDetail{Command: command, Events: events, XML: xml}
}
