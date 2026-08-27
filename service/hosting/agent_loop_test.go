package hosting

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunAgentTurnContinuesSameSessionAcrossWakes(t *testing.T) {
	setupHostingDB(t)
	agent := seedAgent(t, nil)
	sessionID := agent.SessionId

	var n atomic.Int32
	SetInternalBrainRelay(func(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error) {
		i := n.Add(1)
		return &BrainResponse{Content: fmt.Sprintf("ack-%d", i), PromptTokens: 2, OutputTokens: 2}, nil
	})
	t.Cleanup(func() { SetInternalBrainRelay(nil) })

	RunAgentTurn(agent, "first hook: channel 3 failed", 1, "channel.auto_disabled")
	require.Equal(t, sessionID, agent.SessionId)

	fresh, err := model.GetHostingAgentById(agent.Id)
	require.NoError(t, err)
	require.Equal(t, sessionID, fresh.SessionId)

	RunAgentTurn(fresh, "second hook: still failing", 1, "channel.auto_disabled")

	entries, err := model.ListHostingSessionEntries(agent.Id, sessionID, 0, 50)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(entries), 4)

	var userContents []string
	for _, entry := range entries {
		if entry.Role == "user" {
			userContents = append(userContents, entry.Content)
		}
	}
	require.GreaterOrEqual(t, len(userContents), 2)
	assert.Contains(t, userContents[0], "first hook")
	assert.Contains(t, strings.Join(userContents, "\n"), "second hook")
	assert.Equal(t, sessionID, fresh.SessionId)

	reloaded, err := model.GetHostingAgentById(agent.Id)
	require.NoError(t, err)
	assert.Equal(t, sessionID, reloaded.SessionId)
}

func TestRunAgentTurnDailyBudgetHandsOffWithoutBrain(t *testing.T) {
	setupHostingDB(t)
	agent := seedAgent(t, func(a *model.HostingAgent) {
		a.DailyTokenBudget = 10
	})
	require.NoError(t, model.DB.Create(&model.HostingBrainUsage{
		AgentId:      agent.Id,
		Day:          time.Now().Format("2006-01-02"),
		PromptTokens: 8,
		OutputTokens: 8,
		UpdatedAt:    common.GetTimestamp(),
	}).Error)

	called := false
	SetInternalBrainRelay(func(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error) {
		called = true
		return &BrainResponse{Content: "should-not-run"}, nil
	})
	t.Cleanup(func() { SetInternalBrainRelay(nil) })

	RunAgentTurn(agent, "budget check", 2, "quota.exhausted")
	assert.False(t, called)

	items, err := model.ListHostingIncidents(agent.Id, constant.HostingIncidentHandedOff, 10)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	assert.Contains(t, items[0].HandoffReason, "daily token budget")
}

func TestRunAgentTurnDedicatedBrainFailureMarksIncidentSource(t *testing.T) {
	setupHostingDB(t)
	agent := seedAgent(t, func(a *model.HostingAgent) {
		a.BrainSource = constant.HostingBrainDedicated
		a.DedicatedBaseURL = "http://127.0.0.1:1"
		a.DedicatedAPIKey = ""
	})
	SetInternalBrainRelay(func(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error) {
		t.Fatal("internal relay must not run for dedicated brain")
		return nil, nil
	})
	t.Cleanup(func() { SetInternalBrainRelay(nil) })

	RunAgentTurn(agent, "dedicated outage", 3, "host.resource_threshold")

	items, err := model.ListHostingIncidents(agent.Id, constant.HostingIncidentHandedOff, 10)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	assert.Equal(t, constant.HostingBrainDedicated, items[0].BrainSource)
	assert.Contains(t, items[0].HandoffReason, "dedicated")
}

