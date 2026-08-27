package hosting

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func toolGetRuntimeSnapshot(_ *ToolContext, _ map[string]any) (any, error) {
	return RuntimeSnapshot(), nil
}

func toolGetSystemStatus(_ *ToolContext, _ map[string]any) (any, error) {
	return map[string]any{
		"hosting": GetPublicStatus(),
		"version": common.Version,
	}, nil
}

func RuntimeSnapshot() map[string]any {
	var autoDisabled int64
	if model.DB != nil {
		_ = model.DB.Model(&model.Channel{}).Where("status = ?", common.ChannelStatusAutoDisabled).Count(&autoDisabled).Error
	}
	var logs []*model.Log
	if model.LOG_DB != nil {
		logs, _, _ = model.GetAllLogs(model.LogTypeError, time.Now().Add(-24*time.Hour).Unix(), 0, "", "", "", 0, 10, 0, "", "", "")
	}
	summaries := make([]map[string]any, 0, len(logs))
	for _, item := range logs {
		summaries = append(summaries, map[string]any{
			"id":         item.Id,
			"created_at": item.CreatedAt,
			"content":    common.LocalLogPreview(item.Content),
			"channel_id": item.ChannelId,
			"model_name": item.ModelName,
		})
	}
	rt := GetRuntime()
	inspection := map[string]any{}
	if model.DB != nil {
		latest, err := model.GetLatestSystemTasks([]string{
			model.SystemTaskTypeChannelTest,
			model.SystemTaskTypeModelUpdate,
			model.SystemTaskTypeHostingHooks,
		})
		if err == nil {
			for name, task := range latest {
				if task == nil {
					continue
				}
				inspection[name] = map[string]any{
					"status":     task.Status,
					"updated_at": task.UpdatedAt,
					"error":      task.Error,
				}
			}
		}
	}
	return map[string]any{
		"hosting":                rt.State,
		"auto_disabled_count":    autoDisabled,
		"recent_error_logs":      summaries,
		"default_channel_groups": constant.DefaultHostingChannelGroups,
		"inspection_tasks":       inspection,
		"host_resources": map[string]any{
			"alloc_mb": hostAllocMB(),
		},
	}
}

func toolListChannels(_ *ToolContext, args map[string]any) (any, error) {
	page := argInt(args, "page")
	size := argInt(args, "page_size")
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	keyword := strings.TrimSpace(argString(args, "keyword"))
	if keyword != "" {
		channels, err := model.SearchChannels(keyword, "", "", false)
		if err != nil {
			return nil, err
		}
		return map[string]any{"items": channels, "total": len(channels)}, nil
	}
	channels, err := model.GetAllChannels((page-1)*size, size, false, true)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": channels, "page": page, "page_size": size}, nil
}

func toolGetChannel(_ *ToolContext, args map[string]any) (any, error) {
	id := argInt(args, "id")
	channel, err := model.GetChannelById(id, false)
	if err != nil {
		return nil, err
	}
	return channel, nil
}

func toolSearchErrorLogs(_ *ToolContext, args map[string]any) (any, error) {
	if model.LOG_DB == nil {
		return map[string]any{"items": []any{}, "total": 0}, nil
	}
	limit := argInt(args, "limit")
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	channelId := argInt(args, "channel_id")
	logs, total, err := model.GetAllLogs(model.LogTypeError, time.Now().Add(-24*time.Hour).Unix(), 0, "", "", "", 0, limit, channelId, "", "", "")
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(logs))
	for _, item := range logs {
		out = append(out, map[string]any{
			"id":         item.Id,
			"created_at": item.CreatedAt,
			"content":    common.LocalLogPreview(item.Content),
			"channel_id": item.ChannelId,
			"model_name": item.ModelName,
			"username":   item.Username,
		})
	}
	return map[string]any{"items": out, "total": total}, nil
}

func toolSuggestChannelGroups(_ *ToolContext, _ map[string]any) (any, error) {
	return SuggestChannelGroups(), nil
}

