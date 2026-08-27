package hosting

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

func NormalizePermissionPreset(preset string) string {
	switch strings.TrimSpace(preset) {
	case constant.HostingPresetWatch, constant.HostingPresetOperate, constant.HostingPresetFull:
		return strings.TrimSpace(preset)
	default:
		return constant.HostingPresetOperate
	}
}

func NormalizeAutoReviewMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case constant.HostingAutoReviewOff, constant.HostingAutoReviewConservative, constant.HostingAutoReviewBalanced, constant.HostingAutoReviewAggressive:
		return strings.TrimSpace(mode)
	default:
		return constant.HostingAutoReviewBalanced
	}
}

func NormalizeBriefingMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case constant.HostingBriefingOff, constant.HostingBriefingEveryOpen, constant.HostingBriefingDaily:
		return strings.TrimSpace(mode)
	default:
		return constant.HostingBriefingEveryOpen
	}
}

func ToolRiskLevel(name string, mutating bool) string {
	if !mutating {
		return constant.HostingToolRiskRead
	}
	switch name {
	case "test_channel", "fix_abilities", "fetch_upstream_models":
		return constant.HostingToolRiskLow
	case "set_channel_status", "update_channel_routing", "create_hook", "update_hook", "disable_hook", "update_system_setting":
		return constant.HostingToolRiskMedium
	default:
		return constant.HostingToolRiskHigh
	}
}

func AutoReviewAllows(mode, risk string) bool {
	mode = NormalizeAutoReviewMode(mode)
	switch mode {
	case constant.HostingAutoReviewAggressive:
		return risk != constant.HostingToolRiskHigh
	case constant.HostingAutoReviewBalanced:
		return risk == constant.HostingToolRiskRead || risk == constant.HostingToolRiskLow
	case constant.HostingAutoReviewConservative:
		return risk == constant.HostingToolRiskRead
	default:
		return risk == constant.HostingToolRiskRead
	}
}

func actorCanAutoExecute(ctx *ToolContext) bool {
	if ctx == nil {
		return false
	}
	if ctx.ActorRole >= common.RoleAdminUser {
		return true
	}
	return ctx.ActorUserID == 0 || ctx.ActorUserID == ctx.UserID
}

func BrainIsConfigured(agent *model.HostingAgent) bool {
	if agent == nil || strings.TrimSpace(agent.BrainModel) == "" {
		return false
	}
	if agent.BrainSource == constant.HostingBrainDedicated {
		return strings.TrimSpace(agent.DedicatedBaseURL) != ""
	}
	return true
}
