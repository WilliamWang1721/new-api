package hosting

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtectBrainChannelBlocksPinnedInternalChannel(t *testing.T) {
	ctx := &ToolContext{Agent: &model.HostingAgent{
		BrainSource:    constant.HostingBrainInternal,
		BrainChannelId: 7,
	}}
	require.Error(t, protectBrainChannel(ctx, "set_channel_status", map[string]any{"id": 7}))
	require.Error(t, protectBrainChannel(ctx, "update_channel_endpoint", map[string]any{"id": 7}))
	require.Error(t, protectBrainChannel(ctx, "manage_multi_keys", map[string]any{"id": 7, "action": "disable_key"}))
	require.NoError(t, protectBrainChannel(ctx, "manage_multi_keys", map[string]any{"id": 7, "action": "get_key_status"}))
	require.NoError(t, protectBrainChannel(ctx, "set_channel_status", map[string]any{"id": 8}))
	require.NoError(t, protectBrainChannel(ctx, "list_channels", map[string]any{"id": 7}))
}

func TestProtectBrainChannelIgnoresDedicatedSource(t *testing.T) {
	ctx := &ToolContext{Agent: &model.HostingAgent{
		BrainSource:    constant.HostingBrainDedicated,
		BrainChannelId: 7,
	}}
	require.NoError(t, protectBrainChannel(ctx, "set_channel_status", map[string]any{"id": 7}))
}

func TestSanitizeToolResultStripsSecrets(t *testing.T) {
	out := sanitizeToolResult(map[string]any{
		"id":      1,
		"key":     "sk-secret",
		"api_key": "another",
		"name":    "ok",
	})
	m, ok := out.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ok", m["name"])
	assert.NotContains(t, m, "key")
	assert.NotContains(t, m, "api_key")
}

func TestToolRateLimitRejectsBurst(t *testing.T) {
	limiter := &ToolRateLimit{window: map[int][]time.Time{}}
	for i := 0; i < maxToolCallsPerMinute; i++ {
		require.True(t, limiter.Allow(99))
	}
	assert.False(t, limiter.Allow(99))
	assert.True(t, limiter.Allow(100))
}
