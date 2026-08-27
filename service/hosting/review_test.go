package hosting

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoReviewAllows(t *testing.T) {
	assert.False(t, AutoReviewAllows(constant.HostingAutoReviewOff, constant.HostingToolRiskLow))
	assert.True(t, AutoReviewAllows(constant.HostingAutoReviewConservative, constant.HostingToolRiskRead))
	assert.False(t, AutoReviewAllows(constant.HostingAutoReviewConservative, constant.HostingToolRiskLow))
	assert.True(t, AutoReviewAllows(constant.HostingAutoReviewBalanced, constant.HostingToolRiskLow))
	assert.False(t, AutoReviewAllows(constant.HostingAutoReviewBalanced, constant.HostingToolRiskMedium))
	assert.True(t, AutoReviewAllows(constant.HostingAutoReviewAggressive, constant.HostingToolRiskMedium))
	assert.False(t, AutoReviewAllows(constant.HostingAutoReviewAggressive, constant.HostingToolRiskHigh))
}

func TestToolRiskLevel(t *testing.T) {
	assert.Equal(t, constant.HostingToolRiskRead, ToolRiskLevel("list_channels", false))
	assert.Equal(t, constant.HostingToolRiskLow, ToolRiskLevel("test_channel", true))
	assert.Equal(t, constant.HostingToolRiskMedium, ToolRiskLevel("set_channel_status", true))
	assert.Equal(t, constant.HostingToolRiskHigh, ToolRiskLevel("add_user_quota", true))
}

func TestMutatingToolQueuesApprovalWhenAutoReviewOff(t *testing.T) {
	setupHostingDB(t)
	require.NoError(t, authz.Init(model.DB))
	setRuntime(constant.HostingStatusReady, "")
	t.Cleanup(func() { setRuntime(constant.HostingStatusDisabled, "") })

	agent := seedAgent(t, func(agent *model.HostingAgent) {
		agent.AutoReviewMode = constant.HostingAutoReviewOff
		agent.DryRun = false
		agent.AllowAgentHooks = true
	})
	require.NoError(t, authz.SetHostingAgentPermissions(agent.UserId, authz.HostingPresetPermissions(constant.HostingPresetOperate)))

	ctx := &ToolContext{
		Agent:       agent,
		UserID:      agent.UserId,
		Role:        common.RoleAdminUser,
		ActorUserID: 99,
		ActorRole:   common.RoleRootUser,
		Interactive: true,
		FromRunner:  true,
	}
	result, err := ExecuteTool(ctx, "fix_abilities", map[string]any{})
	require.NoError(t, err)
	payload, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, payload["pending_approval"])

	items, err := model.ListHostingApprovals(agent.Id, 0, constant.HostingApprovalPending, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "fix_abilities", items[0].ToolName)
}

