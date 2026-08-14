package response

import "time"

type ArtifactSummary struct {
	FileID       uint64     `json:"fileId"`
	SerialNumber string     `json:"serialNumber"`
	Source       string     `json:"source"`
	OriginalName string     `json:"originalName"`
	Size         int64      `json:"size"`
	ReceivedAt   *time.Time `json:"receivedAt"`
}
