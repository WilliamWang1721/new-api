package hosting

import (
	"fmt"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

var (
	wakeMu      sync.Mutex
	lastWakeAt  = map[int]time.Time{}
	hourlyWakes = map[int][]time.Time{}
	followups   = map[int][]string{}
	runningWake = map[int]bool{}
	agentLocks  sync.Map
	wakeSlots   = make(chan struct{}, constant.HostingMaxConcurrentWakes)
)

func agentLock(id int) *sync.Mutex {
	v, _ := agentLocks.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func seedSystemHooks() error {
	specs := []model.HostingHook{
		{
			Owner:       constant.HostingHookOwnerSystem,
			Name:        "Channel auto-disabled",
			Enabled:     true,
			Kind:        constant.HostingHookKindEvent,
			EventName:   constant.HostingEventChannelDisabled,
			WakeMode:    constant.HostingWakeAI,
			CooldownSec: 120,
			MergeKey:    "channel.auto_disabled",
			PromptHint:  "A channel was auto-disabled. Diagnose within granted tools, repair if possible, then create a follow-up hook or hand off.",
			SystemKey:   "sys_channel_auto_disabled",
		},
		{
			Owner:       constant.HostingHookOwnerSystem,
			Name:        "Quota exhausted",
			Enabled:     true,
			Kind:        constant.HostingHookKindEvent,
			EventName:   constant.HostingEventQuota,
			WakeMode:    constant.HostingWakeNotifyOnly,
			CooldownSec: 300,
			MergeKey:    "quota.exhausted",
			PromptHint:  "A quota alarm fired. Do not wake the model; notify the handoff contact.",
			SystemKey:   "sys_quota_exhausted",
		},
		{
			Owner:       constant.HostingHookOwnerSystem,
			Name:        "Channel test failed",
			Enabled:     true,
			Kind:        constant.HostingHookKindEvent,
			EventName:   constant.HostingEventChannelTest,
			WakeMode:    constant.HostingWakeNotifyOnly,
			CooldownSec: 300,
			MergeKey:    "channel.test_failed",
			PromptHint:  "A scheduled channel test failed. Inspect within granted tools or hand off.",
			SystemKey:   "sys_channel_test_failed",
		},
	}
	now := common.GetTimestamp()
	for _, spec := range specs {
		if _, err := model.GetSystemHookByKey(spec.SystemKey); err == nil {
			continue
		}
		spec.CreatedAt = now
		spec.UpdatedAt = now
		if err := spec.Insert(); err != nil {
			return err
		}
	}
	return nil
}

func HandleHostingEvent(event service.HostingEvent) {
	if !IsReady() {
		return
	}
	hooks, err := model.ListEnabledHostingHooks()
	if err != nil {
		return
	}
	for _, hook := range hooks {
		if hook.Kind != constant.HostingHookKindEvent {
			continue
		}
		if hook.EventName != event.Name {
			continue
		}
		if hook.ChannelId > 0 && event.ChannelId > 0 && hook.ChannelId != event.ChannelId {
			continue
		}
		fireHook(hook, event)
	}
}

func EvaluateHooks() {
	if !IsReady() {
		return
	}
	hooks, err := model.ListEnabledHostingHooks()
	if err != nil {
		return
	}
	now := common.GetTimestamp()
	for _, hook := range hooks {
		switch hook.Kind {
		case constant.HostingHookKindSchedule:
			if hook.NextFireAt > 0 && hook.NextFireAt <= now {
				fireHook(hook, service.HostingEvent{Name: "schedule", Reason: hook.Name})
			}
		case constant.HostingHookKindCondition:
			if matchCondition(hook) {
				fireHook(hook, service.HostingEvent{Name: "condition:" + hook.Condition, Reason: hook.Name})
			}
		}
	}
}

func matchCondition(hook *model.HostingHook) bool {
	n := hook.ConditionN
	if n <= 0 {
		n = 1
	}
	switch hook.Condition {
	case "auto_disabled_count":
		var count int64
		_ = model.DB.Model(&model.Channel{}).Where("status = ?", common.ChannelStatusAutoDisabled).Count(&count).Error
		return int(count) >= n
	case "error_logs_in_window":
		if model.LOG_DB == nil {
			return false
		}
		var count int64
		window := int64(hook.CooldownSec)
		if window <= 0 {
			window = 300
		}
		_ = model.LOG_DB.Model(&model.Log{}).Where("type = ? AND created_at >= ?", model.LogTypeError, common.GetTimestamp()-window).Count(&count).Error
		return int(count) >= n
	case "host_alloc_mb":
		return hostAllocMB() >= n
	default:
		return false
	}
}

func fireHook(hook *model.HostingHook, event service.HostingEvent) {
	if hook.CooldownSec > 0 && hook.LastFiredAt > 0 && common.GetTimestamp()-hook.LastFiredAt < int64(hook.CooldownSec) {
		return
	}
	if hook.MaxFires > 0 && hook.FireCount >= hook.MaxFires {
		return
	}
	hook.LastFiredAt = common.GetTimestamp()
	hook.FireCount++
	if hook.Kind == constant.HostingHookKindSchedule {
		hook.NextFireAt = nextFireAt(hook)
		if hook.FireAt > 0 && hook.Cron == "" {
			hook.Enabled = false
		}
	}
	_ = hook.Update()

	agents := targetAgents(hook)
	summary := fmt.Sprintf("hook=%s event=%s channel=%d reason=%s hint=%s", hook.Name, event.Name, event.ChannelId, event.Reason, hook.PromptHint)
	for _, agent := range agents {
		dispatchHook(agent, hook, event, summary)
	}
}

func targetAgents(hook *model.HostingHook) []*model.HostingAgent {
	if hook.Owner == constant.HostingHookOwnerAgent && hook.AgentId > 0 {
		agent, err := model.GetHostingAgentById(hook.AgentId)
		if err != nil || !agent.Enabled {
			return nil
		}
		return []*model.HostingAgent{agent}
	}
	agents, err := model.ListHostingAgents()
	if err != nil {
		return nil
	}
	out := make([]*model.HostingAgent, 0)
	for _, agent := range agents {
		if agent.Enabled {
			out = append(out, agent)
		}
	}
	return out
}

func dispatchHook(agent *model.HostingAgent, hook *model.HostingHook, event service.HostingEvent, summary string) {
	lock := agentLock(agent.Id)
	lock.Lock()
	defer lock.Unlock()

	resolved, actions := RunPlaybook(agent, hook, event)
	if resolved {
		_, _ = CreateIncident(agent, constant.HostingIncidentAutoResolved, hook.Id, event.Name, "Playbook resolved: "+summary, strings.Join(actions, "; "))
		return
	}
	if hook.WakeMode == constant.HostingWakeNotifyOnly || hook.WakeMode == constant.HostingWakePlaybookOnly {
		_, _ = Handoff(agent, summary, "hook="+hook.WakeMode, hook.Id, event.Name)
		return
	}

	if !tryAcquireWakeSlot() {
		wakeMu.Lock()
		followups[agent.Id] = append(followups[agent.Id], summary)
		wakeMu.Unlock()
		return
	}

	wakeMu.Lock()
	if runningWake[agent.Id] {
		followups[agent.Id] = append(followups[agent.Id], summary)
		wakeMu.Unlock()
		releaseWakeSlot()
		return
	}
	now := time.Now()
	if last := lastWakeAt[agent.Id]; !last.IsZero() && now.Sub(last) < time.Duration(agent.WakeMergeWindowSec)*time.Second {
		followups[agent.Id] = append(followups[agent.Id], summary)
		wakeMu.Unlock()
		releaseWakeSlot()
		return
	}
	if !allowHourlyWake(agent, now) {
		wakeMu.Unlock()
		releaseWakeSlot()
		_, _ = Handoff(agent, summary, "hourly wake budget exceeded", hook.Id, event.Name)
		return
	}
	if overDailyBudget(agent) {
		wakeMu.Unlock()
		releaseWakeSlot()
		_, _ = Handoff(agent, summary, "daily token budget exceeded", hook.Id, event.Name)
		return
	}
	runningWake[agent.Id] = true
	lastWakeAt[agent.Id] = now
	queued := followups[agent.Id]
	followups[agent.Id] = nil
	wakeMu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				common.SysError(fmt.Sprintf("hosting wake panic agent=%d: %v", agent.Id, r))
			}
			wakeMu.Lock()
			runningWake[agent.Id] = false
			wakeMu.Unlock()
			releaseWakeSlot()
		}()
		prompt := summary
		if len(queued) > 0 {
			prompt += "\n" + strings.Join(queued, "\n")
		}
		RunAgentTurn(agent, prompt, hook.Id, event.Name)
		for IsReady() {
			wakeMu.Lock()
			more := followups[agent.Id]
			followups[agent.Id] = nil
			wakeMu.Unlock()
			if len(more) == 0 {
				return
			}
			fresh, err := model.GetHostingAgentById(agent.Id)
			if err != nil {
				return
			}
			RunAgentTurn(fresh, strings.Join(more, "\n"), hook.Id, event.Name)
		}
	}()
}

