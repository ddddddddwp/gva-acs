package request

type ConnectionProfileOverrideRequest struct {
	OverrideURL      string `json:"overrideUrl"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	ClearOverride    bool   `json:"clearOverride"`
	ClearCredentials bool   `json:"clearCredentials"`
}
