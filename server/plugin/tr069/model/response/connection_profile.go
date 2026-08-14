package response

import "time"

type ConnectionProfileResponse struct {
	DeviceID           uint       `json:"deviceId"`
	EffectiveURL       string     `json:"effectiveUrl"`
	DiscoveredURL      string     `json:"discoveredUrl"`
	OverrideURL        string     `json:"overrideUrl"`
	Username           string     `json:"username"`
	CredentialSource   string     `json:"credentialSource"`
	AuthScheme         string     `json:"authScheme"`
	ProvisionState     string     `json:"provisionState"`
	ProvisionCommandID string     `json:"provisionCommandId"`
	LastError          string     `json:"lastError"`
	LastWakeAt         *time.Time `json:"lastWakeAt"`
	LastWakeStatus     string     `json:"lastWakeStatus"`
}
