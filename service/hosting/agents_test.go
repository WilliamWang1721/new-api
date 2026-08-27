package hosting

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicAgentOmitsDedicatedKey(t *testing.T) {
	setupHostingDB(t)
	enc, err := common.EncryptSecret("sk-dedicated-never-leak")
	require.NoError(t, err)
	view := PublicAgent(&model.HostingAgent{
		Id:               3,
		Name:             "ops",
		BrainSource:      constant.HostingBrainDedicated,
		DedicatedAPIKey:  enc,
		DedicatedBaseURL: "https://dedicated.example/v1",
	})
	require.NotNil(t, view)
	assert.True(t, view.DedicatedKeySet)
	assert.Empty(t, view.DedicatedAPIKey)
	assert.NotContains(t, view.DedicatedAPIKey, "sk-dedicated")
}

func TestCreateDedicatedAgentDoesNotInsertChannel(t *testing.T) {
	setupHostingDB(t)
	encKey := "sk-dedicated-isolated"
	result, err := CreateAgent(CreateAgentRequest{
		Name:             "dedicated-ops",
		BrainSource:      constant.HostingBrainDedicated,
		BrainModel:       "gpt-ops",
		DedicatedBaseURL: "https://dedicated.example/v1",
		DedicatedAPIKey:  encKey,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Agent)
	assert.Equal(t, constant.HostingBrainDedicated, result.Agent.BrainSource)
	assert.True(t, result.Agent.DedicatedKeySet)
	assert.Empty(t, result.Agent.DedicatedAPIKey)

	var channels int64
	require.NoError(t, model.DB.Model(&model.Channel{}).Count(&channels).Error)
	assert.Equal(t, int64(0), channels)

	var leaked int64
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("base_url LIKE ?", "%dedicated.example%").Count(&leaked).Error)
	assert.Equal(t, int64(0), leaked)

	stored, err := model.GetHostingAgentById(result.Agent.Id)
	require.NoError(t, err)
	assert.NotEmpty(t, stored.DedicatedAPIKey)
	assert.NotEqual(t, encKey, stored.DedicatedAPIKey)
	plain, err := common.DecryptSecret(stored.DedicatedAPIKey)
	require.NoError(t, err)
	assert.Equal(t, encKey, plain)
}

func TestUpdateAgentSwitchToInternalClearsDedicatedKey(t *testing.T) {
	setupHostingDB(t)
	created, err := CreateAgent(CreateAgentRequest{
		Name:             "switch-ops",
		BrainSource:      constant.HostingBrainDedicated,
		BrainModel:       "gpt-ops",
		DedicatedBaseURL: "https://dedicated.example/v1",
		DedicatedAPIKey:  "sk-keep-secret",
	})
	require.NoError(t, err)

	view, err := UpdateAgent(created.Agent.Id, CreateAgentRequest{
		BrainSource: constant.HostingBrainInternal,
		BrainModel:  "gpt-internal",
	})
	require.NoError(t, err)
	assert.Equal(t, constant.HostingBrainInternal, view.BrainSource)
	assert.False(t, view.DedicatedKeySet)
	assert.Empty(t, view.DedicatedAPIKey)

	stored, err := model.GetHostingAgentById(created.Agent.Id)
	require.NoError(t, err)
	assert.Empty(t, stored.DedicatedAPIKey)
	assert.Empty(t, stored.DedicatedBaseURL)
}

func TestAgentAccountCannotLogin(t *testing.T) {
	setupHostingDB(t)
	hash, err := common.Password2Hash("correct-horse")
	require.NoError(t, err)
	user := &model.User{
		Username:    "ha-nologin",
		Password:    hash,
		Status:      common.UserStatusEnabled,
		AccountKind: constant.AccountKindAgent,
		Role:        common.RoleAdminUser,
		AffCode:     "nologin-aff",
	}
	require.NoError(t, model.DB.Create(user).Error)

	check := model.User{Username: "ha-nologin", Password: "correct-horse"}
	err = check.ValidateAndFill()
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrInvalidCredentials)
}

func TestCreateTwoAgentsSucceedsOnSQLite(t *testing.T) {
	setupHostingDB(t)
	first, err := CreateAgent(CreateAgentRequest{Name: "agent-one", BrainModel: "gpt-test"})
	require.NoError(t, err)
	second, err := CreateAgent(CreateAgentRequest{Name: "agent-two", BrainModel: "gpt-test"})
	require.NoError(t, err)
	assert.NotEqual(t, first.Agent.Id, second.Agent.Id)
	assert.NotEqual(t, first.Agent.UserId, second.Agent.UserId)
}

func TestIssueAndAuthenticateHostingToken(t *testing.T) {
	setupHostingDB(t)
	setRuntime(constant.HostingStatusReady, "")
	t.Cleanup(func() { setRuntime(constant.HostingStatusDisabled, "") })
	agent := seedAgent(t, nil)

	issued, err := IssueAgentToken(agent.Id, "ops", "")
	require.NoError(t, err)
	require.NotNil(t, issued)
	assert.True(t, len(issued.Secret) > 4)
	assert.Equal(t, constant.HostingTokenPrefix, issued.Secret[:4])
	assert.NotEqual(t, issued.Secret, issued.Token.TokenHash)

	got, token, err := AuthenticateAgentToken("Bearer "+issued.Secret, "1.2.3.4")
	require.NoError(t, err)
	assert.Equal(t, agent.Id, got.Id)
	assert.Equal(t, issued.Token.Id, token.Id)

	ctx := ToolContextFromAgent(got, token, "1.2.3.4")
	result, err := ExecuteTool(ctx, "get_system_status", nil)
	require.NoError(t, err)
	status, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, constant.HostingStatusReady, status["hosting"])
}
