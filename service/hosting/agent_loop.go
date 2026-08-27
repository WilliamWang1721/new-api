package hosting

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
)

const hostingSystemPrompt = "You are the New API intelligent hosting agent. Use only the provided tools. Never request or print secrets, payment keys, or root credentials. If a job is beyond granted tools or budgets, call handoff_incident. When the current wake is done, call sleep_until_hook. Do not disable, delete, or change the base URL of the current brain channel."

func RunAgentTurn(agent *model.HostingAgent, hookSummary string, hookId int, eventName string) {
	ctx := &ToolContext{
		Agent:       agent,
		UserID:      agent.UserId,
		Role:        common.RoleAdminUser,
		FromRunner:  true,
		IncidentKey: fmt.Sprintf("%d:%d:%s", agent.Id, hookId, eventName),
		ActorUserID: agent.UserId,
		ActorRole:   common.RoleAdminUser,
	}
	runAgentLoop(agent, hookSummary, hookId, eventName, ctx, hostingSystemPrompt)
}

func runAgentLoop(agent *model.HostingAgent, hookSummary string, hookId int, eventName string, ctx *ToolContext, systemPrompt string) {
	defer func() {
		if r := recover(); r != nil {
			common.SysError(fmt.Sprintf("hosting agent loop panic: %v", r))
			if ctx == nil || !ctx.Interactive {
				_, _ = Handoff(agent, hookSummary, fmt.Sprintf("runner panic: %v", r), hookId, eventName)
			}
		}
	}()
	if agent == nil || !agent.Enabled || ctx == nil {
		return
	}
	if overDailyBudget(agent) {
		if ctx.Interactive {
			return
		}
		_, _ = Handoff(agent, hookSummary, "daily token budget exceeded", hookId, eventName)
		return
	}
	if agent.SessionId == "" {
		agent.SessionId = model.NewHostingSessionID()
		if !ctx.Interactive {
			_ = model.DB.Model(agent).Update("session_id", agent.SessionId).Error
		}
	}

	_ = model.AppendHostingSessionEntry(&model.HostingSessionEntry{
		AgentId:    agent.Id,
		SessionId:  agent.SessionId,
		Role:       "user",
		Content:    hookSummary,
		TokenCount: EstimateTokens(hookSummary),
	})
	_ = model.AddHostingBrainUsage(agent.Id, 0, 0, 1, 0)

	messages := loadSessionMessages(agent)
	authz.MarkHostingAgentUser(agent.UserId)
	tools := brainTools(ctx)
	maxTurns := agent.MaxActionsPerIncident
	if maxTurns <= 0 {
		maxTurns = constant.DefaultHostingMaxActions
	}

	for turn := 0; turn < maxTurns+2; turn++ {
		if overDailyBudget(agent) {
			if !ctx.Interactive {
				_, _ = Handoff(agent, hookSummary, "daily token budget exceeded mid-loop", hookId, eventName)
			}
			return
		}
		extra := ContextExtra{
			Snapshot: SnapshotForPrompt(),
			Skill:    SkillForMessages(messages),
		}
		view := TransformContext(messages, extra)
		if NeedsCompaction(view, agent) {
			summary := compactWithBrain(agent, view)
			view = CompactMessages(agent, view, summary)
			persistCompaction(agent, summary)
		}
		req := BrainRequest{
			Model:    agent.BrainModel,
			Messages: prependSystemPrompt(view, systemPrompt),
			Tools:    tools,
		}
		resp, err := ChatWithBrain(agent, req)
		if LooksLikeContextOverflow(err, resp) {
			summary := compactWithBrain(agent, view)
			view = CompactMessages(agent, view, summary)
			persistCompaction(agent, summary)
			req.Messages = prependSystemPrompt(view, systemPrompt)
			resp, err = ChatWithBrain(agent, req)
			if LooksLikeContextOverflow(err, resp) {
				if !ctx.Interactive {
					_, _ = Handoff(agent, hookSummary, "context overflow after compaction", hookId, eventName)
				}
				return
			}
		}
		if err != nil {
			_ = model.AddHostingBrainUsage(agent.Id, 0, 0, 0, 1)
			if ctx.Interactive {
				_ = model.AppendHostingSessionEntry(&model.HostingSessionEntry{
					AgentId:    agent.Id,
					SessionId:  agent.SessionId,
					Role:       "assistant",
					Content:    "I could not reach the AI model. Check Steward Settings and try again.",
					TokenCount: 12,
				})
				return
			}
			reason := "brain call failed: " + err.Error()
			if agent.BrainSource == constant.HostingBrainDedicated {
				reason = "dedicated brain failed: " + err.Error()
			}
			_, _ = Handoff(agent, hookSummary, reason, hookId, eventName)
			return
		}
		if resp != nil {
			_ = model.AddHostingBrainUsage(agent.Id, resp.PromptTokens, resp.OutputTokens, 0, 0)
		}
		assistant := BrainMessage{Role: "assistant"}
		if resp != nil {
			assistant.Content = resp.Content
			assistant.ToolCalls = resp.ToolCalls
		}
		messages = append(messages, assistant)
		persistMessage(agent, assistant)

		if resp == nil || len(resp.ToolCalls) == 0 {
			if strings.TrimSpace(assistant.Content) != "" || ctx.Interactive {
				return
			}
			_, _ = Handoff(agent, hookSummary, "model returned no tool calls", hookId, eventName)
			return
		}

		for _, call := range resp.ToolCalls {
			args := map[string]any{}
			if strings.TrimSpace(call.Function.Arguments) != "" {
				_ = common.UnmarshalJsonStr(call.Function.Arguments, &args)
			}
			result, toolErr := ExecuteTool(ctx, call.Function.Name, args)
			content := ""
			if toolErr != nil {
				content = "error: " + toolErr.Error()
				if IsToolHandoff(toolErr) {
					if ctx.Interactive {
						content = "paused: " + toolErr.Error()
					} else {
						_, _ = Handoff(agent, hookSummary, toolErr.Error(), hookId, eventName)
						return
					}
				}
			} else {
				raw, _ := common.Marshal(result)
				content = string(raw)
			}
			toolMsg := BrainMessage{
				Role:       "tool",
				Name:       call.Function.Name,
				ToolCallID: call.ID,
				Content:    content,
			}
			messages = append(messages, toolMsg)
			persistMessage(agent, toolMsg)
			if ctx.Terminate {
				return
			}
		}
	}
	if !ctx.Interactive {
		_, _ = Handoff(agent, hookSummary, "action budget exhausted", hookId, eventName)
	}
}

