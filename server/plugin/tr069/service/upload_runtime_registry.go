package service

import (
	"context"
	"errors"
	"sync"
)

type UploadRuntimeRegistry struct {
	mu      sync.Mutex
	nextID  uint64
	blocked map[uint]struct{}
	active  map[uint]map[uint64]*UploadRuntimeHandle
}

type UploadRuntimeHandle struct {
	mu           sync.Mutex
	registry     *UploadRuntimeRegistry
	deviceID     uint
	id           uint64
	cancel       context.CancelFunc
	writer       ArtifactWriter
	cancelled    bool
	unregistered bool
}

func NewUploadRuntimeRegistry() *UploadRuntimeRegistry {
	return &UploadRuntimeRegistry{
		blocked: make(map[uint]struct{}),
		active:  make(map[uint]map[uint64]*UploadRuntimeHandle),
	}
}

func (r *UploadRuntimeRegistry) Register(deviceID uint, cancel context.CancelFunc) (*UploadRuntimeHandle, error) {
	if r == nil || deviceID == 0 || cancel == nil {
		return nil, errors.New("complete upload runtime registration is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, blocked := r.blocked[deviceID]; blocked {
		return nil, ErrDeviceDeleting
	}
	r.nextID++
	handle := &UploadRuntimeHandle{registry: r, deviceID: deviceID, id: r.nextID, cancel: cancel}
	if r.active[deviceID] == nil {
		r.active[deviceID] = make(map[uint64]*UploadRuntimeHandle)
	}
	r.active[deviceID][handle.id] = handle
	return handle, nil
}

func (r *UploadRuntimeRegistry) BlockAndCancel(ctx context.Context, deviceID uint) error {
	if r == nil || deviceID == 0 {
		return errors.New("upload runtime registry and device ID are required")
	}
	r.mu.Lock()
	r.blocked[deviceID] = struct{}{}
	handles := make([]*UploadRuntimeHandle, 0, len(r.active[deviceID]))
	for _, handle := range r.active[deviceID] {
		handles = append(handles, handle)
	}
	r.mu.Unlock()

	var result error
	for _, handle := range handles {
		result = errors.Join(result, handle.cancelAndAbort(ctx))
	}
	return result
}

func (r *UploadRuntimeRegistry) Release(deviceID uint) {
	if r == nil || deviceID == 0 {
		return
	}
	r.mu.Lock()
	delete(r.blocked, deviceID)
	if len(r.active[deviceID]) == 0 {
		delete(r.active, deviceID)
	}
	r.mu.Unlock()
}

func (h *UploadRuntimeHandle) AttachWriter(writer ArtifactWriter) error {
	if h == nil || writer == nil {
		return errors.New("upload runtime handle and writer are required")
	}
	h.mu.Lock()
	if h.cancelled || h.unregistered {
		h.mu.Unlock()
		return errors.Join(ErrDeviceDeleting, writer.Abort(context.Background()))
	}
	h.writer = writer
	h.mu.Unlock()
	return nil
}

func (h *UploadRuntimeHandle) Unregister() {
	if h == nil || h.registry == nil {
		return
	}
	h.mu.Lock()
	if h.unregistered {
		h.mu.Unlock()
		return
	}
	h.unregistered = true
	h.mu.Unlock()

	r := h.registry
	r.mu.Lock()
	if handles := r.active[h.deviceID]; handles != nil {
		delete(handles, h.id)
		if len(handles) == 0 {
			delete(r.active, h.deviceID)
		}
	}
	r.mu.Unlock()
}

func (h *UploadRuntimeHandle) cancelAndAbort(ctx context.Context) error {
	if h == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	h.mu.Lock()
	if h.cancelled {
		h.mu.Unlock()
		return nil
	}
	h.cancelled = true
	cancel := h.cancel
	writer := h.writer
	h.writer = nil
	h.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if writer != nil {
		return writer.Abort(ctx)
	}
	return nil
}
