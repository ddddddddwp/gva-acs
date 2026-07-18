package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrTransferFileTooLarge = errors.New("transfer file exceeds configured limit")
	ErrTransferBusy         = errors.New("transfer ingress is busy")
	ErrTransferSizeMismatch = errors.New("artifact store size mismatch")
)

type UploadDeviceIdentity struct {
	DeviceID     uint      `json:"deviceId"`
	IP           string    `json:"ip"`
	OUI          string    `json:"oui"`
	ProductClass string    `json:"productClass"`
	SerialNumber string    `json:"serialNumber"`
	BoundAt      time.Time `json:"boundAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type ReceiveRequest struct {
	Device                 UploadDeviceIdentity
	Channel                string
	Body                   io.Reader
	ContentLength          int64
	OriginalName           string
	ContentType            string
	SourceIP               string
	Driver                 string
	StoragePrefix          string
	MaxFileSize            int64
	UploadTimeout          time.Duration
	RetentionDays          int
	MaxConcurrent          int
	MaxConcurrentPerDevice int
}

type TransferReceiver struct {
	transfers *TransferStore
	objects   ArtifactStore
	admission *TransferAdmissionController
	now       func() time.Time
	buffers   sync.Pool
}

func NewTransferReceiver(transfers *TransferStore, objects ArtifactStore) *TransferReceiver {
	receiver := &TransferReceiver{transfers: transfers, objects: objects, admission: NewTransferAdmissionController(), now: time.Now}
	receiver.buffers.New = func() any { return make([]byte, 64*1024) }
	return receiver
}

func (r *TransferReceiver) Receive(ctx context.Context, request ReceiveRequest) (model.Artifact, error) {
	if r == nil || r.transfers == nil || r.objects == nil || request.Body == nil || request.Device.DeviceID == 0 || request.Channel == "" {
		return model.Artifact{}, errors.New("complete transfer receive request is required")
	}
	if request.MaxFileSize <= 0 || request.UploadTimeout <= 0 || request.MaxConcurrent <= 0 || request.MaxConcurrentPerDevice <= 0 {
		return model.Artifact{}, errors.New("valid transfer receive limits are required")
	}
	if request.ContentLength > request.MaxFileSize {
		return model.Artifact{}, ErrTransferFileTooLarge
	}
	release, err := r.admission.Acquire(request.Device.DeviceID, request.Channel, request.MaxConcurrent, request.MaxConcurrentPerDevice)
	if err != nil {
		return model.Artifact{}, err
	}
	defer release()

	receiveCtx, cancel := context.WithTimeout(ctx, request.UploadTimeout)
	defer cancel()
	receivedAt := r.now().UTC()
	artifactID := uuid.NewString()
	objectKey, err := ArtifactObjectKey(request.StoragePrefix, request.Channel, request.Device.DeviceID, receivedAt, artifactID)
	if err != nil {
		return model.Artifact{}, err
	}
	var deleteAt *time.Time
	if request.RetentionDays > 0 {
		value := receivedAt.Add(time.Duration(request.RetentionDays) * 24 * time.Hour)
		deleteAt = &value
	}
	metadata := ReceiveMetadata{
		ArtifactID: artifactID, ObjectKey: objectKey, Driver: request.Driver,
		OriginalName: SanitizeArtifactOriginalName(request.OriginalName), ContentType: request.ContentType,
		SourceIP: request.SourceIP, DeleteAt: deleteAt, CreatedAt: receivedAt,
	}

	task, artifact, err := r.createReceivingMetadata(receiveCtx, request, metadata)
	if err != nil {
		return model.Artifact{}, err
	}
	w, err := r.objects.Begin(receiveCtx, ObjectSpec{Key: objectKey, ContentType: request.ContentType, Metadata: map[string]string{"artifact-id": artifact.ArtifactID}})
	if err != nil {
		r.failReceiving(receiveCtx, task, artifact, "storage.begin", "STORE_BEGIN_FAILED", err)
		return model.Artifact{}, err
	}

	hasher := sha256.New()
	limited := &io.LimitedReader{R: request.Body, N: request.MaxFileSize + 1}
	buffer := r.buffers.Get().([]byte)
	defer r.buffers.Put(buffer)
	written, copyErr := io.CopyBuffer(io.MultiWriter(w, hasher), limited, buffer)
	if copyErr != nil {
		_ = w.Abort(context.Background())
		r.failReceiving(context.Background(), task, artifact, "storage.write", "STORE_WRITE_FAILED", copyErr)
		return model.Artifact{}, copyErr
	}
	if written > request.MaxFileSize {
		_ = w.Abort(context.Background())
		r.failReceiving(context.Background(), task, artifact, "ingress.limit", "FILE_TOO_LARGE", ErrTransferFileTooLarge)
		return model.Artifact{}, ErrTransferFileTooLarge
	}
	stat, err := w.Commit(receiveCtx)
	if err != nil {
		_ = w.Abort(context.Background())
		r.failReceiving(context.Background(), task, artifact, "storage.commit", "STORE_COMMIT_FAILED", err)
		return model.Artifact{}, err
	}
	if stat.Size != written {
		_ = r.objects.Delete(context.Background(), objectKey)
		r.failReceiving(context.Background(), task, artifact, "storage.verify", "SIZE_MISMATCH", ErrTransferSizeMismatch)
		return model.Artifact{}, ErrTransferSizeMismatch
	}
	artifact, err = r.transfers.MarkArtifactAvailable(receiveCtx, artifact.ArtifactID, artifact.Version, ArtifactFinalization{
		Size: written, SHA256: hex.EncodeToString(hasher.Sum(nil)), ReceivedAt: receivedAt,
	})
	if err != nil {
		return model.Artifact{}, err
	}
	updates := map[string]any{"file_received_at": receivedAt}
	target := model.TransferStatusWaitingTransfer
	if task.Source == model.TransferSourcePeriodic {
		target = model.TransferStatusCompleted
		updates["completed_at"] = receivedAt
	}
	if _, err := r.transfers.TransitionTask(receiveCtx, TransferTransition{
		TaskID: task.TaskID, FromStatuses: []string{model.TransferStatusReceiving}, ToStatus: target,
		ExpectedVersion: task.Version, EventCode: "FILE_STORED", Phase: "storage.commit", Updates: updates,
	}); err != nil {
		return model.Artifact{}, err
	}
	return artifact, nil
}

func (r *TransferReceiver) createReceivingMetadata(ctx context.Context, request ReceiveRequest, metadata ReceiveMetadata) (model.TransferTask, model.Artifact, error) {
	active, err := r.transfers.FindUniqueWaitingActive(ctx, request.Device.DeviceID, request.Channel)
	if err == nil {
		return r.transfers.CreateActiveReceiving(ctx, active, metadata)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.TransferTask{}, model.Artifact{}, err
	}
	metadata.TaskID = uuid.NewString()
	return r.transfers.CreatePeriodicReceiving(ctx, request.Device.DeviceID, request.Channel, metadata)
}

func (r *TransferReceiver) failReceiving(ctx context.Context, task model.TransferTask, artifact model.Artifact, stage, code string, cause error) {
	_ = r.transfers.MarkArtifactFailed(ctx, artifact.ArtifactID, artifact.Version)
	_, _ = r.transfers.TransitionTask(ctx, TransferTransition{
		TaskID: task.TaskID, FromStatuses: []string{model.TransferStatusReceiving}, ToStatus: model.TransferStatusFailed,
		ExpectedVersion: task.Version, EventCode: code, Phase: stage, Message: fmt.Sprint(cause),
		Updates: map[string]any{"failure_stage": stage, "failure_code": code, "failure_message": fmt.Sprint(cause), "completed_at": time.Now().UTC()},
	})
}

type TransferAdmissionController struct {
	mu       sync.Mutex
	global   int
	channels map[string]int
	devices  map[uint]int
}

func NewTransferAdmissionController() *TransferAdmissionController {
	return &TransferAdmissionController{channels: make(map[string]int), devices: make(map[uint]int)}
}

func (a *TransferAdmissionController) Acquire(deviceID uint, channel string, maxConcurrent, maxPerDevice int) (func(), error) {
	if a == nil || deviceID == 0 || channel == "" || maxConcurrent <= 0 || maxPerDevice <= 0 {
		return nil, ErrTransferBusy
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.global >= maxConcurrent || a.channels[channel] >= maxConcurrent || a.devices[deviceID] >= maxPerDevice {
		return nil, ErrTransferBusy
	}
	a.global++
	a.channels[channel]++
	a.devices[deviceID]++
	var once sync.Once
	return func() {
		once.Do(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			a.global--
			a.channels[channel]--
			a.devices[deviceID]--
		})
	}, nil
}
