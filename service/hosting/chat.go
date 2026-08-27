package hosting

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
)

const stewardChatPrompt = "You are the New API AI Steward. Talk with the operator in clear, non-technical language. Use only the provided tools to inspect or change this API gateway, including system settings. Start by understanding what they want, then use get_setup_checklist or list_system_settings when they are setting the site up. Never request or print secrets, payment keys, or root credentials. If a tool returns pending_approval, tell the user it is waiting for confirmation and do not pretend the change already happened. When you cannot do something, say so plainly. Do not disable, delete, or change the base URL of the current brain channel."

type ChatTurnResult struct {
	SessionId  string                       `json:"session_id"`
	Entries    []*model.HostingSessionEntry `json:"entries"`
	Pending    []*model.HostingApproval     `json:"pending_approvals"`
	NeedsBrain bool                         `json:"needs_brain"`
	Reply      string                       `json:"reply"`
}

var chatLocks sync.Map

func lockAgentChat(agentId int) func() {
	v, _ := chatLocks.LoadOrStore(agentId, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func RunChatTurn(agent *model.HostingAgent, actorUserId, actorRole int, message, clientIP string) (*ChatTurnResult, error) {
	if agent == nil || !agent.Enabled {
		return nil, fmt.Errorf("the steward is turned off")
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, fmt.Errorf("message is required")
	}
	if utf8.RuneCountInString(message) > constant.MaxHostingChatMessageRunes {
		return nil, fmt.Errorf("message is too long")
	}
	if !BrainIsConfigured(agent) {
		return &ChatTurnResult{NeedsBrain: true, Reply: "Set up an AI model in Steward Settings first."}, nil
	}
	unlock := lockAgentChat(agent.Id)
	defer unlock()

	chatSession, err := model.GetOrCreateHostingChatSession(agent.Id, actorUserId)
	if err != nil {
		return nil, err
	}
	sessionId := chatSession.SessionId

	live := *agent
	live.SessionId = sessionId
	ctx := &ToolContext{
		Agent:       &live,
		UserID:      agent.UserId,
		Role:        common.RoleAdminUser,
		ClientIP:    clientIP,
		FromRunner:  true,
		Interactive: true,
		IncidentKey: fmt.Sprintf("chat:%d:%d", agent.Id, actorUserId),
		ActorUserID: actorUserId,
		ActorRole:   actorRole,
	}
	authz.MarkHostingAgentUser(agent.UserId)
	runAgentLoop(&live, message, 0, "steward.chat", ctx, stewardChatPrompt)

	entries, err := model.ListHostingSessionEntries(agent.Id, sessionId, 0, 400)
	if err != nil {
		return nil, err
	}
	pending, _ := ListApprovalsForActor(agent.Id, actorUserId, actorRole, constant.HostingApprovalPending)
	return &ChatTurnResult{
		SessionId: sessionId,
		Entries:   entries,
		Pending:   pending,
		Reply:     latestAssistantText(entries),
	}, nil
}

func latestAssistantText(entries []*model.HostingSessionEntry) string {
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if entry.Role == "assistant" && strings.TrimSpace(entry.Content) != "" {
			return entry.Content
		}
	}
	return ""
}

func StewardChatSession(agentId, userId int) (*model.HostingChatSession, []*model.HostingSessionEntry, error) {
	row, err := model.GetOrCreateHostingChatSession(agentId, userId)
	if err != nil {
		return nil, nil, err
	}
	entries, err := model.ListHostingSessionEntries(agentId, row.SessionId, 0, 400)
	return row, entries, err
}
