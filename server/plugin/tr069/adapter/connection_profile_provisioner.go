package adapter

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	connectionCredentialQueueCapacity = 128
	connectionUsernameRandomBytes     = 12
	connectionPasswordRandomBytes     = 32
	connectionProvisionParameterKey   = "gva-connection-request"
)

var errProvisioningAlreadyInFlight = errors.New("connection credential provisioning is already in flight")

// ConnectionCredentialScheduler is the deliberately small, non-blocking API
// used by Inform and GPV collection paths.
type ConnectionCredentialScheduler interface {
	Schedule(deviceID uint) bool
}

type ConnectionCredentialProvisioner struct {
	queue      chan uint
	manager    *service.CommandManager
	repository *ConnectionProfileRepository
}

func NewConnectionCredentialProvisioner(manager *service.CommandManager, repository *ConnectionProfileRepository) *ConnectionCredentialProvisioner {
	return &ConnectionCredentialProvisioner{
		queue:      make(chan uint, connectionCredentialQueueCapacity),
		manager:    manager,
		repository: repository,
	}
}

// Schedule never waits for random generation, database access, Redis, or CPE
// wakeup. Database correlation in Ensure supplies duplicate safety.
func (p *ConnectionCredentialProvisioner) Schedule(deviceID uint) bool {
	if p == nil || p.queue == nil || deviceID == 0 {
		return false
	}
	select {
	case p.queue <- deviceID:
		return true
	default:
		return false
	}
}

// Run uses one worker. It first recovers DISCOVERED profiles and deliberately
// excludes FAILED profiles, which require an explicit Ensure or manual update.
func (p *ConnectionCredentialProvisioner) Run(ctx context.Context) {
	if p == nil || p.repository == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	deviceIDs, err := p.repository.ListProvisioningCandidates(ctx)
	if err == nil {
		for _, deviceID := range deviceIDs {
			if ctx.Err() != nil {
				return
			}
			_, _ = p.Ensure(ctx, deviceID)
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case deviceID := <-p.queue:
			_, _ = p.Ensure(ctx, deviceID)
		}
	}
}

func (p *ConnectionCredentialProvisioner) Ensure(ctx context.Context, deviceID uint) (service.SubmitResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if p == nil || p.manager == nil || p.repository == nil {
		return service.SubmitResult{}, errors.New("connection credential provisioner is not initialized")
	}
	if deviceID == 0 {
		return service.SubmitResult{}, errors.New("connection profile device ID is required")
	}
	settings := config.CurrentRuntime().Settings.ConnectionRequest
	if !settings.AutoProvisionCredentials {
		return service.SubmitResult{}, nil
	}
	if _, err := NewCredentialCipher(settings); err != nil {
		return service.SubmitResult{}, err
	}

	profile, err := p.repository.Get(ctx, deviceID)
	if err != nil {
		return service.SubmitResult{}, err
	}
	if profile.ProvisionState == model.ConnectionProfileStateProvisioning {
		return p.correlateExisting(ctx, profile)
	}
	if profile.ProvisionState == model.ConnectionProfileStateReady {
		return service.SubmitResult{}, nil
	}
	if profile.ProvisionState != model.ConnectionProfileStateDiscovered && profile.ProvisionState != model.ConnectionProfileStateFailed {
		return service.SubmitResult{}, fmt.Errorf("connection profile has unsupported provision state %q", profile.ProvisionState)
	}
	if effectiveConnectionURL(profile) == "" {
		return service.SubmitResult{}, errors.New("connection profile has no connection request URL")
	}

	usernameToken, err := randomConnectionCredentialToken(connectionUsernameRandomBytes)
	if err != nil {
		return service.SubmitResult{}, err
	}
	password, err := randomConnectionCredentialToken(connectionPasswordRandomBytes)
	if err != nil {
		return service.SubmitResult{}, err
	}
	username := "gva-" + usernameToken
	keyVersion := strings.TrimSpace(settings.CredentialKeyVersion)
	dedupKey := "connection-profile:" + strconv.FormatUint(uint64(deviceID), 10) + ":" + keyVersion
	request := req.SetParameterValuesRequest{
		ParameterKey: connectionProvisionParameterKey,
		Parameters: []req.SetParameterValue{
			{Name: connectionRequestUsernameName, Type: "xsd:string", Value: username},
			{Name: connectionRequestPasswordName, Type: "xsd:string", Value: password},
		},
	}

	result, err := p.manager.SubmitSystem(ctx, deviceID, "SetParameterValues", request, dedupKey,
		func(hookCtx context.Context, tx *gorm.DB, command *model.Command) error {
			return p.repository.LinkProvisioningCommand(hookCtx, tx, deviceID, username, keyVersion, command)
		},
	)
	if errors.Is(err, errProvisioningAlreadyInFlight) {
		current, loadErr := p.repository.Get(ctx, deviceID)
		if loadErr != nil {
			return service.SubmitResult{}, errors.Join(err, loadErr)
		}
		return p.correlateExisting(ctx, current)
	}
	if err != nil {
		return result, err
	}
	if result.Status == model.CommandStatusFailed || result.Status == model.CommandStatusTimeout {
		lastError := result.Status
		var command model.Command
		if loadErr := p.repository.database().WithContext(ctx).First(&command, "command_id = ?", result.CommandID).Error; loadErr == nil && strings.TrimSpace(command.FaultString) != "" {
			lastError = command.FaultString
		}
		if terminalErr := p.repository.MarkTerminal(ctx, nil, result.CommandID, model.ConnectionProfileStateFailed, lastError); terminalErr != nil {
			return result, terminalErr
		}
	}
	return result, nil
}

