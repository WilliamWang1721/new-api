package hosting

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"gorm.io/gorm"
)

type AgentView struct {
	model.HostingAgent
	DedicatedKeySet bool                 `json:"dedicated_key_set"`
	Permissions     authz.PermissionsMap `json:"permissions"`
	TokenPrefixes   []string             `json:"token_prefixes"`
}

type CreateAgentRequest struct {
	Name                  string               `json:"name"`
	Enabled               *bool                `json:"enabled"`
	HandoffUserId         int                  `json:"handoff_user_id"`
	BrainSource           string               `json:"brain_source"`
	BrainModel            string               `json:"brain_model"`
	BrainGroup            string               `json:"brain_group"`
	BrainChannelId        int                  `json:"brain_channel_id"`
	DedicatedBaseURL      string               `json:"dedicated_base_url"`
	DedicatedAPIKey       string               `json:"dedicated_api_key"`
	DedicatedAPIType      string               `json:"dedicated_api_type"`
	DedicatedHeaders      string               `json:"dedicated_headers"`
	DedicatedTimeoutSec   int                  `json:"dedicated_timeout_sec"`
	DefaultChannelGroups  string               `json:"default_channel_groups"`
	MaxActionsPerIncident int                  `json:"max_actions_per_incident"`
	DryRun                bool                 `json:"dry_run"`
	DailyTokenBudget      int                  `json:"daily_token_budget"`
	WakeMergeWindowSec    int                  `json:"wake_merge_window_sec"`
	MaxWakesPerHour       int                  `json:"max_wakes_per_hour"`
	AllowAgentHooks       *bool                `json:"allow_agent_hooks"`
	MaxAgentHooks         int                  `json:"max_agent_hooks"`
	MinHookIntervalSec    int                  `json:"min_hook_interval_sec"`
	ContextWindow         int                  `json:"context_window"`
	ReserveTokens         int                  `json:"reserve_tokens"`
	KeepRecentTokens      int                  `json:"keep_recent_tokens"`
	Remark                string               `json:"remark"`
	ApplyRecommendedPerms bool                 `json:"apply_recommended_permissions"`
	Permissions           authz.PermissionsMap `json:"permissions"`
	IssueToken            bool                 `json:"issue_token"`
	TokenName             string               `json:"token_name"`
	TokenAllowIPs         string               `json:"token_allow_ips"`
}

type CreateAgentResult struct {
	Agent  *AgentView `json:"agent"`
	Secret string     `json:"token,omitempty"`
}

func PublicAgent(agent *model.HostingAgent) *AgentView {
	if agent == nil {
		return nil
	}
	view := &AgentView{HostingAgent: *agent}
	view.DedicatedAPIKey = ""
	view.DedicatedKeySet = agent.DedicatedAPIKey != ""
	view.Permissions = authz.Capabilities(agent.UserId, common.RoleAdminUser)
	if tokens, err := model.ListHostingTokens(agent.Id); err == nil {
		for _, token := range tokens {
			if token.Status == common.UserStatusEnabled {
				view.TokenPrefixes = append(view.TokenPrefixes, token.TokenPrefix)
			}
		}
	}
	return view
}

