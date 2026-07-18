package response

import "time"

type ArtifactSummary struct {
	ArtifactID   string     `json:"artifactId"`
	TaskID       string     `json:"taskId"`
	DeviceID     uint       `json:"deviceId"`
	SerialNumber string     `json:"serialNumber"`
	OUI          string     `json:"oui"`
	Channel      string     `json:"channel"`
	Source       string     `json:"source"`
	Status       string     `json:"status"`
	OriginalName string     `json:"originalName"`
	ContentType  string     `json:"contentType"`
	Size         int64      `json:"size"`
	SHA256       string     `json:"sha256"`
	ReceivedAt   *time.Time `json:"receivedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
}