func allowHourlyWake(agent *model.HostingAgent, now time.Time) bool {
	cutoff := now.Add(-time.Hour)
	kept := hourlyWakes[agent.Id][:0]
	for _, ts := range hourlyWakes[agent.Id] {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= agent.MaxWakesPerHour {
		hourlyWakes[agent.Id] = kept
		return false
	}
	hourlyWakes[agent.Id] = append(kept, now)
	return true
}

func overDailyBudget(agent *model.HostingAgent) bool {
	used := model.HostingUsageDayTotal(agent.Id, time.Now().Format("2006-01-02"))
	return used >= agent.DailyTokenBudget
}

func nextFireAt(hook *model.HostingHook) int64 {
	if strings.TrimSpace(hook.Cron) == "" {
		return 0
	}
	next, err := NextCronTime(hook.Cron, time.Now().Add(time.Minute))
	if err != nil {
		return 0
	}
	return next.Unix()
}

func CreateAgentHook(agent *model.HostingAgent, args map[string]any) (*model.HostingHook, error) {
	if agent == nil || !agent.AllowAgentHooks {
		return nil, fmt.Errorf("agent hooks are disabled")
	}
	n, err := model.CountAgentHooks(agent.Id)
	if err != nil {
		return nil, err
	}
	if int(n) >= agent.MaxAgentHooks {
		return nil, fmt.Errorf("agent hook limit reached")
	}
	kind := strings.TrimSpace(argString(args, "kind"))
	switch kind {
	case constant.HostingHookKindEvent, constant.HostingHookKindSchedule, constant.HostingHookKindCondition:
	default:
		return nil, fmt.Errorf("unsupported hook kind")
	}
	cooldown := argInt(args, "cooldown_sec")
	if cooldown > 0 && cooldown < agent.MinHookIntervalSec {
		return nil, fmt.Errorf("cooldown is below the minimum interval of %d seconds", agent.MinHookIntervalSec)
	}
	if cooldown <= 0 {
		cooldown = agent.MinHookIntervalSec
	}
	fireAt := int64(argInt(args, "fire_at"))
	cron := strings.TrimSpace(argString(args, "cron"))
	if kind == constant.HostingHookKindSchedule {
		if fireAt > 0 {
			if fireAt-common.GetTimestamp() < int64(agent.MinHookIntervalSec) {
				return nil, fmt.Errorf("one-shot hooks must be at least %d seconds in the future", agent.MinHookIntervalSec)
			}
		} else if cron != "" {
			if err := ValidateAgentCron(cron, agent.MinHookIntervalSec); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("schedule hooks require fire_at or cron")
		}
	}
	if kind == constant.HostingHookKindCondition {
		cond := strings.TrimSpace(argString(args, "condition"))
		if cond != "auto_disabled_count" && cond != "error_logs_in_window" && cond != "host_alloc_mb" {
			return nil, fmt.Errorf("condition is not in the allowlist")
		}
	}
	wake := strings.TrimSpace(argString(args, "wake_mode"))
	if wake == "" {
		wake = constant.HostingWakeAI
	}
	hook := &model.HostingHook{
		AgentId:     agent.Id,
		Owner:       constant.HostingHookOwnerAgent,
		Name:        strings.TrimSpace(argString(args, "name")),
		Enabled:     true,
		Kind:        kind,
		EventName:   strings.TrimSpace(argString(args, "event_name")),
		Cron:        cron,
		FireAt:      fireAt,
		Condition:   strings.TrimSpace(argString(args, "condition")),
		ConditionN:  argInt(args, "condition_n"),
		ChannelId:   argInt(args, "channel_id"),
		WakeMode:    wake,
		CooldownSec: cooldown,
		PromptHint:  argString(args, "prompt_hint"),
	}
	if fireAt > 0 {
		hook.NextFireAt = fireAt
	} else if cron != "" {
		hook.NextFireAt = nextFireAt(hook)
	}
	if err := hook.Insert(); err != nil {
		return nil, err
	}
	auditHostingAction(agent.UserId, "", constant.HostingAuthMethod, "hosting.hook.create", map[string]any{
		"hook_id": hook.Id,
		"name":    hook.Name,
		"kind":    hook.Kind,
	})
	return hook, nil
}

func UpdateAgentHook(agent *model.HostingAgent, id int, args map[string]any) (*model.HostingHook, error) {
	hook, err := model.GetHostingHookById(id)
	if err != nil {
		return nil, err
	}
	if hook.Owner != constant.HostingHookOwnerAgent || hook.AgentId != agent.Id {
		return nil, fmt.Errorf("cannot modify this hook")
	}
	updates := map[string]any{"updated_at": common.GetTimestamp()}
	if enabled, ok := argBool(args, "enabled"); ok {
		updates["enabled"] = enabled
	}
	if wake := argString(args, "wake_mode"); wake != "" {
		updates["wake_mode"] = wake
	}
	if cooldown := argInt(args, "cooldown_sec"); cooldown > 0 {
		if cooldown < agent.MinHookIntervalSec {
			return nil, fmt.Errorf("cooldown is below the minimum interval")
		}
		updates["cooldown_sec"] = cooldown
	}
	if hint := argString(args, "prompt_hint"); hint != "" {
		updates["prompt_hint"] = hint
	}
	if fireAt := int64(argInt(args, "fire_at")); fireAt > 0 {
		if fireAt-common.GetTimestamp() < int64(agent.MinHookIntervalSec) {
			return nil, fmt.Errorf("one-shot hooks must be at least %d seconds in the future", agent.MinHookIntervalSec)
		}
		updates["fire_at"] = fireAt
		updates["next_fire_at"] = fireAt
	}
	if err := model.DB.Model(hook).Updates(updates).Error; err != nil {
		return nil, err
	}
	auditHostingAction(agent.UserId, "", constant.HostingAuthMethod, "hosting.hook.update", map[string]any{
		"hook_id": id,
	})
	return model.GetHostingHookById(id)
}

func DeleteAgentHook(agent *model.HostingAgent, id int) error {
	hook, err := model.GetHostingHookById(id)
	if err != nil {
		return err
	}
	if hook.Owner != constant.HostingHookOwnerAgent || hook.AgentId != agent.Id {
		return fmt.Errorf("cannot delete this hook")
	}
	if err := model.DeleteHostingHook(id); err != nil {
		return err
	}
	auditHostingAction(agent.UserId, "", constant.HostingAuthMethod, "hosting.hook.delete", map[string]any{
		"hook_id": id,
	})
	return nil
}

func AdminUpdateHook(id int, enabled *bool, wakeMode string) (*model.HostingHook, error) {
	hook, err := model.GetHostingHookById(id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": common.GetTimestamp()}
	if enabled != nil {
		updates["enabled"] = *enabled
	}
	if wakeMode != "" {
		updates["wake_mode"] = wakeMode
	}
	if err := model.DB.Model(hook).Updates(updates).Error; err != nil {
		return nil, err
	}
	return model.GetHostingHookById(id)
}

func AdminDeleteHook(id int) error {
	hook, err := model.GetHostingHookById(id)
	if err != nil {
		return err
	}
	if hook.Owner == constant.HostingHookOwnerSystem {
		return fmt.Errorf("system hooks cannot be deleted")
	}
	return model.DeleteHostingHook(id)
}

func tryAcquireWakeSlot() bool {
	select {
	case wakeSlots <- struct{}{}:
		return true
	default:
		return false
	}
}

func releaseWakeSlot() {
	select {
	case <-wakeSlots:
	default:
	}
}

func hostAllocMB() int {
	var ms goruntime.MemStats
	goruntime.ReadMemStats(&ms)
	return int(ms.Alloc / (1024 * 1024))
}

func auditHostingAction(userId int, ip, authMethod, action string, params map[string]any) {
	if params == nil {
		params = map[string]any{}
	}
	adminInfo := map[string]any{
		"admin_id":    userId,
		"auth_method": authMethod,
	}
	model.RecordOperationAuditLog(userId, action, ip, action, params, adminInfo, params)
}
