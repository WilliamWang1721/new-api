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
	Agent     *model.HostingAgent
	UserID    int
	Role      int
	TokenID   int
	ClientIP  string
	Terminate bool
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
}

var toolRate = &ToolRateLimit{window: map[int][]time.Time{}}

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
		return nil, fmt.Errorf("tool rate limit exceeded")
	}
	tool := toolByName(name)
	if tool == nil {
		return nil, fmt.Errorf("unknown tool %s", name)
	}
	if !toolAllowed(ctx, tool) {
		return nil, fmt.Errorf("tool %s is not authorized", name)
	}
	if args == nil {
		args = map[string]any{}
	}
	if err := protectBrainChannel(ctx, name, args); err != nil {
		return nil, err
	}
	if ctx.Agent.DryRun && tool.Mutating && name != "handoff_incident" && name != "sleep_until_hook" {
		if name != "manage_multi_keys" || (argString(args, "action") != "" && argString(args, "action") != "get_key_status") {
			return map[string]any{"dry_run": true, "tool": name, "args": args}, nil
		}
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
