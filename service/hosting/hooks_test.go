package hosting

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestWakeSlotCap(t *testing.T) {
	held := 0
	t.Cleanup(func() {
		for i := 0; i < held; i++ {
			releaseWakeSlot()
		}
	})
	for i := 0; i < constant.HostingMaxConcurrentWakes; i++ {
		require.True(t, tryAcquireWakeSlot())
		held++
	}
	assert.False(t, tryAcquireWakeSlot())
	for i := 0; i < constant.HostingMaxConcurrentWakes; i++ {
		releaseWakeSlot()
		held--
	}
	assert.True(t, tryAcquireWakeSlot())
	held++
}

func TestDebounceQueuesFollowupWhileRunning(t *testing.T) {
	agent := &model.HostingAgent{Id: 777, WakeMergeWindowSec: 60, MaxWakesPerHour: 10, DailyTokenBudget: 1_000_000}
	t.Cleanup(func() {
		wakeMu.Lock()
		delete(runningWake, agent.Id)
		delete(followups, agent.Id)
		delete(lastWakeAt, agent.Id)
		delete(hourlyWakes, agent.Id)
		wakeMu.Unlock()
	})
	wakeMu.Lock()
	runningWake[agent.Id] = true
	wakeMu.Unlock()
	dispatchHook(agent, &model.HostingHook{Id: 1, WakeMode: constant.HostingWakeAI}, service.HostingEvent{Name: "x"}, "later")
	wakeMu.Lock()
	queued := append([]string{}, followups[agent.Id]...)
	wakeMu.Unlock()
	assert.Contains(t, queued, "later")
}