func persistCompaction(agent *model.HostingAgent, summary string) {
	agent.LastCompactSummary = summary
	_ = model.DB.Model(agent).Updates(map[string]any{
		"last_compact_summary": summary,
		"updated_at":           common.GetTimestamp(),
	}).Error
	_ = model.AppendHostingSessionEntry(&model.HostingSessionEntry{
		AgentId:    agent.Id,
		SessionId:  agent.SessionId,
		Role:       compactRole,
		Content:    FormatCompactRecord(summary),
		TokenCount: EstimateTokens(summary),
	})
}

func loadSessionMessages(agent *model.HostingAgent) []BrainMessage {
	entries, err := model.ListHostingSessionEntries(agent.Id, agent.SessionId, 0, 400)
	if err != nil {
		return nil
	}
	out := make([]BrainMessage, 0, len(entries))
	for _, entry := range entries {
		if entry.Role == compactRole {
			out = append(out, BrainMessage{Role: "user", Content: entry.Content})
			continue
		}
		out = append(out, BrainMessage{
			Role:       entry.Role,
			Name:       entry.Name,
			Content:    entry.Content,
			ToolCallID: entry.ToolCallID,
		})
	}
	return out
}

func persistMessage(agent *model.HostingAgent, msg BrainMessage) {
	content := msg.Content
	if len(msg.ToolCalls) > 0 {
		raw, _ := common.Marshal(msg.ToolCalls)
		content = strings.TrimSpace(content + "\n" + string(raw))
	}
	_ = model.AppendHostingSessionEntry(&model.HostingSessionEntry{
		AgentId:    agent.Id,
		SessionId:  agent.SessionId,
		Role:       msg.Role,
		Name:       msg.Name,
		Content:    content,
		ToolCallID: msg.ToolCallID,
		TokenCount: EstimateTokens(content),
	})
}

func prependSystem(messages []BrainMessage) []BrainMessage {
	return prependSystemPrompt(messages, hostingSystemPrompt)
}

func prependSystemPrompt(messages []BrainMessage, prompt string) []BrainMessage {
	if strings.TrimSpace(prompt) == "" {
		prompt = hostingSystemPrompt
	}
	out := make([]BrainMessage, 0, len(messages)+1)
	out = append(out, BrainMessage{Role: "system", Content: prompt})
	out = append(out, messages...)
	return out
}

func brainTools(ctx *ToolContext) []BrainTool {
	listed := ListToolsForAgent(ctx)
	out := make([]BrainTool, 0, len(listed))
	for _, item := range listed {
		name, _ := item["name"].(string)
		desc, _ := item["description"].(string)
		schema, _ := item["input_schema"].(map[string]any)
		out = append(out, BrainTool{
			Type: "function",
			Function: BrainToolFn{
				Name:        name,
				Description: desc,
				Parameters:  schema,
			},
		})
	}
	return out
}

func compactWithBrain(agent *model.HostingAgent, messages []BrainMessage) string {
	older, _ := SplitForCompaction(messages, agent.KeepRecentTokens)
	fallback := ExtractiveSummary(older)
	req := BrainRequest{
		Model:          agent.BrainModel,
		PromptCacheOff: true,
		Messages: []BrainMessage{
			{Role: "system", Content: "Summarize this operations transcript into Goal, Progress, Decisions, Next Steps, and Critical Context. Do not invent facts."},
			{Role: "user", Content: fallback},
		},
	}
	maxTokens := 800
	req.MaxTokens = &maxTokens
	resp, err := ChatWithBrain(agent, req)
	if err != nil || resp == nil || strings.TrimSpace(resp.Content) == "" {
		return fallback
	}
	_ = model.AddHostingBrainUsage(agent.Id, resp.PromptTokens, resp.OutputTokens, 0, 0)
	return resp.Content
}

func SessionTimeline(agentId int, sessionId string) ([]*model.HostingSessionEntry, error) {
	if sessionId == "" {
		agent, err := model.GetHostingAgentById(agentId)
		if err != nil {
			return nil, err
		}
		sessionId = agent.SessionId
	}
	return model.ListHostingSessionEntries(agentId, sessionId, 0, 500)
}

func ExportSession(agentId int, sessionId string) (string, error) {
	entries, err := SessionTimeline(agentId, sessionId)
	if err != nil {
		return "", err
	}
	raw, err := common.Marshal(entries)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func IsToolHandoff(err error) bool {
	var target *ToolHandoffError
	return errors.As(err, &target)
}
