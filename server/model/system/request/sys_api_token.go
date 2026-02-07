package request

import (
	"github.com/ddddddddwp/gva-acs/server/model/common/request"
	"github.com/ddddddddwp/gva-acs/server/model/system"
)

type SysApiTokenSearch struct {
	system.SysApiToken
	request.PageInfo
    Status *bool `json:"status" form:"status"`
}
