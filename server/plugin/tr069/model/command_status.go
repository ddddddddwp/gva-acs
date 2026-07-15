package model

const (
	CommandStatusQueued          = "QUEUED"
	CommandStatusWaitingDevice   = "WAITING_DEVICE"
	CommandStatusBuilding        = "BUILDING"
	CommandStatusSent            = "SENT"
	CommandStatusWaitingTransfer = "WAITING_TRANSFER"
	CommandStatusCompleted       = "COMPLETED"
	CommandStatusFailed          = "FAILED"
	CommandStatusTimeout         = "TIMEOUT"

	CommandEventCreated = "CREATED"
)

var commandStatusTransitions = map[string]map[string]struct{}{
	CommandStatusQueued: {
		CommandStatusWaitingDevice: {},
		CommandStatusFailed:        {},
	},
	CommandStatusWaitingDevice: {
		CommandStatusBuilding: {},
		CommandStatusFailed:   {},
		CommandStatusTimeout:  {},
	},
	CommandStatusBuilding: {
		CommandStatusWaitingDevice: {},
		CommandStatusSent:          {},
		CommandStatusFailed:        {},
	},
	CommandStatusSent: {
		CommandStatusWaitingTransfer: {},
		CommandStatusCompleted:       {},
		CommandStatusFailed:          {},
		CommandStatusTimeout:         {},
	},
	CommandStatusWaitingTransfer: {
		CommandStatusCompleted: {},
		CommandStatusFailed:    {},
		CommandStatusTimeout:   {},
	},
	CommandStatusCompleted: {},
	CommandStatusFailed:    {},
	CommandStatusTimeout:   {},
}

var nonTerminalCommandStatuses = []string{
	CommandStatusQueued,
	CommandStatusWaitingDevice,
	CommandStatusBuilding,
	CommandStatusSent,
	CommandStatusWaitingTransfer,
}

func IsCommandStatus(status string) bool {
	_, ok := commandStatusTransitions[status]
	return ok
}

func IsTerminalCommandStatus(status string) bool {
	switch status {
	case CommandStatusCompleted, CommandStatusFailed, CommandStatusTimeout:
		return true
	default:
		return false
	}
}

func IsNonTerminalCommandStatus(status string) bool {
	return IsCommandStatus(status) && !IsTerminalCommandStatus(status)
}

func CanTransitionCommand(from, to string) bool {
	allowed, ok := commandStatusTransitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

func NonTerminalCommandStatuses() []string {
	return append([]string(nil), nonTerminalCommandStatuses...)
}