func CreateAgent(req CreateAgentRequest) (*CreateAgentResult, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	agent := &model.HostingAgent{
		Name:                  req.Name,
		Enabled:               true,
		HandoffUserId:         req.HandoffUserId,
		BrainSource:           req.BrainSource,
		BrainModel:            req.BrainModel,
		BrainGroup:            req.BrainGroup,
		BrainChannelId:        req.BrainChannelId,
		DedicatedBaseURL:      strings.TrimSpace(req.DedicatedBaseURL),
		DedicatedAPIType:      req.DedicatedAPIType,
		DedicatedHeaders:      req.DedicatedHeaders,
		DedicatedTimeoutSec:   req.DedicatedTimeoutSec,
		DefaultChannelGroups:  req.DefaultChannelGroups,
		MaxActionsPerIncident: req.MaxActionsPerIncident,
		DryRun:                req.DryRun,
		DailyTokenBudget:      req.DailyTokenBudget,
		WakeMergeWindowSec:    req.WakeMergeWindowSec,
		MaxWakesPerHour:       req.MaxWakesPerHour,
		AllowAgentHooks:       true,
		MaxAgentHooks:         req.MaxAgentHooks,
		MinHookIntervalSec:    req.MinHookIntervalSec,
		ContextWindow:         req.ContextWindow,
		ReserveTokens:         req.ReserveTokens,
		KeepRecentTokens:      req.KeepRecentTokens,
		Remark:                req.Remark,
		SessionId:             model.NewHostingSessionID(),
	}
	if req.Enabled != nil {
		agent.Enabled = *req.Enabled
	}
	if req.AllowAgentHooks != nil {
		agent.AllowAgentHooks = *req.AllowAgentHooks
	}
	if req.DedicatedAPIKey != "" {
		enc, err := common.EncryptSecret(req.DedicatedAPIKey)
		if err != nil {
			return nil, err
		}
		agent.DedicatedAPIKey = enc
	}
	agent.Normalize()

	var issued *IssuedToken
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		user, err := insertAgentUser(tx, agent.Name)
		if err != nil {
			return err
		}
		agent.UserId = user.Id
		now := common.GetTimestamp()
		agent.CreatedAt = now
		agent.UpdatedAt = now
		if err := tx.Create(agent).Error; err != nil {
			return err
		}
		perms := req.Permissions
		if req.ApplyRecommendedPerms {
			perms = mergePermissions(authz.RecommendedHostingAgentPermissions(), req.Permissions)
		}
		return authz.SetHostingAgentPermissionsInTx(tx, user.Id, perms)
	})
	if err != nil {
		return nil, err
	}
	if err := authz.ReloadPolicy(); err != nil {
		common.SysError("hosting reload policy after create: " + err.Error())
	}
	authz.MarkHostingAgentUser(agent.UserId)

	result := &CreateAgentResult{Agent: PublicAgent(agent)}
	if req.IssueToken {
		issued, err = IssueAgentToken(agent.Id, req.TokenName, req.TokenAllowIPs)
		if err != nil {
			return result, err
		}
		result.Secret = issued.Secret
		result.Agent = PublicAgent(agent)
	}
	return result, nil
}

func UpdateAgent(id int, req CreateAgentRequest) (*AgentView, error) {
	agent, err := model.GetHostingAgentById(id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"updated_at": common.GetTimestamp(),
	}
	if req.Name != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.HandoffUserId != 0 {
		updates["handoff_user_id"] = req.HandoffUserId
	}
	if req.BrainSource != "" {
		updates["brain_source"] = req.BrainSource
		if req.BrainSource != constant.HostingBrainDedicated {
			updates["dedicated_api_key"] = ""
			updates["dedicated_base_url"] = ""
		}
	}
	if req.BrainModel != "" {
		updates["brain_model"] = req.BrainModel
	}
	updates["brain_group"] = req.BrainGroup
	updates["brain_channel_id"] = req.BrainChannelId
	if req.DedicatedBaseURL != "" {
		updates["dedicated_base_url"] = strings.TrimSpace(req.DedicatedBaseURL)
	}
	if req.DedicatedAPIKey != "" {
		enc, err := common.EncryptSecret(req.DedicatedAPIKey)
		if err != nil {
			return nil, err
		}
		updates["dedicated_api_key"] = enc
	}
	if req.DedicatedAPIType != "" {
		updates["dedicated_api_type"] = req.DedicatedAPIType
	}
	if req.DedicatedHeaders != "" {
		updates["dedicated_headers"] = req.DedicatedHeaders
	}
	if req.DedicatedTimeoutSec > 0 {
		updates["dedicated_timeout_sec"] = req.DedicatedTimeoutSec
	}
	if req.DefaultChannelGroups != "" {
		updates["default_channel_groups"] = req.DefaultChannelGroups
	}
	if req.MaxActionsPerIncident > 0 {
		updates["max_actions_per_incident"] = req.MaxActionsPerIncident
	}
	updates["dry_run"] = req.DryRun
	if req.DailyTokenBudget > 0 {
		updates["daily_token_budget"] = req.DailyTokenBudget
	}
	if req.WakeMergeWindowSec > 0 {
		updates["wake_merge_window_sec"] = req.WakeMergeWindowSec
	}
	if req.MaxWakesPerHour > 0 {
		updates["max_wakes_per_hour"] = req.MaxWakesPerHour
	}
	if req.AllowAgentHooks != nil {
		updates["allow_agent_hooks"] = *req.AllowAgentHooks
	}
	if req.MaxAgentHooks > 0 {
		updates["max_agent_hooks"] = req.MaxAgentHooks
	}
	if req.MinHookIntervalSec > 0 {
		updates["min_hook_interval_sec"] = req.MinHookIntervalSec
	}
	if req.ContextWindow > 0 {
		updates["context_window"] = req.ContextWindow
	}
	if req.ReserveTokens > 0 {
		updates["reserve_tokens"] = req.ReserveTokens
	}
	if req.KeepRecentTokens > 0 {
		updates["keep_recent_tokens"] = req.KeepRecentTokens
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}
	if err := model.DB.Model(agent).Updates(updates).Error; err != nil {
		return nil, err
	}
	if req.Permissions != nil {
		if err := authz.SetHostingAgentPermissions(agent.UserId, req.Permissions); err != nil {
			return nil, err
		}
		_ = authz.ReloadPolicy()
	}
	agent, err = model.GetHostingAgentById(id)
	if err != nil {
		return nil, err
	}
	return PublicAgent(agent), nil
}