func toolTestChannel(_ *ToolContext, args map[string]any) (any, error) {
	id := argInt(args, "id")
	channel, err := model.GetChannelById(id, true)
	if err != nil {
		return nil, err
	}
	ok, message := ProbeChannel(channel)
	return map[string]any{"id": id, "ok": ok, "message": message}, nil
}

func toolSetChannelStatus(_ *ToolContext, args map[string]any) (any, error) {
	id := argInt(args, "id")
	status := argInt(args, "status")
	reason := argString(args, "reason")
	if status != common.ChannelStatusEnabled && status != common.ChannelStatusAutoDisabled && status != common.ChannelStatusManuallyDisabled {
		return nil, fmt.Errorf("unsupported channel status")
	}
	ok := model.UpdateChannelStatus(id, "", status, reason)
	return map[string]any{"id": id, "status": status, "updated": ok}, nil
}

func toolFixAbilities(_ *ToolContext, _ map[string]any) (any, error) {
	success, fail, err := model.FixAbility()
	if err != nil {
		return nil, err
	}
	return map[string]any{"success": success, "fail": fail}, nil
}

func toolUpdateChannelRouting(ctx *ToolContext, args map[string]any) (any, error) {
	id := argInt(args, "id")
	channel, err := model.GetChannelById(id, true)
	if err != nil {
		return nil, err
	}
	if group := argString(args, "group"); group != "" {
		channel.Group = ResolveChannelGroups(group, ctx.Agent.DefaultChannelGroups)
	} else if strings.TrimSpace(channel.Group) == "" {
		channel.Group = ResolveChannelGroups("", ctx.Agent.DefaultChannelGroups)
	}
	if models := argString(args, "models"); models != "" {
		channel.Models = models
	}
	if name := argString(args, "name"); name != "" {
		channel.Name = name
	}
	if err := channel.Update(); err != nil {
		return nil, err
	}
	return map[string]any{"id": channel.Id, "group": channel.Group, "models": channel.Models}, nil
}

func toolFetchUpstreamModels(_ *ToolContext, args map[string]any) (any, error) {
	id := argInt(args, "id")
	channel, err := model.GetChannelById(id, true)
	if err != nil {
		return nil, err
	}
	names, message := FetchUpstreamModelNames(channel)
	return map[string]any{"id": id, "models": names, "message": message}, nil
}

func toolManageMultiKeys(_ *ToolContext, args map[string]any) (any, error) {
	id := argInt(args, "id")
	channel, err := model.GetChannelById(id, true)
	if err != nil {
		return nil, err
	}
	if !channel.ChannelInfo.IsMultiKey {
		return nil, fmt.Errorf("channel is not in multi-key mode")
	}
	action := strings.TrimSpace(argString(args, "action"))
	if action == "" {
		action = "get_key_status"
	}
	keys := channel.GetKeys()
	if action == "get_key_status" {
		items := make([]map[string]any, 0, len(keys))
		for i, key := range keys {
			preview := key
			if len(preview) > 10 {
				preview = preview[:10] + "..."
			}
			status := 1
			if channel.ChannelInfo.MultiKeyStatusList != nil {
				if s, ok := channel.ChannelInfo.MultiKeyStatusList[i]; ok {
					status = s
				}
			}
			items = append(items, map[string]any{
				"index":   i,
				"status":  status,
				"preview": preview,
			})
		}
		return map[string]any{"id": id, "keys": items}, nil
	}
	index := argInt(args, "index")
	if index < 0 || index >= len(keys) {
		return nil, fmt.Errorf("key index out of range")
	}
	status := common.ChannelStatusEnabled
	reason := argString(args, "reason")
	switch action {
	case "enable_key":
		status = common.ChannelStatusEnabled
	case "disable_key":
		status = common.ChannelStatusAutoDisabled
		if reason == "" {
			reason = "hosting agent disabled key"
		}
	default:
		return nil, fmt.Errorf("unsupported multi-key action")
	}
	ok := model.UpdateChannelStatus(id, keys[index], status, reason)
	return map[string]any{"id": id, "index": index, "status": status, "updated": ok}, nil
}

