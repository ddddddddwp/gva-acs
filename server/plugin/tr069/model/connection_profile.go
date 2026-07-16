package model

import "time"

const (
	ConnectionProfileStateDiscovered   = "DISCOVERED"
	ConnectionProfileStateProvisioning = "PROVISIONING"
	ConnectionProfileStateReady        = "READY"
	ConnectionProfileStateFailed       = "FAILED"

	ConnectionCredentialSourceAuto   = "AUTO"
	ConnectionCredentialSourceManual = "MANUAL"
)

// ConnectionProfile stores the ACS-side state required to wake one CPE.
// Secret material is never exposed through JSON.
type ConnectionProfile struct {
	ID                   uint       `json:"id" gorm:"primaryKey"`
	DeviceID             uint       `json:"deviceId" gorm:"uniqueIndex;not null"`
	DiscoveredURL        string     `json:"discoveredUrl" gorm:"size:2048"`
	OverrideURL          string     `json:"overrideUrl" gorm:"size:2048"`
	Username             string     `json:"username" gorm:"size:256"`
	PasswordCiphertext   []byte     `json:"-" gorm:"type:longblob"`
	CredentialKeyVersion string     `json:"-" gorm:"size:32"`
	CredentialSource     string     `json:"credentialSource" gorm:"size:16"`
	AuthScheme           string     `json:"authScheme" gorm:"size:16"`
	ProvisionState       string     `json:"provisionState" gorm:"size:24;index"`
	ProvisionCommandID   string     `json:"provisionCommandId" gorm:"size:64;index"`
	LastError            string     `json:"lastError" gorm:"type:text"`
	LastWakeAt           *time.Time `json:"lastWakeAt"`
	LastWakeStatus       string     `json:"lastWakeStatus" gorm:"size:32"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

func (ConnectionProfile) TableName() string {
	return "tr069_connection_request_profiles"
}
