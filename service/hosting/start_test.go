package hosting

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestStartDisabledByEnv(t *testing.T) {
	t.Setenv("HOSTING_ENABLED", "false")
	t.Cleanup(func() { setRuntime(constant.HostingStatusDisabled, "") })
	Start()
	assert.Equal(t, constant.HostingStatusDisabled, GetPublicStatus())
	assert.False(t, IsReady())
}

func TestStartDisabledByPanel(t *testing.T) {
	t.Setenv("HOSTING_ENABLED", "true")
	prev := common.HostingEnabled
	common.HostingEnabled = false
	t.Cleanup(func() {
		common.HostingEnabled = prev
		setRuntime(constant.HostingStatusDisabled, "")
	})
	Start()
	assert.Equal(t, constant.HostingStatusDisabled, GetPublicStatus())
	assert.False(t, IsReady())
}

func TestStartFailsOpenWithoutDatabase(t *testing.T) {
	t.Setenv("HOSTING_ENABLED", "true")
	t.Cleanup(func() { setRuntime(constant.HostingStatusDisabled, "") })
	Start()
	assert.Equal(t, constant.HostingStatusError, GetPublicStatus())
	assert.False(t, IsReady())
	rt := GetRuntime()
	assert.Contains(t, rt.Error, "database")
}

func TestPublicStatusOmitsError(t *testing.T) {
	setRuntime(constant.HostingStatusError, "migrate boom")
	t.Cleanup(func() { setRuntime(constant.HostingStatusDisabled, "") })
	assert.Equal(t, constant.HostingStatusError, GetPublicStatus())
	rt := GetRuntime()
	assert.Equal(t, "migrate boom", rt.Error)
}

func TestHookEvalHandlerUsesSystemTask(t *testing.T) {
	h := hookEvalHandler{}
	assert.Equal(t, model.SystemTaskTypeHostingHooks, h.Type())
	assert.Equal(t, hookEvalInterval, h.Interval())
	setRuntime(constant.HostingStatusDisabled, "")
	assert.False(t, h.Enabled())
	setRuntime(constant.HostingStatusReady, "")
	t.Cleanup(func() { setRuntime(constant.HostingStatusDisabled, "") })
	assert.True(t, h.Enabled())
}

func TestApplyPanelSwitchKeepsEnvKillSwitch(t *testing.T) {
	t.Setenv("HOSTING_ENABLED", "false")
	prev := common.HostingEnabled
	common.HostingEnabled = true
	t.Cleanup(func() {
		common.HostingEnabled = prev
		setRuntime(constant.HostingStatusDisabled, "")
	})
	ApplyPanelSwitch()
	assert.Equal(t, constant.HostingStatusDisabled, GetPublicStatus())
	assert.False(t, IsReady())
}
