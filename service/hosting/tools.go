package hosting

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
)

type ToolContext struct {
	Agent       *model.HostingAgent
	UserID      int
	Role        int
	TokenID     int
	ClientIP    string
	Terminate   bool
	FromRunner  bool
	IncidentKey string
	ActorUserID int
	ActorRole   int
	SkipReview  bool
	Interactive bool
}

type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
	Permission  *authz.Permission
	AlwaysAllow bool
	Mutating    bool
	Handler     func(ctx *ToolContext, args map[string]any) (any, error)
}

type ToolRateLimit struct {
	mu     sync.Mutex
	window map[int][]time.Time
	inc    map[string][]time.Time
}

var toolRate = &ToolRateLimit{window: map[int][]time.Time{}, inc: map[string][]time.Time{}}

const maxToolCallsPerMinute = 60

func (r *ToolRateLimit) Allow(agentId int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	kept := r.window[agentId][:0]
	for _, ts := range r.window[agentId] {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= maxToolCallsPerMinute {
		r.window[agentId] = kept
		return false
	}
	r.window[agentId] = append(kept, now)
	return true
}

func (r *ToolRateLimit) AllowIncident(key string) bool {
	if key == "" {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.inc == nil {
		r.inc = map[string][]time.Time{}
	}
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	kept := r.inc[key][:0]
	for _, ts := range r.inc[key] {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= constant.HostingMaxToolCallsPerIncidentMin {
		r.inc[key] = kept
		return false
	}
	r.inc[key] = append(kept, now)
	return true
}

func toolSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func allTools() []*ToolSpec {
	return []*ToolSpec{
		{
			Name:        "get_runtime_snapshot",
			Description: "Get hosting status, auto-disabled channel count, and recent error logs.",
			InputSchema: toolSchema(nil),
			Permission:  &authz.HostingRead,
			Handler:     toolGetRuntimeSnapshot,
		},
		{
			Name:        "get_system_status",
			Description: "Get public system status including hosting state.",
			InputSchema: toolSchema(nil),
			AlwaysAllow: true,
			Handler:     toolGetSystemStatus,
		},
		{
			Name:        "list_channels",
			Description: "List channels without secrets.",
			InputSchema: toolSchema(map[string]any{
				"page":      map[string]any{"type": "integer"},
				"page_size": map[string]any{"type": "integer"},
				"keyword":   map[string]any{"type": "string"},
			}),
			Permission: &authz.ChannelRead,
			Handler:    toolListChannels,
		},
		{
			Name:        "get_channel",
			Description: "Get a channel by id without secrets.",
			InputSchema: toolSchema(map[string]any{"id": map[string]any{"type": "integer"}}, "id"),
			Permission:  &authz.ChannelRead,
			Handler:     toolGetChannel,
		},
		{
			Name:        "search_error_logs",
			Description: "Search recent error logs.",
			InputSchema: toolSchema(map[string]any{
				"channel_id": map[string]any{"type": "integer"},
				"keyword":    map[string]any{"type": "string"},
				"limit":      map[string]any{"type": "integer"},
			}),
			Permission: &authz.LogRead,
			Handler:    toolSearchErrorLogs,
		},
		{
			Name:        "suggest_channel_groups",
			Description: "Suggest existing routing groups to assign to a channel.",
			InputSchema: toolSchema(nil),
			Permission:  &authz.ChannelRead,
			Handler:     toolSuggestChannelGroups,
		},
		{
			Name:        "test_channel",
			Description: "Probe a channel upstream for basic connectivity.",
			InputSchema: toolSchema(map[string]any{"id": map[string]any{"type": "integer"}}, "id"),
			Permission:  &authz.ChannelOperate,
			Mutating:    true,
			Handler:     toolTestChannel,
		},
		{
			Name:        "set_channel_status",
			Description: "Enable or disable a channel. Cannot target the agent's current brain channel.",
			InputSchema: toolSchema(map[string]any{
				"id":     map[string]any{"type": "integer"},
				"status": map[string]any{"type": "integer"},
				"reason": map[string]any{"type": "string"},
			}, "id", "status"),
			Permission: &authz.ChannelOperate,
			Mutating:   true,
			Handler:    toolSetChannelStatus,
		},
		{
			Name:        "fix_abilities",
			Description: "Rebuild channel abilities from current channel rows.",
			InputSchema: toolSchema(nil),
			Permission:  &authz.ChannelOperate,
			Mutating:    true,
			Handler:     toolFixAbilities,
		},
		{
			Name:        "update_channel_routing",
			Description: "Update non-sensitive channel routing: groups, models, name.",
			InputSchema: toolSchema(map[string]any{
				"id":     map[string]any{"type": "integer"},
				"group":  map[string]any{"type": "string"},
				"models": map[string]any{"type": "string"},
				"name":   map[string]any{"type": "string"},
			}, "id"),
			Permission: &authz.ChannelWrite,
			Mutating:   true,
			Handler:    toolUpdateChannelRouting,
		},
		{
			Name:        "fetch_upstream_models",
			Description: "Fetch model names advertised by a channel upstream.",
			InputSchema: toolSchema(map[string]any{"id": map[string]any{"type": "integer"}}, "id"),
			Permission:  &authz.ChannelOperate,
			Handler:     toolFetchUpstreamModels,
		},
		{
			Name:        "manage_multi_keys",
			Description: "List or enable/disable individual keys on a multi-key channel. Never returns full secrets.",
			InputSchema: toolSchema(map[string]any{
				"id":     map[string]any{"type": "integer"},
				"action": map[string]any{"type": "string", "description": "get_key_status, enable_key, or disable_key"},
				"index":  map[string]any{"type": "integer"},
				"reason": map[string]any{"type": "string"},
			}, "id"),
			Permission: &authz.ChannelOperate,
			Mutating:   true,
			Handler:    toolManageMultiKeys,
		},
		{
			Name:        "create_channel",
			Description: "Create a channel. Empty group is filled with hosting default groups.",
			InputSchema: toolSchema(map[string]any{
				"name":     map[string]any{"type": "string"},
				"type":     map[string]any{"type": "integer"},
				"key":      map[string]any{"type": "string"},
				"base_url": map[string]any{"type": "string"},
				"models":   map[string]any{"type": "string"},
				"group":    map[string]any{"type": "string"},
			}, "name", "key", "models"),
			Permission: &authz.ChannelSensitiveWrite,
			Mutating:   true,
			Handler:    toolCreateChannel,
		},
		{
			Name:        "update_channel_endpoint",
			Description: "Update a channel key or base URL. Cannot target the agent's current brain channel.",
			InputSchema: toolSchema(map[string]any{
				"id":       map[string]any{"type": "integer"},
				"key":      map[string]any{"type": "string"},
				"base_url": map[string]any{"type": "string"},
			}, "id"),
			Permission: &authz.ChannelSensitiveWrite,
			Mutating:   true,
			Handler:    toolUpdateChannelEndpoint,
		},
		{
			Name:        "list_users",
			Description: "List human users without secrets.",
			InputSchema: toolSchema(map[string]any{
				"page":      map[string]any{"type": "integer"},
				"page_size": map[string]any{"type": "integer"},
			}),
			Permission: &authz.UserRead,
			Handler:    toolListUsers,
		},
		{
			Name:        "get_user",
			Description: "Get a human user by id without secrets.",
			InputSchema: toolSchema(map[string]any{"id": map[string]any{"type": "integer"}}, "id"),
			Permission:  &authz.UserRead,
			Handler:     toolGetUser,
		},
		{
			Name:        "list_redemptions",
			Description: "List redemption codes.",
			InputSchema: toolSchema(map[string]any{
				"page":      map[string]any{"type": "integer"},
				"page_size": map[string]any{"type": "integer"},
			}),
			Permission: &authz.RedemptionRead,
			Handler:    toolListRedemptions,
		},
		{
			Name:        "list_groups",
			Description: "List configured routing groups.",
			InputSchema: toolSchema(nil),
			Permission:  &authz.GroupRead,
			Handler:     toolListGroups,
		},
		{
			Name:        "update_group_ratio",
			Description: "Set the ratio for an existing routing group. Does not create payment settings.",
			InputSchema: toolSchema(map[string]any{
				"group": map[string]any{"type": "string"},
				"ratio": map[string]any{"type": "number"},
			}, "group", "ratio"),
			Permission: &authz.GroupWrite,
			Mutating:   true,
			Handler:    toolUpdateGroupRatio,
		},
		{
			Name:        "list_tokens",
			Description: "List a user's API tokens without secrets.",
			InputSchema: toolSchema(map[string]any{
				"user_id":   map[string]any{"type": "integer"},
				"page":      map[string]any{"type": "integer"},
				"page_size": map[string]any{"type": "integer"},
			}, "user_id"),
			Permission: &authz.TokenRead,
			Handler:    toolListTokens,
		},
		{
			Name:        "get_token",
			Description: "Get a user API token by id without the secret key.",
			InputSchema: toolSchema(map[string]any{
				"id":      map[string]any{"type": "integer"},
				"user_id": map[string]any{"type": "integer"},
			}, "id", "user_id"),
			Permission: &authz.TokenRead,
			Handler:    toolGetToken,
		},
		{
			Name:        "create_token",
			Description: "Create a user API token. The full key is never returned.",
			InputSchema: toolSchema(map[string]any{
				"user_id":         map[string]any{"type": "integer"},
				"name":            map[string]any{"type": "string"},
				"remain_quota":    map[string]any{"type": "integer"},
				"unlimited_quota": map[string]any{"type": "boolean"},
				"group":           map[string]any{"type": "string"},
				"expired_time":    map[string]any{"type": "integer"},
			}, "user_id", "name"),
			Permission: &authz.TokenWrite,
			Mutating:   true,
			Handler:    toolCreateToken,
		},
		{
			Name:        "update_token",
			Description: "Update a user API token name, status, quota, group, or IP allow list. Never returns the secret.",
			InputSchema: toolSchema(map[string]any{
				"id":           map[string]any{"type": "integer"},
				"user_id":      map[string]any{"type": "integer"},
				"name":         map[string]any{"type": "string"},
				"status":       map[string]any{"type": "integer"},
				"remain_quota": map[string]any{"type": "integer"},
				"group":        map[string]any{"type": "string"},
				"allow_ips":    map[string]any{"type": "string"},
			}, "id", "user_id"),
			Permission: &authz.TokenWrite,
			Mutating:   true,
			Handler:    toolUpdateToken,
		},
		{
			Name:        "update_user_status",
			Description: "Enable or disable a non-root human user. Cannot target root or hosting agents.",
			InputSchema: toolSchema(map[string]any{
				"id":     map[string]any{"type": "integer"},
				"status": map[string]any{"type": "integer"},
			}, "id", "status"),
			Permission: &authz.UserWrite,
			Mutating:   true,
			Handler:    toolUpdateUserStatus,
		},
		{
			Name:        "add_user_quota",
			Description: "Add wallet quota to a non-root human user. Cannot recharge via payment.",
			InputSchema: toolSchema(map[string]any{
				"id":    map[string]any{"type": "integer"},
				"quota": map[string]any{"type": "integer"},
			}, "id", "quota"),
			Permission: &authz.UserWrite,
			Mutating:   true,
			Handler:    toolAddUserQuota,
		},
		{
			Name:        "create_redemption",
			Description: "Create redemption codes. Returns generated codes so a human can distribute them.",
			InputSchema: toolSchema(map[string]any{
				"name":         map[string]any{"type": "string"},
				"quota":        map[string]any{"type": "integer"},
				"count":        map[string]any{"type": "integer"},
				"expired_time": map[string]any{"type": "integer"},
			}, "name", "quota"),
			Permission: &authz.RedemptionWrite,
			Mutating:   true,
			Handler:    toolCreateRedemption,
		},
		{
			Name:        "update_redemption_status",
			Description: "Enable or disable a redemption code.",
			InputSchema: toolSchema(map[string]any{
				"id":     map[string]any{"type": "integer"},
				"status": map[string]any{"type": "integer"},
			}, "id", "status"),
			Permission: &authz.RedemptionWrite,
			Mutating:   true,
			Handler:    toolUpdateRedemptionStatus,
		},
		{
			Name:        "list_subscription_plans",
			Description: "List subscription plans.",
			InputSchema: toolSchema(nil),
			Permission:  &authz.SubscriptionRead,
			Handler:     toolListSubscriptionPlans,
		},
		{
			Name:        "list_user_subscriptions",
			Description: "List a user's subscriptions.",
			InputSchema: toolSchema(map[string]any{
				"user_id": map[string]any{"type": "integer"},
			}, "user_id"),
			Permission: &authz.SubscriptionRead,
			Handler:    toolListUserSubscriptions,
		},
		{
			Name:        "set_subscription_plan_enabled",
			Description: "Enable or disable a subscription plan. Does not change payment keys.",
			InputSchema: toolSchema(map[string]any{
				"id":      map[string]any{"type": "integer"},
				"enabled": map[string]any{"type": "boolean"},
			}, "id", "enabled"),
			Permission: &authz.SubscriptionWrite,
			Mutating:   true,
			Handler:    toolSetSubscriptionPlanEnabled,
		},
		{
			Name:        "list_vendors",
			Description: "List vendors.",
			InputSchema: toolSchema(map[string]any{
				"page":      map[string]any{"type": "integer"},
				"page_size": map[string]any{"type": "integer"},
			}),
			Permission: &authz.VendorRead,
			Handler:    toolListVendors,
		},
		{
			Name:        "get_vendor",
			Description: "Get a vendor by id.",
			InputSchema: toolSchema(map[string]any{"id": map[string]any{"type": "integer"}}, "id"),
			Permission:  &authz.VendorRead,
			Handler:     toolGetVendor,
		},
		{
			Name:        "create_vendor",
			Description: "Create a vendor record. Does not change payment settings.",
			InputSchema: toolSchema(map[string]any{
				"name":        map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"icon":        map[string]any{"type": "string"},
			}, "name"),
			Permission: &authz.VendorWrite,
			Mutating:   true,
			Handler:    toolCreateVendor,
		},
		{
			Name:        "list_model_meta",
			Description: "List model metadata records.",
			InputSchema: toolSchema(map[string]any{
				"page":      map[string]any{"type": "integer"},
				"page_size": map[string]any{"type": "integer"},
				"keyword":   map[string]any{"type": "string"},
			}),
			Permission: &authz.ModelMetaRead,
			Handler:    toolListModelMeta,
		},
		{
			Name:        "get_model_meta",
			Description: "Get a model metadata record by id.",
			InputSchema: toolSchema(map[string]any{"id": map[string]any{"type": "integer"}}, "id"),
			Permission:  &authz.ModelMetaRead,
			Handler:     toolGetModelMeta,
		},
		{
			Name:        "list_system_tasks",
			Description: "List recent scheduled system tasks without payloads.",
			InputSchema: toolSchema(map[string]any{
				"limit": map[string]any{"type": "integer"},
			}),
			Permission: &authz.SystemTaskRead,
			Handler:    toolListSystemTasks,
		},
		{
			Name:        "list_hooks",
			Description: "List system hooks and this agent's hooks.",
			InputSchema: toolSchema(nil),
			AlwaysAllow: true,
			Handler:     toolListHooks,
		},
		{
			Name:        "create_hook",
			Description: "Create a reminder or condition hook for this agent.",
			InputSchema: toolSchema(map[string]any{
				"name":         map[string]any{"type": "string"},
				"kind":         map[string]any{"type": "string"},
				"event_name":   map[string]any{"type": "string"},
				"cron":         map[string]any{"type": "string"},
				"fire_at":      map[string]any{"type": "integer"},
				"condition":    map[string]any{"type": "string"},
				"condition_n":  map[string]any{"type": "integer"},
				"channel_id":   map[string]any{"type": "integer"},
				"wake_mode":    map[string]any{"type": "string"},
				"cooldown_sec": map[string]any{"type": "integer"},
				"prompt_hint":  map[string]any{"type": "string"},
			}, "name", "kind"),
			AlwaysAllow: true,
			Mutating:    true,
			Handler:     toolCreateHook,
		},
		{
			Name:        "update_hook",
			Description: "Update one of this agent's hooks. System hooks are read-only.",
			InputSchema: toolSchema(map[string]any{
				"id":           map[string]any{"type": "integer"},
				"enabled":      map[string]any{"type": "boolean"},
				"wake_mode":    map[string]any{"type": "string"},
				"cooldown_sec": map[string]any{"type": "integer"},
				"prompt_hint":  map[string]any{"type": "string"},
				"fire_at":      map[string]any{"type": "integer"},
			}, "id"),
			AlwaysAllow: true,
			Mutating:    true,
			Handler:     toolUpdateHook,
		},
		{
			Name:        "disable_hook",
			Description: "Disable one of this agent's hooks.",
			InputSchema: toolSchema(map[string]any{"id": map[string]any{"type": "integer"}}, "id"),
			AlwaysAllow: true,
			Mutating:    true,
			Handler:     toolDisableHook,
		},
		{
			Name:        "delete_hook",
			Description: "Delete one of this agent's hooks. System hooks cannot be deleted.",
			InputSchema: toolSchema(map[string]any{"id": map[string]any{"type": "integer"}}, "id"),
			AlwaysAllow: true,
			Mutating:    true,
			Handler:     toolDeleteHook,
		},
		{
			Name:        "get_setup_checklist",
			Description: "Get a simple first-time setup checklist: site name, sign-ups, channels, and whether the steward can talk.",
			InputSchema: toolSchema(nil),
			AlwaysAllow: true,
			Handler:     toolGetSetupChecklist,
		},
		{
			Name:        "list_system_settings",
			Description: "List system settings with plain-language titles. Secrets are hidden. Optional keyword or category: site, access, billing, operations, features.",
			InputSchema: toolSchema(map[string]any{
				"keyword":  map[string]any{"type": "string"},
				"category": map[string]any{"type": "string"},
			}),
			Permission: &authz.OptionRead,
			Handler:    toolListSystemSettings,
		},
		{
			Name:        "update_system_setting",
			Description: "Change one system setting by key. Explain the change in plain language first. Never print secret values. Risky keys wait for approval.",
			InputSchema: toolSchema(map[string]any{
				"key":   map[string]any{"type": "string"},
				"value": map[string]any{"type": "string"},
			}, "key", "value"),
			Permission: &authz.OptionWrite,
			Mutating:   true,
			Handler:    toolUpdateSystemSetting,
		},
		{
			Name:        "request_user_access",
			Description: "File a user access request: quota, group, enable, or admin. Small quota may be auto-approved.",
			InputSchema: toolSchema(map[string]any{
				"request_type":   map[string]any{"type": "string", "description": "quota, group, enable, or admin"},
				"target_user_id": map[string]any{"type": "integer"},
				"quota":          map[string]any{"type": "integer"},
				"group":          map[string]any{"type": "string"},
				"note":           map[string]any{"type": "string"},
			}, "request_type"),
			AlwaysAllow: true,
			Handler:     queueUserAccessRequest,
		},
		{
			Name:        "list_approvals",
			Description: "List recent permission and tool-action approvals.",
			InputSchema: toolSchema(map[string]any{
				"status": map[string]any{"type": "string"},
			}),
			AlwaysAllow: true,
			Handler:     toolListApprovals,
		},
		{
			Name:        "handoff_incident",
			Description: "Hand the current issue to a human. Always available.",
			InputSchema: toolSchema(map[string]any{
				"summary": map[string]any{"type": "string"},
				"reason":  map[string]any{"type": "string"},
			}, "summary"),
			AlwaysAllow: true,
			Mutating:    true,
			Handler:     toolHandoffIncident,
		},
		{
			Name:        "sleep_until_hook",
			Description: "End this wake cycle and keep the session. Further work waits for a hook.",
			InputSchema: toolSchema(map[string]any{
				"note": map[string]any{"type": "string"},
			}),
			AlwaysAllow: true,
			Handler:     toolSleepUntilHook,
		},
	}
}

func toolListApprovals(ctx *ToolContext, args map[string]any) (any, error) {
	status := argString(args, "status")
	items, err := ListApprovalsForActor(ctx.Agent.Id, ctx.ActorUserID, ctx.ActorRole, status)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items}, nil
}

func toolByName(name string) *ToolSpec {
	for _, tool := range allTools() {
		if tool.Name == name {
			return tool
		}
	}
	return nil
}

func ListToolsForAgent(ctx *ToolContext) []map[string]any {
	out := make([]map[string]any, 0)
	for _, tool := range allTools() {
		if !toolAllowed(ctx, tool) {
			continue
		}
		out = append(out, map[string]any{
			"name":         tool.Name,
			"description":  tool.Description,
			"input_schema": tool.InputSchema,
		})
	}
	return out
}

func toolAllowed(ctx *ToolContext, tool *ToolSpec) bool {
	if tool.AlwaysAllow || tool.Permission == nil {
		return true
	}
	if ctx == nil || ctx.Agent == nil {
		return false
	}
	if tool.Permission.Action == authz.ActionSecretView {
		return false
	}
	return authz.Can(ctx.UserID, ctx.Role, *tool.Permission)
}

func ExecuteTool(ctx *ToolContext, name string, args map[string]any) (any, error) {
	if ctx == nil || ctx.Agent == nil {
		return nil, fmt.Errorf("missing hosting agent context")
	}
	if !IsReady() {
		return nil, fmt.Errorf("hosting is not ready")
	}
	if !toolRate.Allow(ctx.Agent.Id) {
		err := fmt.Errorf("tool rate limit exceeded")
		if ctx.FromRunner {
			return nil, handoffErr("rate_limit", err.Error())
		}
		return nil, err
	}
	if !toolRate.AllowIncident(ctx.IncidentKey) {
		err := fmt.Errorf("per-incident tool rate limit exceeded")
		if ctx.FromRunner {
			return nil, handoffErr("rate_limit", err.Error())
		}
		return nil, err
	}
	tool := toolByName(name)
	if tool == nil {
		return nil, fmt.Errorf("unknown tool %s", name)
	}
	if !toolAllowed(ctx, tool) {
		err := fmt.Errorf("tool %s is not authorized", name)
		if ctx.FromRunner {
			return nil, handoffErr("unauthorized", err.Error())
		}
		return nil, err
	}
	if args == nil {
		args = map[string]any{}
	}
	if err := protectBrainChannel(ctx, name, args); err != nil {
		if ctx.FromRunner {
			return nil, handoffErr("brain_channel", err.Error())
		}
		return nil, err
	}
	if ctx.Agent.DryRun && tool.Mutating && name != "handoff_incident" && name != "sleep_until_hook" && name != "request_user_access" {
		if name != "manage_multi_keys" || (argString(args, "action") != "" && argString(args, "action") != "get_key_status") {
			if ctx.FromRunner && !ctx.Interactive {
				return nil, handoffErr("dry_run", "dry-run blocked mutating tool "+name)
			}
			return map[string]any{"dry_run": true, "tool": name, "args": args}, nil
		}
	}
	queued, handled, err := maybeQueueToolApproval(ctx, tool, args)
	if err != nil {
		return nil, err
	}
	if handled {
		return queued, nil
	}
	result, err := tool.Handler(ctx, args)
	auditTool(ctx, name, args, err)
	if err != nil {
		return nil, err
	}
	return sanitizeToolResult(result), nil
}

func protectBrainChannel(ctx *ToolContext, name string, args map[string]any) error {
	if ctx.Agent.BrainSource != constant.HostingBrainInternal || ctx.Agent.BrainChannelId <= 0 {
		return nil
	}
	switch name {
	case "set_channel_status", "update_channel_endpoint", "manage_multi_keys":
	default:
		return nil
	}
	if name == "manage_multi_keys" {
		action := argString(args, "action")
		if action == "" || action == "get_key_status" {
			return nil
		}
	}
	id := argInt(args, "id")
	if id == ctx.Agent.BrainChannelId {
		return fmt.Errorf("refusing to change the current brain channel #%d", id)
	}
	return nil
}

func auditTool(ctx *ToolContext, name string, args map[string]any, execErr error) {
	params := map[string]any{"tool": name}
	if id := argInt(args, "id"); id > 0 {
		params["id"] = id
	}
	adminInfo := map[string]any{
		"admin_id":    ctx.UserID,
		"auth_method": constant.HostingAuthMethod,
		"agent_id":    ctx.Agent.Id,
	}
	if execErr != nil {
		adminInfo["error"] = execErr.Error()
	}
	model.RecordOperationAuditLog(ctx.UserID, "hosting tool "+name, ctx.ClientIP, "hosting.tool", params, adminInfo, map[string]any{
		"tool":    name,
		"success": execErr == nil,
	})
}

func sanitizeToolResult(v any) any {
	raw, err := common.Marshal(v)
	if err != nil {
		return v
	}
	var generic any
	if err := common.Unmarshal(raw, &generic); err != nil {
		return v
	}
	stripSecrets(generic)
	return generic
}

func stripSecrets(v any) {
	switch item := v.(type) {
	case map[string]any:
		for key, val := range item {
			lower := strings.ToLower(key)
			if lower == "key" || lower == "access_token" || strings.Contains(lower, "secret") || strings.Contains(lower, "api_key") || lower == "password" {
				delete(item, key)
				continue
			}
			stripSecrets(val)
		}
	case []any:
		for _, val := range item {
			stripSecrets(val)
		}
	}
}

func argInt(args map[string]any, key string) int {
	v, ok := args[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		var i int
		_, _ = fmt.Sscanf(n, "%d", &i)
		return i
	default:
		return 0
	}
}

func argString(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return fmt.Sprint(s)
	}
}

func argBool(args map[string]any, key string) (bool, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func argFloat(args map[string]any, key string) float64 {
	v, ok := args[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		var f float64
		_, _ = fmt.Sscanf(n, "%f", &f)
		return f
	default:
		return 0
	}
}
