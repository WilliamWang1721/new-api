package hosting

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestStartDisabledByEnv(t *testing.T) {
	t.Setenv("HOSTING_ENABLED", "false")
	t.Cleanup(func() { setRuntime(constant.HostingStatusDisabled, "") })
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