func DeleteAgent(id int) error {
	agent, err := model.GetHostingAgentById(id)
	if err != nil {
		return err
	}
	_ = authz.ClearUserAuthorization(agent.UserId)
	authz.UnmarkHostingAgentUser(agent.UserId)
	if err := model.DB.Delete(&model.User{}, "id = ?", agent.UserId).Error; err != nil {
		common.SysError("failed to delete hosting agent user: " + err.Error())
	}
	_ = model.DB.Where("agent_id = ?", id).Delete(&model.HostingAgentToken{}).Error
	_ = model.DB.Where("agent_id = ? AND owner = ?", id, constant.HostingHookOwnerAgent).Delete(&model.HostingHook{}).Error
	return model.DeleteHostingAgent(id)
}

func RotateAgentSession(id int) (*AgentView, error) {
	agent, err := model.GetHostingAgentById(id)
	if err != nil {
		return nil, err
	}
	if err := model.DB.Model(agent).Updates(map[string]any{
		"session_id":           model.NewHostingSessionID(),
		"last_compact_summary": "",
		"updated_at":           common.GetTimestamp(),
	}).Error; err != nil {
		return nil, err
	}
	agent, err = model.GetHostingAgentById(id)
	if err != nil {
		return nil, err
	}
	return PublicAgent(agent), nil
}

func insertAgentUser(tx *gorm.DB, name string) (*model.User, error) {
	password, err := common.Password2Hash(common.GetRandomString(32))
	if err != nil {
		return nil, err
	}
	suffix := strings.ToLower(common.GetRandomString(8))
	username := "ha-" + suffix
	if len(username) > model.UserNameMaxLength {
		username = username[:model.UserNameMaxLength]
	}
	display := name
	if len(display) > 20 {
		display = display[:20]
	}
	user := &model.User{
		Username:    username,
		Password:    password,
		DisplayName: display,
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		AccountKind: constant.AccountKindAgent,
		Group:       "default",
		Quota:       0,
	}
	if err := tx.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func mergePermissions(base, extra authz.PermissionsMap) authz.PermissionsMap {
	if extra == nil {
		return base
	}
	for resource, actions := range extra {
		if base[resource] == nil {
			base[resource] = map[string]bool{}
		}
		for action, allowed := range actions {
			base[resource][action] = allowed
		}
	}
	return base
}
