package hosting

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
)

func CreateIncident(agent *model.HostingAgent, status string, hookId int, event, summary, actions string) (*model.HostingIncident, error) {
	item := &model.HostingIncident{
		AgentId:      agent.Id,
		Status:       status,
		SourceHookId: hookId,
		SourceEvent:  event,
		Summary:      summary,
		ActionsJSON:  actions,
		BrainSource:  agent.BrainSource,
	}
	if agent.HandoffUserId > 0 {
		item.AssigneeUserId = agent.HandoffUserId
	}
	if err := item.Insert(); err != nil {
		return nil, err
	}
	return item, nil
}

func Handoff(agent *model.HostingAgent, summary, reason string, hookId int, event string) (*model.HostingIncident, error) {
	if summary == "" {
		summary = "Hosting agent requested a human handoff"
	}
	if reason == "" {
		reason = "unspecified"
	}
	if agent != nil && agent.BrainSource == constant.HostingBrainDedicated && !strings.Contains(reason, "dedicated") {
		reason = "dedicated: " + reason
	}
	item, err := CreateIncident(agent, constant.HostingIncidentHandedOff, hookId, event, summary, "")
	if err != nil {
		return nil, err
	}
	item.HandoffReason = reason
	item.BrainSource = agent.BrainSource
	_ = item.Update()
	notifyHandoff(agent, summary, reason)
	return item, nil
}

func ResolveIncident(id int, status string) (*model.HostingIncident, error) {
	item, err := model.GetHostingIncidentById(id)
	if err != nil {
		return nil, err
	}
	switch status {
	case constant.HostingIncidentIgnored, constant.HostingIncidentHandedOff, constant.HostingIncidentAutoResolved:
		item.Status = status
	default:
		return nil, fmt.Errorf("unsupported incident status")
	}
	if err := item.Update(); err != nil {
		return nil, err
	}
	return item, nil
}

func notifyHandoff(agent *model.HostingAgent, summary, reason string) {
	defer func() { _ = recover() }()
	if agent == nil || model.DB == nil {
		return
	}
	subject := fmt.Sprintf("Hosting handoff: %s", agent.Name)
	content := fmt.Sprintf("Agent %s (#%d) handed off an incident.\nBrain: %s\nReason: %s\nSummary: %s", agent.Name, agent.Id, agent.BrainSource, reason, summary)
	userId := agent.HandoffUserId
	if userId <= 0 {
		root := model.GetRootUser()
		if root == nil || root.Id <= 0 {
			common.SysLog("hosting handoff: no root user to notify")
			return
		}
		service.NotifyRootUser(dto.NotifyTypeHostingHandoff, subject, content)
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		root := model.GetRootUser()
		if root == nil || root.Id <= 0 {
			common.SysLog("hosting handoff: assignee missing and no root user to notify")
			return
		}
		service.NotifyRootUser(dto.NotifyTypeHostingHandoff, subject, content)
		return
	}
	err = service.NotifyUser(user.Id, user.Email, user.GetSetting(), dto.NewNotify(dto.NotifyTypeHostingHandoff, subject, content, nil))
	if err != nil {
		common.SysLog("hosting handoff notify failed: " + err.Error())
	}
}

func CostSnapshot(agentId int) map[string]any {
	day := time.Now().Format("2006-01-02")
	used := 0
	wakes := 0
	if agentId > 0 {
		if row, err := model.GetHostingBrainUsage(agentId, day); err == nil {
			used = row.PromptTokens + row.OutputTokens
			wakes = row.WakeCount
		}
	}
	return map[string]any{
		"day":         day,
		"tokens_used": used,
		"wakes":       wakes,
		"agent_id":    agentId,
	}
}
