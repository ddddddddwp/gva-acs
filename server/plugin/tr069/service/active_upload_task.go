package service

import (
	"context"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func NewActiveUploadTaskHook(taskID func() string) CommandCreatedHook {
	if taskID == nil {
		taskID = uuid.NewString
	}
	return func(ctx context.Context, tx *gorm.DB, command *model.Command) error {
		if command == nil || command.Operation != "Upload" {
			return nil
		}
		return NewTransferStore(tx).CreateActiveTask(ctx, command, taskID())
	}
}
