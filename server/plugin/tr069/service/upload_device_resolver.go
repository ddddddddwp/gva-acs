package service

import (
	"context"
	"errors"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/gorm"
)

var (
	ErrUploadDeviceNotFound  = errors.New("upload device not found")
	ErrUploadDeviceAmbiguous = errors.New("upload device identity is ambiguous")
)

type UploadIdentityResolver interface {
	Resolve(context.Context, string) ([]UploadDeviceIdentity, error)
}

type UploadDeviceResolver struct {
	db         *gorm.DB
	transfers  *TransferStore
	identities UploadIdentityResolver
	bindingTTL time.Duration
	now        func() time.Time
}

func NewUploadDeviceResolver(db *gorm.DB, transfers *TransferStore, identities UploadIdentityResolver, bindingTTL time.Duration) *UploadDeviceResolver {
	if bindingTTL <= 0 {
		bindingTTL = 30 * time.Minute
	}
	return &UploadDeviceResolver{db: db, transfers: transfers, identities: identities, bindingTTL: bindingTTL, now: time.Now}
}

func (r *UploadDeviceResolver) Resolve(ctx context.Context, sourceIP, channel string) (UploadDeviceIdentity, error) {
	if r == nil || r.db == nil || sourceIP == "" || channel == "" {
		return UploadDeviceIdentity{}, ErrUploadDeviceNotFound
	}
	if identity, found, err := r.resolveActive(ctx, sourceIP, channel); err != nil {
		return UploadDeviceIdentity{}, err
	} else if found {
		return identity, nil
	}
	if r.identities != nil {
		candidates, err := r.identities.Resolve(ctx, sourceIP)
		if err != nil {
			return UploadDeviceIdentity{}, err
		}
		valid, err := r.registeredCandidates(ctx, sourceIP, candidates)
		if err != nil {
			return UploadDeviceIdentity{}, err
		}
		switch len(valid) {
		case 1:
			return valid[0], nil
		case 0:
		default:
			return UploadDeviceIdentity{}, ErrUploadDeviceAmbiguous
		}
	}
	return r.resolveDatabaseFallback(ctx, sourceIP)
}

func (r *UploadDeviceResolver) resolveActive(ctx context.Context, sourceIP, channel string) (UploadDeviceIdentity, bool, error) {
	type activeRow struct {
		DeviceID     uint
		IP           string
		OUI          string
		ProductClass string
		SerialNumber string
	}
	var rows []activeRow
	err := r.db.WithContext(ctx).Table("tr069_transfer_tasks AS tasks").
		Select("devices.id AS device_id, devices.ip, devices.oui, devices.product_class, devices.serial_number").
		Joins("JOIN tr069_devices AS devices ON devices.id = tasks.device_id").
		Where("tasks.source = ? AND tasks.channel = ? AND tasks.status = ?", model.TransferSourceActive, channel, model.TransferStatusWaitingFile).
		Where("devices.ip = ? AND devices.deleted_at IS NULL AND devices.serial_number <> '' AND devices.oui <> ''", sourceIP).
		Order("tasks.created_at ASC").Limit(2).Scan(&rows).Error
	if err != nil {
		return UploadDeviceIdentity{}, false, err
	}
	if len(rows) > 1 {
		return UploadDeviceIdentity{}, false, ErrUploadDeviceAmbiguous
	}
	if len(rows) == 0 {
		return UploadDeviceIdentity{}, false, nil
	}
	row := rows[0]
	return UploadDeviceIdentity{DeviceID: row.DeviceID, IP: row.IP, OUI: row.OUI, ProductClass: row.ProductClass, SerialNumber: row.SerialNumber}, true, nil
}

func (r *UploadDeviceResolver) registeredCandidates(ctx context.Context, sourceIP string, candidates []UploadDeviceIdentity) ([]UploadDeviceIdentity, error) {
	valid := make([]UploadDeviceIdentity, 0, len(candidates))
	seen := make(map[uint]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate.DeviceID == 0 || candidate.IP != sourceIP {
			continue
		}
		if _, exists := seen[candidate.DeviceID]; exists {
			continue
		}
		var device model.Device
		err := r.db.WithContext(ctx).Where("id = ? AND serial_number <> '' AND oui <> ''", candidate.DeviceID).First(&device).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		seen[device.ID] = struct{}{}
		valid = append(valid, UploadDeviceIdentity{
			DeviceID: device.ID, IP: candidate.IP, OUI: device.OUI, ProductClass: device.ProductClass,
			SerialNumber: device.SerialNumber, BoundAt: candidate.BoundAt, ExpiresAt: candidate.ExpiresAt,
		})
	}
	return valid, nil
}

func (r *UploadDeviceResolver) resolveDatabaseFallback(ctx context.Context, sourceIP string) (UploadDeviceIdentity, error) {
	cutoff := r.now().UTC().Add(-r.bindingTTL)
	var devices []model.Device
	err := r.db.WithContext(ctx).
		Where("ip = ? AND last_inform >= ? AND serial_number <> '' AND oui <> ''", sourceIP, cutoff).
		Order("last_inform DESC").Order("id ASC").Limit(2).Find(&devices).Error
	if err != nil {
		return UploadDeviceIdentity{}, err
	}
	if len(devices) == 0 {
		return UploadDeviceIdentity{}, ErrUploadDeviceNotFound
	}
	if len(devices) > 1 {
		return UploadDeviceIdentity{}, ErrUploadDeviceAmbiguous
	}
	device := devices[0]
	return UploadDeviceIdentity{DeviceID: device.ID, IP: sourceIP, OUI: device.OUI, ProductClass: device.ProductClass, SerialNumber: device.SerialNumber}, nil
}