func (p *ConnectionCredentialProvisioner) correlateExisting(ctx context.Context, profile model.ConnectionProfile) (service.SubmitResult, error) {
	if strings.TrimSpace(profile.ProvisionCommandID) == "" {
		return service.SubmitResult{}, errors.New("provisioning profile has no correlated command")
	}
	var command model.Command
	if err := p.repository.database().WithContext(ctx).First(&command, "command_id = ?", profile.ProvisionCommandID).Error; err != nil {
		return service.SubmitResult{}, err
	}
	result := service.SubmitResult{CommandID: command.CommandID, Status: command.Status}
	switch command.Status {
	case model.CommandStatusCompleted:
		if err := p.repository.MarkTerminal(ctx, nil, command.CommandID, model.ConnectionProfileStateReady, ""); err != nil {
			return result, err
		}
	case model.CommandStatusFailed, model.CommandStatusTimeout:
		lastError := command.FaultString
		if strings.TrimSpace(lastError) == "" {
			lastError = command.Status
		}
		if err := p.repository.MarkTerminal(ctx, nil, command.CommandID, model.ConnectionProfileStateFailed, lastError); err != nil {
			return result, err
		}
	}
	return result, nil
}

func randomConnectionCredentialToken(size int) (string, error) {
	if size <= 0 {
		return "", errors.New("connection credential random size must be positive")
	}
	value := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, value); err != nil {
		return "", errors.New("generate connection request credential")
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func (r *ConnectionProfileRepository) ListProvisioningCandidates(ctx context.Context) ([]uint, error) {
	if r == nil || r.database() == nil {
		return nil, errors.New("connection profile repository database is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var deviceIDs []uint
	err := r.database().WithContext(ctx).Model(new(model.ConnectionProfile)).
		Where("provision_state = ?", model.ConnectionProfileStateDiscovered).
		Order("device_id ASC").
		Pluck("device_id", &deviceIDs).Error
	return deviceIDs, err
}

// LinkProvisioningCommand runs after the payload protector in SubmitSystem's
// transaction. The generated username/key checks prove that the password was
// moved to encrypted profile storage before a command row may commit.
func (r *ConnectionProfileRepository) LinkProvisioningCommand(ctx context.Context, tx *gorm.DB, deviceID uint, username, keyVersion string, command *model.Command) error {
	if r == nil || tx == nil || command == nil || command.CommandID == "" {
		return errors.New("connection profile provisioning correlation is not initialized")
	}
	var profile model.ConnectionProfile
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&profile, "device_id = ?", deviceID).Error; err != nil {
		return err
	}
	if profile.CredentialSource != model.ConnectionCredentialSourceAuto || profile.Username != username || profile.CredentialKeyVersion != keyVersion || len(profile.PasswordCiphertext) == 0 {
		return errors.New("connection profile automatic credential was not protected")
	}
	if profile.ProvisionCommandID != "" {
		var existing model.Command
		if err := tx.WithContext(ctx).First(&existing, "command_id = ?", profile.ProvisionCommandID).Error; err != nil {
			return err
		}
		if model.IsNonTerminalCommandStatus(existing.Status) {
			return errProvisioningAlreadyInFlight
		}
	}
	result := tx.WithContext(ctx).Model(new(model.ConnectionProfile)).
		Where("id = ? AND username = ? AND credential_source = ? AND credential_key_version = ?", profile.ID, username, model.ConnectionCredentialSourceAuto, keyVersion).
		Updates(map[string]any{
			"provision_state":      model.ConnectionProfileStateProvisioning,
			"provision_command_id": command.CommandID,
			"last_error":           "",
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("connection profile provisioning correlation conflict")
	}
	return nil
}

func (r *ConnectionProfileRepository) MarkTerminal(ctx context.Context, tx *gorm.DB, commandID, state, lastError string) error {
	if r == nil || strings.TrimSpace(commandID) == "" {
		return errors.New("connection profile terminal command ID is required")
	}
	if state != model.ConnectionProfileStateReady && state != model.ConnectionProfileStateFailed {
		return fmt.Errorf("unsupported connection profile terminal state %q", state)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db := tx
	if db == nil {
		db = r.database()
	}
	if db == nil {
		return errors.New("connection profile repository database is required")
	}
	if state == model.ConnectionProfileStateReady {
		lastError = ""
	}
	return db.WithContext(ctx).Model(new(model.ConnectionProfile)).
		Where("provision_command_id = ? AND provision_state = ?", commandID, model.ConnectionProfileStateProvisioning).
		Updates(map[string]any{
			"provision_state": state,
			"last_error":      lastError,
		}).Error
}

var _ ConnectionCredentialScheduler = (*ConnectionCredentialProvisioner)(nil)
