package adapter

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	connectionRequestURLName      = "Device.ManagementServer.ConnectionRequestURL"
	connectionRequestUsernameName = "Device.ManagementServer.ConnectionRequestUsername"
	connectionRequestPasswordName = "Device.ManagementServer.ConnectionRequestPassword"
)

var ErrConnectionProfileNotReady = errors.New("connection profile is not ready")

type CollectResult struct {
	Profile           model.ConnectionProfile
	NeedsProvisioning bool
}

type ResolvedConnectionProfile struct {
	DeviceID         uint
	URL              string
	Username         string
	Password         string
	AuthScheme       string
	CredentialSource string
}

type ConnectionProfileOverride struct {
	OverrideURL      string
	Username         string
	Password         string
	ClearOverride    bool
	ClearCredentials bool
}

type ConnectionProfileRepository struct {
	db     *gorm.DB
	cipher CredentialCipher
}

func NewConnectionProfileRepository(db *gorm.DB, credentialCipher CredentialCipher) *ConnectionProfileRepository {
	return &ConnectionProfileRepository{db: db, cipher: credentialCipher}
}

func (r *ConnectionProfileRepository) database() *gorm.DB {
	if r != nil && r.db != nil {
		return r.db
	}
	return global.GVA_DB
}

func (r *ConnectionProfileRepository) Collect(ctx context.Context, deviceID uint, values map[string]string) (CollectResult, error) {
	if r == nil || r.database() == nil {
		return CollectResult{}, errors.New("connection profile repository database is required")
	}
	if deviceID == 0 {
		return CollectResult{}, errors.New("connection profile device ID is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	urlValue := strings.TrimSpace(values[connectionRequestURLName])
	usernameValue := strings.TrimSpace(values[connectionRequestUsernameName])
	passwordValue := values[connectionRequestPasswordName]

	var result CollectResult
	err := r.database().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		profile, err := loadOrCreateConnectionProfile(ctx, tx, deviceID, urlValue != "" || usernameValue != "" || passwordValue != "")
		if err != nil {
			return err
		}
		if profile.ID == 0 {
			result = CollectResult{}
			return nil
		}

		updates := map[string]any{}
		if urlValue != "" && urlValue != profile.DiscoveredURL {
			updates["discovered_url"] = urlValue
			profile.DiscoveredURL = urlValue
		}
		if profile.CredentialSource != model.ConnectionCredentialSourceManual {
			if usernameValue != "" && usernameValue != profile.Username {
				updates["username"] = usernameValue
				profile.Username = usernameValue
			}
			if passwordValue != "" {
				if r.cipher == nil {
					return errors.New("connection profile credential cipher is required")
				}
				encrypted, err := r.cipher.Encrypt(passwordValue)
				if err != nil {
					return err
				}
				updates["password_ciphertext"] = encrypted.Ciphertext
				updates["credential_key_version"] = encrypted.Version
				updates["credential_source"] = model.ConnectionCredentialSourceAuto
				profile.PasswordCiphertext = encrypted.Ciphertext
				profile.CredentialKeyVersion = encrypted.Version
				profile.CredentialSource = model.ConnectionCredentialSourceAuto
			}
			if profile.Username != "" && len(profile.PasswordCiphertext) > 0 {
				updates["provision_state"] = model.ConnectionProfileStateReady
				updates["last_error"] = ""
				profile.ProvisionState = model.ConnectionProfileStateReady
			}
		}
		if len(updates) > 0 {
			if err := tx.WithContext(ctx).Model(&model.ConnectionProfile{}).
				Where("id = ?", profile.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		if err := tx.WithContext(ctx).First(&profile, profile.ID).Error; err != nil {
			return err
		}
		result = CollectResult{
			Profile: profile,
			NeedsProvisioning: effectiveConnectionURL(profile) != "" &&
				profile.ProvisionState == model.ConnectionProfileStateDiscovered &&
				(profile.Username == "" || len(profile.PasswordCiphertext) == 0),
		}
		return nil
	})
	return result, err
}

func loadOrCreateConnectionProfile(ctx context.Context, tx *gorm.DB, deviceID uint, create bool) (model.ConnectionProfile, error) {
	var profile model.ConnectionProfile
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&profile, "device_id = ?", deviceID).Error
	if err == nil {
		return profile, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) || !create {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ConnectionProfile{}, nil
		}
		return model.ConnectionProfile{}, err
	}
	profile = model.ConnectionProfile{
		DeviceID:       deviceID,
		AuthScheme:     "digest",
		ProvisionState: model.ConnectionProfileStateDiscovered,
	}
	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&profile).Error; err != nil {
		return model.ConnectionProfile{}, err
	}
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&profile, "device_id = ?", deviceID).Error; err != nil {
		return model.ConnectionProfile{}, err
	}
	return profile, nil
}

func (r *ConnectionProfileRepository) StoreCredential(ctx context.Context, tx *gorm.DB, deviceID uint, username, password, source string) error {
	if r == nil || r.cipher == nil {
		return errors.New("connection profile credential cipher is required")
	}
	username = strings.TrimSpace(username)
	if deviceID == 0 || username == "" || password == "" {
		return errors.New("device ID, username and password are required")
	}
	if source != model.ConnectionCredentialSourceAuto && source != model.ConnectionCredentialSourceManual {
		return fmt.Errorf("unsupported connection credential source %q", source)
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
	encrypted, err := r.cipher.Encrypt(password)
	if err != nil {
		return err
	}
	profile, err := loadOrCreateConnectionProfile(ctx, db, deviceID, true)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.ConnectionProfile{}).Where("id = ?", profile.ID).Updates(map[string]any{
		"username":               username,
		"password_ciphertext":    encrypted.Ciphertext,
		"credential_key_version": encrypted.Version,
		"credential_source":      source,
		"provision_state":        model.ConnectionProfileStateReady,
		"last_error":             "",
	}).Error
}