func toolCreateChannel(ctx *ToolContext, args map[string]any) (any, error) {
	name := strings.TrimSpace(argString(args, "name"))
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	chType := argInt(args, "type")
	if chType == 0 {
		chType = constant.ChannelTypeOpenAI
	}
	key := strings.TrimSpace(argString(args, "key"))
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}
	models := strings.TrimSpace(argString(args, "models"))
	if models == "" {
		return nil, fmt.Errorf("models is required")
	}
	baseURL := strings.TrimSpace(argString(args, "base_url"))
	group := ResolveChannelGroups(argString(args, "group"), ctx.Agent.DefaultChannelGroups)
	channel := model.Channel{
		Name:        name,
		Type:        chType,
		Key:         key,
		Models:      models,
		Group:       group,
		Status:      common.ChannelStatusEnabled,
		CreatedTime: common.GetTimestamp(),
	}
	if baseURL != "" {
		channel.BaseURL = &baseURL
	}
	if err := channel.Insert(); err != nil {
		return nil, err
	}
	return map[string]any{"id": channel.Id, "name": channel.Name, "group": channel.Group}, nil
}

func toolUpdateChannelEndpoint(_ *ToolContext, args map[string]any) (any, error) {
	id := argInt(args, "id")
	channel, err := model.GetChannelById(id, true)
	if err != nil {
		return nil, err
	}
	if key := strings.TrimSpace(argString(args, "key")); key != "" {
		channel.Key = key
	}
	if baseURL := strings.TrimSpace(argString(args, "base_url")); baseURL != "" {
		channel.BaseURL = &baseURL
	}
	if err := channel.Update(); err != nil {
		return nil, err
	}
	return map[string]any{"id": channel.Id, "updated": true}, nil
}

func toolListUsers(_ *ToolContext, args map[string]any) (any, error) {
	page := argInt(args, "page")
	size := argInt(args, "page_size")
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	info := &common.PageInfo{Page: page, PageSize: size}
	users, total, err := model.GetAllUsers(info)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": users, "total": total, "page": page}, nil
}

func toolGetUser(_ *ToolContext, args map[string]any) (any, error) {
	id := argInt(args, "id")
	user, err := model.GetUserById(id, false)
	if err != nil {
		return nil, err
	}
	if user.IsAgentAccount() {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func toolListRedemptions(_ *ToolContext, args map[string]any) (any, error) {
	page := argInt(args, "page")
	size := argInt(args, "page_size")
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	items, total, err := model.GetAllRedemptions((page-1)*size, size)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items, "total": total}, nil
}

func toolListGroups(_ *ToolContext, _ map[string]any) (any, error) {
	names := make([]string, 0)
	for name := range ratio_setting.GetGroupRatioCopy() {
		names = append(names, name)
	}
	return names, nil
}

func toolListHooks(ctx *ToolContext, _ map[string]any) (any, error) {
	hooks, err := model.ListHostingHooks(ctx.Agent.Id)
	if err != nil {
		return nil, err
	}
	return hooks, nil
}

func toolCreateHook(ctx *ToolContext, args map[string]any) (any, error) {
	hook, err := CreateAgentHook(ctx.Agent, args)
	if err != nil {
		return nil, err
	}
	return hook, nil
}

func toolUpdateHook(ctx *ToolContext, args map[string]any) (any, error) {
	return UpdateAgentHook(ctx.Agent, argInt(args, "id"), args)
}

func toolDisableHook(ctx *ToolContext, args map[string]any) (any, error) {
	return UpdateAgentHook(ctx.Agent, argInt(args, "id"), map[string]any{"enabled": false})
}

func toolDeleteHook(ctx *ToolContext, args map[string]any) (any, error) {
	if err := DeleteAgentHook(ctx.Agent, argInt(args, "id")); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true}, nil
}

func toolHandoffIncident(ctx *ToolContext, args map[string]any) (any, error) {
	incident, err := Handoff(ctx.Agent, argString(args, "summary"), argString(args, "reason"), 0, "")
	if err != nil {
		return nil, err
	}
	return incident, nil
}

func toolSleepUntilHook(ctx *ToolContext, args map[string]any) (any, error) {
	ctx.Terminate = true
	return map[string]any{"sleeping": true, "note": argString(args, "note")}, nil
}
