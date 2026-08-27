package hosting

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
)

type StewardSettings struct {
	Enabled          bool   `json:"enabled"`
	BrainSource      string `json:"brain_source"`
	BrainModel       string `json:"brain_model"`
	BrainGroup       string `json:"brain_group"`
	BrainChannelId   int    `json:"brain_channel_id"`
	DedicatedBaseURL string `json:"dedicated_base_url"`
	DedicatedAPIKey  string `json:"dedicated_api_key,omitempty"`
	DedicatedAPIType string `json:"dedicated_api_type"`
	DedicatedKeySet  bool   `json:"dedicated_key_set"`
	PermissionPreset string `json:"permission_preset"`
	AutoReviewMode   string `json:"auto_review_mode"`
	BriefingMode     string `json:"briefing_mode"`
	DailyTokenBudget int    `json:"daily_token_budget"`
	DryRun           bool   `json:"dry_run"`
	NeedsBrain       bool   `json:"needs_brain"`
	AgentId          int    `json:"agent_id"`
	PendingApprovals int64  `json:"pending_approvals"`
}

type StewardStatus struct {
	Ready            bool   `json:"ready"`
	NeedsBrain       bool   `json:"needs_brain"`
	PanelEnabled     bool   `json:"panel_enabled"`
	EnvEnabled       bool   `json:"env_enabled"`
	DefaultAgentId   int    `json:"default_agent_id"`
	PendingApprovals int64  `json:"pending_approvals"`
	CanReview        bool   `json:"can_review"`
	PermissionPreset string `json:"permission_preset"`
	AutoReviewMode   string `json:"auto_review_mode"`
	BriefingMode     string `json:"briefing_mode"`
	PracticeMode     bool   `json:"practice_mode"`
}

func EnsureDefaultSteward() error {
	if model.DB == nil {
		return fmt.Errorf("database is not initialized")
	}
	existing, err := model.GetDefaultHostingAgent()
	if err == nil && existing != nil {
		if !existing.IsDefault {
			_ = model.SetDefaultHostingAgent(existing.Id)
		}
		if strings.EqualFold(strings.TrimSpace(existing.Name), "ops-agent") {
			_ = model.DB.Model(existing).Update("name", constant.DefaultHostingStewardName).Error
		}
		upgrades := map[string]any{}
		if strings.TrimSpace(existing.PermissionPreset) == "" {
			upgrades["permission_preset"] = constant.HostingPresetOperate
			existing.PermissionPreset = constant.HostingPresetOperate
		}
		if strings.TrimSpace(existing.AutoReviewMode) == "" {
			upgrades["auto_review_mode"] = constant.HostingAutoReviewBalanced
		}
		if strings.TrimSpace(existing.BriefingMode) == "" {
			upgrades["briefing_mode"] = constant.HostingBriefingEveryOpen
		}
		if len(upgrades) > 0 {
			upgrades["updated_at"] = common.GetTimestamp()
			_ = model.DB.Model(existing).Updates(upgrades).Error
		}
		grantStewardOptionAccess(existing)
		return nil
	}
	enabled := true
	_, err = CreateAgent(CreateAgentRequest{
		Name:                  constant.DefaultHostingStewardName,
		Enabled:               &enabled,
		BrainSource:           constant.HostingBrainInternal,
		BrainGroup:            "default",
		PermissionPreset:      constant.HostingPresetOperate,
		AutoReviewMode:        constant.HostingAutoReviewBalanced,
		BriefingMode:          constant.HostingBriefingEveryOpen,
		ApplyRecommendedPerms: true,
		IsDefault:             true,
		DailyTokenBudget:      constant.DefaultHostingDailyBudget,
	})
	return err
}

func PublicStewardStatus(actorRole int) StewardStatus {
	rt := GetRuntime()
	st := StewardStatus{
		Ready:        rt.Enabled && rt.State == constant.HostingStatusReady,
		PanelEnabled: rt.PanelEnabled,
		EnvEnabled:   rt.EnvEnabled,
		CanReview:    actorRole >= common.RoleAdminUser,
	}
	agent, err := model.GetDefaultHostingAgent()
	if err != nil || agent == nil {
		st.NeedsBrain = true
		return st
	}
	st.DefaultAgentId = agent.Id
	st.NeedsBrain = !BrainIsConfigured(agent)
	st.PermissionPreset = NormalizePermissionPreset(agent.PermissionPreset)
	st.AutoReviewMode = NormalizeAutoReviewMode(agent.AutoReviewMode)
	st.BriefingMode = NormalizeBriefingMode(agent.BriefingMode)
	st.PracticeMode = agent.DryRun
	st.PendingApprovals, _ = model.CountPendingHostingApprovals(agent.Id)
	return st
}

