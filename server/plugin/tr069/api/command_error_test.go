package api

import (
	"errors"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
)

func TestCommandFailureMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "offline", err: service.ErrDeviceOffline, want: "设备离线，无法下发任务"},
		{name: "queue unavailable", err: service.ErrCommandQueueUnavailable, want: "命令队列不可用"},
		{name: "other", err: errors.New("boom"), want: "任务下发失败: boom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commandFailureMessage(tt.err); got != tt.want {
				t.Fatalf("commandFailureMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
