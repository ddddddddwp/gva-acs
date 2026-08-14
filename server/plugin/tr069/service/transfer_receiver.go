package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrTransferFileTooLarge    = errors.New("transfer file exceeds configured limit")
	ErrTransferBusy            = errors.New("transfer ingress is busy")
	ErrTransferSizeMismatch    = errors.New("artifact store size mismatch")
	ErrTransferContentConflict = errors.New("active transfer retry content differs")
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
	lifecycle *TransferLifecycle
	runtime   *UploadRuntimeRegistry
	now       func() time.Time
	buffers   sync.Pool
}

func NewTransferReceiver(transfers *TransferStore, objects ArtifactStore, registries ...*UploadRuntimeRegistry) *TransferReceiver {
	var lifecycle *TransferLifecycle
	if transfers != nil {
		lifecycle = NewTransferLifecycle(transfers.db)
	}
	receiver := &TransferReceiver{transfers: transfers, objects: objects, admission: NewTransferAdmissionController(), lifecycle: lifecycle, now: time.Now}
	if len(registries) > 0 {
		receiver.runtime = registries[0]
	}
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
	var runtimeHandle *UploadRuntimeHandle
	if r.runtime != nil {
		runtimeCancel := func() {
			cancel()
			if closer, ok := request.Body.(io.Closer); ok {
				_ = closer.Close()
			}
		}
		var err error
		runtimeHandle, err = r.runtime.Register(request.Device.DeviceID, runtimeCancel)
		if err != nil {
			return model.Artifact{}, err
		}
		defer runtimeHandle.Unregister()
	}
	if existing, found, err := r.existingActiveArtifact(receiveCtx, request.Device.DeviceID, request.Channel); err != nil {
		return model.Artifact{}, err
	} else if found {
		return r.compareActiveRetry(receiveCtx, request, existing)
	}
	receivedAt := r.now().UTC()
	var deleteAt *time.Time
	if request.RetentionDays > 0 {
		value := receivedAt.Add(time.Duration(request.RetentionDays) * 24 * time.Hour)
		deleteAt = &value
	}
	metadata := ReceiveMetadata{
		StoragePrefix: request.StoragePrefix, Driver: request.Driver,
		OriginalName: SanitizeArtifactOriginalName(request.OriginalName), ContentType: request.ContentType,
		SourceIP: request.SourceIP, DeleteAt: deleteAt, CreatedAt: receivedAt,
	}

	task, artifact, err := r.createReceivingMetadata(receiveCtx, request, metadata)
	if err != nil {
		return model.Artifact{}, err
	}
	objectKey := artifact.ObjectKey
	w, err := r.objects.Begin(receiveCtx, ObjectSpec{Key: objectKey, ContentType: request.ContentType, Metadata: map[string]string{"file-id": strconv.FormatUint(artifact.ID, 10)}})
	if err != nil {
		r.failReceiving(receiveCtx, task, artifact, "storage.begin", "STORE_BEGIN_FAILED", err)
		return model.Artifact{}, err
	}
	if runtimeHandle != nil {
		if err := runtimeHandle.AttachWriter(w); err != nil {
			r.failReceiving(context.Background(), task, artifact, "runtime.register", "DEVICE_DELETING", err)
			return model.Artifact{}, err
		}
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
	artifact, err = r.transfers.MarkArtifactAvailable(receiveCtx, artifact.ID, artifact.Version, ArtifactFinalization{
		Size: written, SHA256: hex.EncodeToString(hasher.Sum(nil)), ReceivedAt: receivedAt,
	})
	if err != nil {
		return model.Artifact{}, err
	}
	if r.lifecycle == nil {
		return model.Artifact{}, errors.New("transfer lifecycle is required")
	}
	if err := r.lifecycle.OnArtifactAvailable(receiveCtx, task.TaskID, artifact.ID, receivedAt); err != nil {
		return model.Artifact{}, err
	}
	return artifact, nil
}

func (r *TransferReceiver) existingActiveArtifact(ctx context.Context, deviceID uint, channel string) (model.Artifact, bool, error) {
	task, err := r.transfers.FindUniqueWaitingActive(ctx, deviceID, channel)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Artifact{}, false, nil
	}
	if err != nil {
		return model.Artifact{}, false, err
	}
	if task.Status == model.TransferStatusWaitingFile {
		return model.Artifact{}, false, nil
	}
	artifact, err := r.transfers.GetTaskArtifact(ctx, task.TaskID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Artifact{}, false, ErrTransferBusy
	}
	if err != nil {
		return model.Artifact{}, false, err
	}
	if artifact.Status != model.ArtifactStatusAvailable {
		return model.Artifact{}, false, ErrTransferBusy
	}
	return artifact, true, nil
}

func (r *TransferReceiver) compareActiveRetry(ctx context.Context, request ReceiveRequest, existing model.Artifact) (model.Artifact, error) {
	hasher := sha256.New()
	limited := &io.LimitedReader{R: request.Body, N: request.MaxFileSize + 1}
	buffer := r.buffers.Get().([]byte)
	defer r.buffers.Put(buffer)
	written, err := io.CopyBuffer(hasher, limited, buffer)
	if err != nil {
		return model.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return model.Artifact{}, err
	}
	if written > request.MaxFileSize {
		return model.Artifact{}, ErrTransferFileTooLarge
	}
	digest := hex.EncodeToString(hasher.Sum(nil))
	if written != existing.Size || digest != existing.SHA256 {
		if err := r.transfers.AppendTransferEvent(ctx, model.TransferEvent{
			TaskID: existing.TaskID, FileID: existing.ID, Code: "DUPLICATE_CONTENT_CONFLICT",
			Phase: "ingress.idempotency", Message: "active upload retry content differs", CreatedAt: r.now().UTC(),
		}); err != nil {
			return model.Artifact{}, err
		}
		return model.Artifact{}, ErrTransferContentConflict
	}
	if err := r.transfers.AppendTransferEvent(ctx, model.TransferEvent{
		TaskID: existing.TaskID, FileID: existing.ID, Code: "DUPLICATE_ACCEPTED",
		Phase: "ingress.idempotency", Message: "active upload retry matched existing artifact", CreatedAt: r.now().UTC(),
	}); err != nil {
		return model.Artifact{}, err
	}
	return existing, nil
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
	_ = r.transfers.MarkArtifactFailed(ctx, artifact.ID, artifact.Version)
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
