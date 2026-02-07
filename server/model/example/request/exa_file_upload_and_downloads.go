package request

import (
	"github.com/ddddddddwp/gva-acs/server/model/common/request"
)

type ExaAttachmentCategorySearch struct {
	ClassId int `json:"classId" form:"classId"`
	request.PageInfo
}
