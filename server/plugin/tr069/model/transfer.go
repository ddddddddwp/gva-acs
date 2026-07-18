package model

import "time"

const (
	TransferSourceActive   = "ACTIVE"
	TransferSourcePeriodic = "PERIODIC"

	TransferStatusWaitingFile     = "WAITING_FILE"
	TransferStatusReceiving       = "RECEIVING"
	TransferStatusWaitingTransfer = "WAITING_TRANSFER"
	TransferStatusCompleted       = "COMPLETED"
	TransferStatusFailed          = "FAILED"
	TransferStatusTimeout         = "TIMEOUT"

	ArtifactStatusReceiving = "RECEIVING"
	ArtifactStatusAvailable = "AVAILABLE"
	ArtifactStatusFailed    = "FAILED"
	ArtifactStatusDeleting  = "DELETING"
	ArtifactStatusDeleted   = "DELETED"
)

type TransferTask struct {
	TaskID               string     `json:"taskId" gorm:"primaryKey;size:64"`
	DeviceID             uint       `json:"deviceId" gorm:"index:idx_tr069_transfer_device_status,priority:1"`
	Channel              string     `json:"channel" gorm:"size:24;index"`
	Source               string     `json:"source" gorm:"size:16;index"`
	CommandID            *string    `json:"-" gorm:"size:64;uniqueIndex"`
	CommandKey           *string    `json:"-" gorm:"size:64;index"`
	Status               string     `json:"status" gorm:"size:32;index;index:idx_tr069_transfer_device_status,priority:2"`
	UploadResponseStatus *int       `json:"uploadResponseStatus"`
	FileReceivedAt       *time.Time `json:"fileReceivedAt"`
	TransferCompletedAt  *time.Time `json:"transferCompletedAt"`
	CompletedAt          *time.Time `json:"completedAt"`
	FailureStage         string     `json:"failureStage" gorm:"size:64"`
	FailureCode          string     `json:"failureCode" gorm:"size:64"`
	FailureMessage       string     `json:"failureMessage" gorm:"type:text"`
	PhaseDeadlineAt      *time.Time `json:"phaseDeadlineAt" gorm:"index"`
	Version              uint       `json:"version"`
	CreatedAt            time.Time  `json:"createdAt" gorm:"index"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

func (TransferTask) TableName() string { return "tr069_transfer_tasks" }

type Artifact struct {
	ArtifactID   string     `json:"artifactId" gorm:"primaryKey;size:64"`
	TaskID       string     `json:"taskId" gorm:"size:64;uniqueIndex"`
	DeviceID     uint       `json:"deviceId" gorm:"index"`
	Channel      string     `json:"channel" gorm:"size:24;index"`
	Status       string     `json:"status" gorm:"size:24;index"`
	Driver       string     `json:"driver" gorm:"size:24"`
	ObjectKey    string     `json:"-" gorm:"size:512;uniqueIndex"`
	OriginalName string     `json:"originalName" gorm:"size:255"`
	ContentType  string     `json:"contentType" gorm:"size:128"`
	Size         int64      `json:"size"`
	SHA256       string     `json:"sha256" gorm:"size:64;index"`
	SourceIP     string     `json:"sourceIp" gorm:"size:64"`
	ReceivedAt   *time.Time `json:"receivedAt"`
	DeleteAt     *time.Time `json:"deleteAt" gorm:"index"`
	DeletedAt    *time.Time `json:"deletedAt"`
	Version      uint       `json:"version"`
	CreatedAt    time.Time  `json:"createdAt" gorm:"index"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (Artifact) TableName() string { return "tr069_artifacts" }

type TransferEvent struct {
	ID           uint64       `json:"id" gorm:"primaryKey"`
	TaskID       string       `json:"taskId" gorm:"size:64;index"`
	ArtifactID   string       `json:"artifactId,omitempty" gorm:"size:64;index"`
	Code         string       `json:"code" gorm:"size:64;index"`
	Phase        string       `json:"phase" gorm:"size:64"`
	FromStatus   string       `json:"fromStatus" gorm:"size:32"`
	ToStatus     string       `json:"toStatus" gorm:"size:32"`
	Message      string       `json:"message" gorm:"type:text"`
	MetadataJSON LongTextJSON `json:"metadata" gorm:"type:longtext"`
	CreatedAt    time.Time    `json:"createdAt" gorm:"index"`
}

func (TransferEvent) TableName() string { return "tr069_transfer_events" }
