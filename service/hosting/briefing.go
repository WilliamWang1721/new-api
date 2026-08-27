package hosting

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

type BriefingResult struct {
	Text      string `json:"text"`
	Generated bool   `json:"generated"`
	Mode      string `json:"mode"`
	Cached    bool   `json:"cached"`
}

func BriefingIsDue(mode string, lastAt, now int64, force bool) bool {
	mode = NormalizeBriefingMode(mode)
	if mode == constant.HostingBriefingOff {
		return false
	}
	if force || lastAt <= 0 {
		return true
	}
	if now < lastAt {
		return true
	}
	switch mode {
	case constant.HostingBriefingDaily:
		return now-lastAt >= constant.HostingBriefingDailySec
	default:
		return now-lastAt >= constant.HostingBriefingEveryOpenSec
	}
}

func fallbackBriefing(agent *model.HostingAgent) string {
	pending := int64(0)
	if agent != nil {
		pending, _ = model.CountPendingHostingApprovals(agent.Id)
	}
	disabled := 0
	snap := RuntimeSnapshot()
	if raw, ok := snap["auto_disabled_count"]; ok {
		switch n := raw.(type) {
		case int:
			disabled = n
		case int64:
			disabled = int(n)
		case float64:
			disabled = int(n)
		}
	}
	parts := []string{"Here is the current status:"}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d request(s) are waiting for your confirmation.", pending))
	} else {
		parts = append(parts, "There are no pending approval requests.")
	}
	if disabled > 0 {
		parts = append(parts, fmt.Sprintf("%d channel(s) were auto-disabled.", disabled))
	} else {
		parts = append(parts, "No channels are in the auto-disabled list.")
	}
	if agent != nil && agent.DryRun {
		parts = append(parts, "Practice mode is on, so the steward will not apply real changes.")
	}
	if agent != nil && !BrainIsConfigured(agent) {
		parts = append(parts, "Choose an AI model in Steward Settings to get a fuller briefing next time.")
	}
	return strings.Join(parts, " ")
}

func GenerateStewardBriefing(agent *model.HostingAgent, userId int, force bool) (*BriefingResult, error) {
	if agent == nil {
		return nil, fmt.Errorf("the steward is not ready yet")
	}
	mode := NormalizeBriefingMode(agent.BriefingMode)
	row, err := model.GetOrCreateHostingChatSession(agent.Id, userId)
	if err != nil {
		return nil, err
	}
	if mode == constant.HostingBriefingOff {
		return &BriefingResult{Mode: mode, Text: ""}, nil
	}
	now := common.GetTimestamp()
	if !BriefingIsDue(mode, row.LastBriefingAt, now, force) {
		return &BriefingResult{Mode: mode, Text: row.LastBriefingText, Cached: true}, nil
	}
	text := fallbackBriefing(agent)
	generated := false
	if BrainIsConfigured(agent) && !overDailyBudget(agent) {
		if aiText := generateAIBriefing(agent, text); aiText != "" {
			text = aiText
			generated = true
		}
	}
	if err := model.SaveHostingChatBriefing(agent.Id, userId, text); err != nil {
		return nil, err
	}
	return &BriefingResult{Mode: mode, Text: text, Generated: generated}, nil
}

func generateAIBriefing(agent *model.HostingAgent, fallback string) string {
	maxTokens := 400
	req := BrainRequest{
		Model: agent.BrainModel,
		Messages: []BrainMessage{
			{
				Role:    "system",
				Content: "You are the New API AI Steward. Write a short status briefing in plain language. Use short sentences. Do not mention tools, tokens, or implementation details. Do not invent incidents. If the facts are calm, say so.",
			},
			{
				Role:    "user",
				Content: "Facts:\n" + fallback + "\n\nRuntime:\n" + SnapshotForPrompt() + "\n\nWrite the briefing now.",
			},
		},
		MaxTokens:      &maxTokens,
		PromptCacheOff: true,
	}
	resp, err := ChatWithBrain(agent, req)
	if err != nil || resp == nil {
		return ""
	}
	_ = model.AddHostingBrainUsage(agent.Id, resp.PromptTokens, resp.OutputTokens, 0, 0)
	text := strings.TrimSpace(resp.Content)
	if text == "" {
		return ""
	}
	if utf8.RuneCountInString(text) > 2000 {
		runes := []rune(text)
		text = string(runes[:2000])
	}
	return text
}
