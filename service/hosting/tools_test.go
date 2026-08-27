package hosting

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
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

func TestExecuteToolUnauthorizedHandsOffFromRunner(t *testing.T) {
	setRuntime(constant.HostingStatusReady, "")
	t.Cleanup(func() { setRuntime(constant.HostingStatusDisabled, "") })
	ctx := &ToolContext{
		Agent:      &model.HostingAgent{Id: 1, UserId: 1},
		FromRunner: true,
	}
	_, err := ExecuteTool(ctx, "list_channels", nil)
	require.Error(t, err)
	assert.True(t, IsToolHandoff(err))
}

func TestExecuteToolDryRunHandsOffFromRunner(t *testing.T) {
	setRuntime(constant.HostingStatusReady, "")
	t.Cleanup(func() { setRuntime(constant.HostingStatusDisabled, "") })
	ctx := &ToolContext{
		Agent:      &model.HostingAgent{Id: 2, UserId: 2, DryRun: true, AllowAgentHooks: true},
		FromRunner: true,
	}
	_, err := ExecuteTool(ctx, "create_hook", map[string]any{"name": "later", "kind": "schedule"})
	require.Error(t, err)
	assert.True(t, IsToolHandoff(err))
	assert.Contains(t, err.Error(), "dry-run")
}

func TestIncidentRateLimit(t *testing.T) {
	limiter := &ToolRateLimit{window: map[int][]time.Time{}, inc: map[string][]time.Time{}}
	for i := 0; i < constant.HostingMaxToolCallsPerIncidentMin; i++ {
		require.True(t, limiter.AllowIncident("a:1"))
	}
	assert.False(t, limiter.AllowIncident("a:1"))
	assert.True(t, limiter.AllowIncident("b:1"))
}

func TestAlwaysAllowToolsVisibleWithoutGrants(t *testing.T) {
	ctx := &ToolContext{Agent: &model.HostingAgent{Id: 1, UserId: 1}}
	listed := ListToolsForAgent(ctx)
	names := make([]string, 0, len(listed))
	for _, item := range listed {
		name, _ := item["name"].(string)
		names = append(names, name)
	}
	assert.Contains(t, names, "handoff_incident")
	assert.Contains(t, names, "sleep_until_hook")
	assert.Contains(t, names, "list_hooks")
	assert.NotContains(t, names, "list_channels")
	assert.NotContains(t, names, "secret_view")
}

func TestCatalogIncludesAdminOps(t *testing.T) {
	names := map[string]bool{}
	for _, tool := range allTools() {
		names[tool.Name] = true
	}
	for _, want := range []string{
		"list_tokens", "get_token", "create_token", "update_token",
		"update_user_status", "add_user_quota", "update_group_ratio",
		"create_redemption", "update_redemption_status",
		"list_subscription_plans", "list_user_subscriptions", "set_subscription_plan_enabled",
		"list_vendors", "get_vendor", "create_vendor",
		"list_model_meta", "get_model_meta", "list_system_tasks",
	} {
		assert.True(t, names[want], want)
	}
}

func TestPublicTokenOmitsSecret(t *testing.T) {
	key := "sk-super-secret-value"
	out := publicToken(&model.Token{Id: 9, UserId: 3, Name: "ops", Key: key, Status: 1})
	assert.Equal(t, 9, out["id"])
	assert.Equal(t, "ops", out["name"])
	assert.NotContains(t, out, "key")
	assert.NotEqual(t, key, out["key_prefix"])
}

func TestRejectProtectedUser(t *testing.T) {
	require.Error(t, rejectProtectedUser(&model.User{Id: 1, Role: common.RoleRootUser}, 2))
	require.Error(t, rejectProtectedUser(&model.User{Id: 5, Role: common.RoleAdminUser}, 5))
	require.Error(t, rejectProtectedUser(&model.User{Id: 8, Role: common.RoleAdminUser, AccountKind: constant.AccountKindAgent}, 2))
	require.NoError(t, rejectProtectedUser(&model.User{Id: 8, Role: common.RoleCommonUser}, 2))
}