func LoadStewardSettings() (*StewardSettings, error) {
	agent, err := model.GetDefaultHostingAgent()
	if err != nil {
		return &StewardSettings{NeedsBrain: true, PermissionPreset: constant.HostingPresetOperate, AutoReviewMode: constant.HostingAutoReviewBalanced, BriefingMode: constant.HostingBriefingEveryOpen}, nil
	}
	pending, _ := model.CountPendingHostingApprovals(agent.Id)
	return &StewardSettings{
		Enabled:          agent.Enabled,
		BrainSource:      agent.BrainSource,
		BrainModel:       agent.BrainModel,
		BrainGroup:       agent.BrainGroup,
		BrainChannelId:   agent.BrainChannelId,
		DedicatedBaseURL: agent.DedicatedBaseURL,
		DedicatedAPIType: agent.DedicatedAPIType,
		DedicatedKeySet:  agent.DedicatedAPIKey != "",
		PermissionPreset: NormalizePermissionPreset(agent.PermissionPreset),
		AutoReviewMode:   NormalizeAutoReviewMode(agent.AutoReviewMode),
		BriefingMode:     NormalizeBriefingMode(agent.BriefingMode),
		DailyTokenBudget: agent.DailyTokenBudget,
		DryRun:           agent.DryRun,
		NeedsBrain:       !BrainIsConfigured(agent),
		AgentId:          agent.Id,
		PendingApprovals: pending,
	}, nil
}

type UpdateStewardRequest struct {
	Enabled          *bool  `json:"enabled"`
	BrainSource      string `json:"brain_source"`
	BrainModel       string `json:"brain_model"`
	BrainGroup       string `json:"brain_group"`
	BrainChannelId   int    `json:"brain_channel_id"`
	DedicatedBaseURL string `json:"dedicated_base_url"`
	DedicatedAPIKey  string `json:"dedicated_api_key"`
	DedicatedAPIType string `json:"dedicated_api_type"`
	PermissionPreset string `json:"permission_preset"`
	AutoReviewMode   string `json:"auto_review_mode"`
	BriefingMode     string `json:"briefing_mode"`
	DailyTokenBudget int    `json:"daily_token_budget"`
	DryRun           *bool  `json:"dry_run"`
}

func SaveStewardSettings(req UpdateStewardRequest) (*StewardSettings, error) {
	agent, err := model.GetDefaultHostingAgent()
	if err != nil {
		enabled := true
		created, err := CreateAgent(CreateAgentRequest{
			Name:             constant.DefaultHostingStewardName,
			Enabled:          &enabled,
			IsDefault:        true,
			PermissionPreset: constant.HostingPresetOperate,
			AutoReviewMode:   constant.HostingAutoReviewBalanced,
			BriefingMode:     constant.HostingBriefingEveryOpen,
		})
		if err != nil {
			return nil, err
		}
		agent, err = model.GetHostingAgentById(created.Agent.Id)
		if err != nil {
			return nil, err
		}
	}
	updates := map[string]any{"updated_at": common.GetTimestamp()}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.BrainSource != "" {
		updates["brain_source"] = req.BrainSource
		if req.BrainSource != constant.HostingBrainDedicated {
			updates["dedicated_api_key"] = ""
			updates["dedicated_base_url"] = ""
		}
	}
	if req.BrainModel != "" {
		updates["brain_model"] = strings.TrimSpace(req.BrainModel)
	}
	if req.BrainGroup != "" {
		updates["brain_group"] = strings.TrimSpace(req.BrainGroup)
	}
	if req.BrainChannelId > 0 {
		updates["brain_channel_id"] = req.BrainChannelId
	}
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
	if req.PermissionPreset != "" {
		preset := NormalizePermissionPreset(req.PermissionPreset)
		updates["permission_preset"] = preset
		if err := authz.SetHostingAgentPermissions(agent.UserId, authz.HostingPresetPermissions(preset)); err != nil {
			return nil, err
		}
		_ = authz.ReloadPolicy()
	}
	if req.AutoReviewMode != "" {
		updates["auto_review_mode"] = NormalizeAutoReviewMode(req.AutoReviewMode)
	}
	if req.BriefingMode != "" {
		updates["briefing_mode"] = NormalizeBriefingMode(req.BriefingMode)
	}
	if req.DailyTokenBudget > 0 {
		updates["daily_token_budget"] = req.DailyTokenBudget
	}
	if req.DryRun != nil {
		updates["dry_run"] = *req.DryRun
	}
	if err := model.DB.Model(agent).Updates(updates).Error; err != nil {
		return nil, err
	}
	_ = model.SetDefaultHostingAgent(agent.Id)
	return LoadStewardSettings()
}

func grantStewardOptionAccess(agent *model.HostingAgent) {
	if agent == nil || agent.UserId <= 0 {
		return
	}
	preset := NormalizePermissionPreset(agent.PermissionPreset)
	option := map[string]bool{authz.ActionRead: true}
	if preset != constant.HostingPresetWatch {
		option[authz.ActionWrite] = true
	}
	current := authz.Capabilities(agent.UserId, common.RoleAdminUser)
	merged := mergePermissions(current, authz.PermissionsMap{authz.ResourceOption: option})
	if err := authz.SetHostingAgentPermissions(agent.UserId, merged); err != nil {
		common.SysError("hosting failed to grant steward setting permissions: " + err.Error())
		return
	}
	_ = authz.ReloadPolicy()
}
