package response

import (
	"github.com/ddddddddwp/gva-acs/server/model/system/request"
)

type PolicyPathResponse struct {
	Paths []request.CasbinInfo `json:"paths"`
}
