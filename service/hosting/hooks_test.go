package hosting

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestAllowHourlyWakeRespectsBudget(t *testing.T) {
	agent := &model.HostingAgent{Id: 4242, MaxWakesPerHour: 2}
	t.Cleanup(func() {
		wakeMu.Lock()
		delete(hourlyWakes, agent.Id)
		wakeMu.Unlock()
	})
	now := time.Now()
	assert.True(t, allowHourlyWake(agent, now))
	assert.True(t, allowHourlyWake(agent, now))
	assert.False(t, allowHourlyWake(agent, now))
}