func (r *ConnectionProfileRepository) Resolve(ctx context.Context, deviceID uint) (ResolvedConnectionProfile, error) {
	if r == nil || r.database() == nil || r.cipher == nil {
		return ResolvedConnectionProfile{}, errors.New("connection profile repository is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var profile model.ConnectionProfile
	if err := r.database().WithContext(ctx).First(&profile, "device_id = ?", deviceID).Error; err != nil {
		return ResolvedConnectionProfile{}, err
	}
	if profile.ProvisionState != model.ConnectionProfileStateReady {
		return ResolvedConnectionProfile{}, ErrConnectionProfileNotReady
	}
	urlValue := effectiveConnectionURL(profile)
	if urlValue == "" || profile.Username == "" || len(profile.PasswordCiphertext) == 0 {
		return ResolvedConnectionProfile{}, errors.New("connection profile is incomplete")
	}
	password, err := r.cipher.Decrypt(profile.CredentialKeyVersion, profile.PasswordCiphertext)
	if err != nil {
		return ResolvedConnectionProfile{}, err
	}
	return ResolvedConnectionProfile{
		DeviceID:         deviceID,
		URL:              urlValue,
		Username:         profile.Username,
		Password:         password,
		AuthScheme:       profile.AuthScheme,
		CredentialSource: profile.CredentialSource,
	}, nil
}

func (r *ConnectionProfileRepository) LoadCredential(ctx context.Context, deviceID uint) (string, string, error) {
	if r == nil || r.database() == nil || r.cipher == nil {
		return "", "", errors.New("connection profile repository is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var profile model.ConnectionProfile
	if err := r.database().WithContext(ctx).First(&profile, "device_id = ?", deviceID).Error; err != nil {
		return "", "", err
	}
	if profile.Username == "" || len(profile.PasswordCiphertext) == 0 {
		return "", "", errors.New("connection profile credential is incomplete")
	}
	password, err := r.cipher.Decrypt(profile.CredentialKeyVersion, profile.PasswordCiphertext)
	if err != nil {
		return "", "", err
	}
	return profile.Username, password, nil
}

func (r *ConnectionProfileRepository) Get(ctx context.Context, deviceID uint) (model.ConnectionProfile, error) {
	if r == nil || r.database() == nil {
		return model.ConnectionProfile{}, errors.New("connection profile repository database is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var profile model.ConnectionProfile
	err := r.database().WithContext(ctx).First(&profile, "device_id = ?", deviceID).Error
	return profile, err
}

func (r *ConnectionProfileRepository) UpdateOverride(ctx context.Context, deviceID uint, input ConnectionProfileOverride) (model.ConnectionProfile, error) {
	if r == nil || r.database() == nil {
		return model.ConnectionProfile{}, errors.New("connection profile repository database is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	input.OverrideURL = strings.TrimSpace(input.OverrideURL)
	input.Username = strings.TrimSpace(input.Username)
	if input.ClearOverride && input.OverrideURL != "" {
		return model.ConnectionProfile{}, errors.New("override URL cannot be set and cleared together")
	}
	if input.ClearCredentials && (input.Username != "" || input.Password != "") {
		return model.ConnectionProfile{}, errors.New("credentials cannot be set and cleared together")
	}
	if (input.Username == "") != (input.Password == "") {
		return model.ConnectionProfile{}, errors.New("username and password must be provided together")
	}
	if input.OverrideURL != "" {
		normalized, err := normalizeConnectionProfileURL(input.OverrideURL)
		if err != nil {
			return model.ConnectionProfile{}, err
		}
		input.OverrideURL = normalized
	}

	var result model.ConnectionProfile
	err := r.database().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		profile, err := loadOrCreateConnectionProfile(ctx, tx, deviceID, true)
		if err != nil {
			return err
		}
		updates := map[string]any{}
		if input.ClearOverride {
			updates["override_url"] = ""
		} else if input.OverrideURL != "" {
			updates["override_url"] = input.OverrideURL
		}
		if input.ClearCredentials {
			updates["username"] = ""
			updates["password_ciphertext"] = []byte(nil)
			updates["credential_key_version"] = ""
			updates["credential_source"] = ""
			updates["provision_state"] = model.ConnectionProfileStateDiscovered
			updates["provision_command_id"] = ""
			updates["last_error"] = ""
		}
		if len(updates) > 0 {
			if err := tx.WithContext(ctx).Model(&model.ConnectionProfile{}).Where("id = ?", profile.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		if input.Username != "" {
			if err := r.StoreCredential(ctx, tx, deviceID, input.Username, input.Password, model.ConnectionCredentialSourceManual); err != nil {
				return err
			}
		}
		return tx.WithContext(ctx).First(&result, "device_id = ?", deviceID).Error
	})
	return result, err
}

func normalizeConnectionProfileURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		return "", errors.New("connection request URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("connection request URL must use http or https")
	}
	if parsed.Host == "" {
		return "", errors.New("connection request URL host is required")
	}
	if parsed.User != nil {
		return "", errors.New("connection request URL must not contain credentials")
	}
	parsed.Fragment = ""
	return parsed.String(), nil
}

func effectiveConnectionURL(profile model.ConnectionProfile) string {
	if value := strings.TrimSpace(profile.OverrideURL); value != "" {
		return value
	}
	return strings.TrimSpace(profile.DiscoveredURL)
}
