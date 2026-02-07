package response

import "github.com/ddddddddwp/gva-acs/server/model/example"

type ExaFileResponse struct {
	File example.ExaFileUploadAndDownload `json:"file"`
}
