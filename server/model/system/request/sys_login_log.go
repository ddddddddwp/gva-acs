package request

import (
	"github.com/ddddddddwp/gva-acs/server/model/common/request"
	"github.com/ddddddddwp/gva-acs/server/model/system"
)

type SysLoginLogSearch struct {
	system.SysLoginLog
	request.PageInfo
}