func TestBalancedAutoReviewExecutesLowRiskTool(t *testing.T) {
	setupHostingDB(t)
	require.NoError(t, authz.Init(model.DB))
	setRuntime(constant.HostingStatusReady, "")
	t.Cleanup(func() { setRuntime(constant.HostingStatusDisabled, "") })

	agent := seedAgent(t, func(agent *model.HostingAgent) {
		agent.AutoReviewMode = constant.HostingAutoReviewBalanced
		agent.DryRun = false
	})
	require.NoError(t, authz.SetHostingAgentPermissions(agent.UserId, authz.HostingPresetPermissions(constant.HostingPresetOperate)))

	ctx := &ToolContext{
		Agent:       agent,
		UserID:      agent.UserId,
		Role:        common.RoleAdminUser,
		ActorUserID: agent.UserId,
		ActorRole:   common.RoleAdminUser,
		SkipReview:  false,
	}
	_, err := ExecuteTool(ctx, "fix_abilities", map[string]any{})
	require.NoError(t, err)
	items, err := model.ListHostingApprovals(agent.Id, 0, "", 10)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestUserQuotaRequestIsCapped(t *testing.T) {
	setupHostingDB(t)
	user := seedHandoffUser(t)
	agent := seedAgent(t, func(agent *model.HostingAgent) {
		agent.AutoReviewMode = constant.HostingAutoReviewOff
	})
	ctx := &ToolContext{
		Agent:       agent,
		UserID:      agent.UserId,
		ActorUserID: user.Id,
		ActorRole:   common.RoleCommonUser,
	}
	_, err := queueUserAccessRequest(ctx, map[string]any{
		"request_type": "quota",
		"quota":        constant.MaxHostingQuotaGrant + 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quota must be between")
}

func TestUserQuotaRequestCreatesPendingApproval(t *testing.T) {
	setupHostingDB(t)
	user := seedHandoffUser(t)
	agent := seedAgent(t, func(agent *model.HostingAgent) {
		agent.AutoReviewMode = constant.HostingAutoReviewOff
	})
	ctx := &ToolContext{
		Agent:       agent,
		UserID:      agent.UserId,
		ActorUserID: user.Id,
		ActorRole:   common.RoleCommonUser,
	}
	result, err := queueUserAccessRequest(ctx, map[string]any{
		"request_type": "quota",
		"quota":        1000,
		"note":         "need more calls",
	})
	require.NoError(t, err)
	payload, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, payload["pending_approval"])
}

func TestEnsureDefaultStewardRenamesOpsAgent(t *testing.T) {
	setupHostingDB(t)
	require.NoError(t, authz.Init(model.DB))
	created, err := CreateAgent(CreateAgentRequest{Name: "ops-agent", BrainModel: "gpt-test"})
	require.NoError(t, err)
	require.NoError(t, EnsureDefaultSteward())
	agent, err := model.GetDefaultHostingAgent()
	require.NoError(t, err)
	assert.Equal(t, created.Agent.Id, agent.Id)
	assert.True(t, agent.IsDefault)
	assert.Equal(t, constant.DefaultHostingStewardName, agent.Name)
	assert.True(t, authz.Can(agent.UserId, common.RoleAdminUser, authz.OptionWrite))
}

func TestRunChatTurnWithoutBrain(t *testing.T) {
	setupHostingDB(t)
	agent := seedAgent(t, func(agent *model.HostingAgent) {
		agent.BrainModel = ""
		agent.Enabled = true
	})
	result, err := RunChatTurn(agent, 1, common.RoleRootUser, "hello", "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.NeedsBrain)
}

func TestHostingPresetNeverIncludesSecretView(t *testing.T) {
	for _, preset := range []string{constant.HostingPresetWatch, constant.HostingPresetOperate, constant.HostingPresetFull} {
		perms := authz.HostingPresetPermissions(preset)
		if perms["channel"] != nil {
			assert.False(t, perms["channel"]["secret_view"], preset)
			assert.False(t, perms["channel"]["sensitive_write"], preset)
		}
		if preset == constant.HostingPresetWatch {
			assert.True(t, perms["option"]["read"])
			assert.False(t, perms["option"]["write"])
		} else {
			assert.True(t, perms["option"]["read"], preset)
			assert.True(t, perms["option"]["write"], preset)
		}
	}
}

func TestNonAdminChatNeverAutoExecutesMutatingTool(t *testing.T) {
	setupHostingDB(t)
	require.NoError(t, authz.Init(model.DB))
	setRuntime(constant.HostingStatusReady, "")
	t.Cleanup(func() { setRuntime(constant.HostingStatusDisabled, "") })

	agent := seedAgent(t, func(agent *model.HostingAgent) {
		agent.AutoReviewMode = constant.HostingAutoReviewAggressive
		agent.DryRun = false
	})
	require.NoError(t, authz.SetHostingAgentPermissions(agent.UserId, authz.HostingPresetPermissions(constant.HostingPresetOperate)))
	user := seedHandoffUser(t)

	ctx := &ToolContext{
		Agent:       agent,
		UserID:      agent.UserId,
		Role:        common.RoleAdminUser,
		ActorUserID: user.Id,
		ActorRole:   common.RoleCommonUser,
		Interactive: true,
		FromRunner:  true,
	}
	result, err := ExecuteTool(ctx, "fix_abilities", map[string]any{})
	require.NoError(t, err)
	payload, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, payload["pending_approval"])
}

func TestDecideApprovalRejectsNonPending(t *testing.T) {
	setupHostingDB(t)
	agent := seedAgent(t, nil)
	item := &model.HostingApproval{
		AgentId:         agent.Id,
		RequesterUserId: 1,
		Kind:            constant.HostingApprovalUserPermission,
		Status:          constant.HostingApprovalAutoApproved,
		Summary:         "already done",
	}
	require.NoError(t, item.Insert())
	_, err := DecideApproval(item.Id, 1, common.RoleRootUser, ApprovalDecision{Approve: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer pending")
}

func TestBriefingIsDue(t *testing.T) {
	assert.False(t, BriefingIsDue(constant.HostingBriefingOff, 0, 100, true))
	assert.True(t, BriefingIsDue(constant.HostingBriefingEveryOpen, 0, 100, false))
	assert.False(t, BriefingIsDue(constant.HostingBriefingEveryOpen, 50, 100, false))
	assert.True(t, BriefingIsDue(constant.HostingBriefingEveryOpen, 50, 50+constant.HostingBriefingEveryOpenSec, false))
	assert.False(t, BriefingIsDue(constant.HostingBriefingDaily, 10, 10+1000, false))
	assert.True(t, BriefingIsDue(constant.HostingBriefingDaily, 10, 10+constant.HostingBriefingDailySec, false))
}

func TestSensitiveOptionKeysAreHidden(t *testing.T) {
	assert.True(t, IsSensitiveOptionKey("SMTPToken"))
	assert.True(t, IsSensitiveOptionKey("StripeApiSecret"))
	assert.True(t, IsSensitiveOptionKey("GitHubClientSecret"))
	assert.False(t, IsSensitiveOptionKey("SystemName"))
	assert.Equal(t, "(hidden)", redactOptionValue("SMTPToken", "abc"))
	assert.Equal(t, "New API", redactOptionValue("SystemName", "New API"))
	assert.Equal(t, constant.HostingToolRiskHigh, SettingWriteRisk("StripeApiSecret"))
	assert.Equal(t, constant.HostingToolRiskLow, SettingWriteRisk("SystemName"))
}

func TestFallbackBriefingWithoutBrain(t *testing.T) {
	setupHostingDB(t)
	agent := seedAgent(t, func(agent *model.HostingAgent) {
		agent.BrainModel = ""
		agent.BriefingMode = constant.HostingBriefingEveryOpen
	})
	result, err := GenerateStewardBriefing(agent, 1, true)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Text)
	assert.False(t, result.Generated)
}

func TestValidateSettingWriteBounds(t *testing.T) {
	require.Error(t, validateSettingWrite("", "x"))
	require.NoError(t, validateSettingWrite("SystemName", "Acme API"))
	tooLong := strings.Repeat("a", constant.MaxHostingOptionValueRunes+1)
	require.Error(t, validateSettingWrite("Notice", tooLong))
}
