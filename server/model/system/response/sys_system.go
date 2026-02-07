package response

import "github.com/ddddddddwp/gva-acs/server/config"

type SysConfigResponse struct {
	Config config.Server `json:"config"`
}