func TestRunAgentTurnCompactsThenContinues(t *testing.T) {
	setupHostingDB(t)
	agent := seedAgent(t, func(a *model.HostingAgent) {
		a.ContextWindow = 80
		a.ReserveTokens = 10
		a.KeepRecentTokens = 8
	})

	var compactCalls, turnCalls atomic.Int32
	SetInternalBrainRelay(func(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error) {
		if req.PromptCacheOff {
			compactCalls.Add(1)
			assert.Equal(t, false, BrainCompletionPayload(req)["store"])
			return &BrainResponse{
				Content:      "Goal: recover channel\nProgress: compacted\nDecisions: continue\nNext Steps: probe\nCritical Context: ch-9",
				PromptTokens: 6,
				OutputTokens: 4,
			}, nil
		}
		turnCalls.Add(1)
		joined := ""
		for _, msg := range req.Messages {
			joined += msg.Content
		}
		assert.Contains(t, joined, "COMPACTION")
		return &BrainResponse{Content: "continued after compact", PromptTokens: 3, OutputTokens: 2}, nil
	})
	t.Cleanup(func() { SetInternalBrainRelay(nil) })

	RunAgentTurn(agent, strings.Repeat("channel.auto_disabled probe history ", 80), 4, "channel.auto_disabled")

	assert.Greater(t, compactCalls.Load(), int32(0))
	assert.Greater(t, turnCalls.Load(), int32(0))

	fresh, err := model.GetHostingAgentById(agent.Id)
	require.NoError(t, err)
	assert.NotEmpty(t, fresh.LastCompactSummary)
	assert.Contains(t, fresh.LastCompactSummary, "Goal")

	used := model.HostingUsageDayTotal(agent.Id, time.Now().Format("2006-01-02"))
	assert.Greater(t, used, 0)
}

func TestRunAgentTurnOverflowAfterCompactHandsOff(t *testing.T) {
	setupHostingDB(t)
	agent := seedAgent(t, func(a *model.HostingAgent) {
		a.ContextWindow = 80
		a.ReserveTokens = 10
		a.KeepRecentTokens = 8
	})

	SetInternalBrainRelay(func(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error) {
		return nil, fmt.Errorf("maximum context length exceeded: prompt too long")
	})
	t.Cleanup(func() { SetInternalBrainRelay(nil) })

	RunAgentTurn(agent, strings.Repeat("overflow payload ", 40), 5, "channel.auto_disabled")

	items, err := model.ListHostingIncidents(agent.Id, constant.HostingIncidentHandedOff, 10)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	assert.Contains(t, items[0].HandoffReason, "overflow")
}

func TestDispatchNotifyOnlyHandsOffWithoutBrain(t *testing.T) {
	setupHostingDB(t)
	agent := seedAgent(t, nil)
	called := false
	SetInternalBrainRelay(func(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error) {
		called = true
		return &BrainResponse{Content: "nope"}, nil
	})
	t.Cleanup(func() { SetInternalBrainRelay(nil) })

	dispatchHook(agent, &model.HostingHook{
		Id:       9,
		WakeMode: constant.HostingWakeNotifyOnly,
		Name:     "quota",
	}, service.HostingEvent{Name: constant.HostingEventQuota, Reason: "quota exhausted"}, "quota exhausted")
	assert.False(t, called)

	items, err := model.ListHostingIncidents(agent.Id, constant.HostingIncidentHandedOff, 10)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	assert.Contains(t, items[0].HandoffReason, "notify_only")
}

func TestCostSnapshotIncludesBudget(t *testing.T) {
	setupHostingDB(t)
	agent := seedAgent(t, func(a *model.HostingAgent) {
		a.DailyTokenBudget = 12345
		a.MaxWakesPerHour = 7
	})
	require.NoError(t, model.AddHostingBrainUsage(agent.Id, 10, 5, 2, 0))
	snap := CostSnapshot(agent.Id)
	assert.Equal(t, 15, snap["tokens_used"])
	assert.Equal(t, 2, snap["wakes"])
	assert.Equal(t, 12345, snap["daily_token_budget"])
	assert.Equal(t, 7, snap["max_wakes_per_hour"])
}
