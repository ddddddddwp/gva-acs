package core

import (
	"testing"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/utils"
	"github.com/spf13/viper"
)

func TestApplyConfigChangeTriggersHandlerAfterSuccessfulUnmarshal(t *testing.T) {
	previousEvents := utils.GlobalSystemEvents
	previousConfig := global.GVA_CONFIG
	t.Cleanup(func() {
		utils.GlobalSystemEvents = previousEvents
		global.GVA_CONFIG = previousConfig
	})

	events := &utils.SystemEvents{}
	utils.GlobalSystemEvents = events
	called := 0
	events.RegisterConfigChangeHandler(func() {
		called++
	})

	valid := viper.New()
	valid.Set("system.addr", 9999)
	if err := applyConfigChange(valid); err != nil {
		t.Fatalf("valid config change: %v", err)
	}
	if called != 1 {
		t.Fatalf("handler calls after valid change=%d", called)
	}

	invalid := viper.New()
	invalid.Set("system", "not-a-system-map")
	if err := applyConfigChange(invalid); err == nil {
		t.Fatal("invalid config change unexpectedly unmarshaled")
	}
	if called != 1 {
		t.Fatalf("handler calls after invalid change=%d", called)
	}
}
