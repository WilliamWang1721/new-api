package hosting

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/model"
)

const (
	maxToolResultChars = 8000
	compactRole        = "compaction"
)

func EstimateTokens(text string) int {
	n := utf8.RuneCountInString(text)
	if n <= 0 {
		return 0
	}
	tokens := (n + 3) / 4
	if tokens < 1 {
		return 1
	}
	return tokens
}

func TransformContext(messages []BrainMessage) []BrainMessage {
	out := make([]BrainMessage, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == "tool" && utf8.RuneCountInString(msg.Content) > maxToolResultChars {
			msg.Content = truncateRunes(msg.Content, maxToolResultChars) + "\n[truncated tool result]"
		}
		out = append(out, msg)
	}
	return out
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}

func ContextTokenCount(messages []BrainMessage) int {
	total := 0
	for _, msg := range messages {
		total += EstimateTokens(msg.Content)
		for _, call := range msg.ToolCalls {
			total += EstimateTokens(call.Function.Name) + EstimateTokens(call.Function.Arguments)
		}
	}
	return total
}

func NeedsCompaction(messages []BrainMessage, agent *model.HostingAgent) bool {
	if agent == nil {
		return false
	}
	limit := agent.ContextWindow - agent.ReserveTokens
	if limit <= 0 {
		return false
	}
	return ContextTokenCount(messages) > limit
}

func SplitForCompaction(messages []BrainMessage, keepRecentTokens int) (older []BrainMessage, recent []BrainMessage) {
	if keepRecentTokens <= 0 {
		keepRecentTokens = 20000
	}
	used := 0
	cut := len(messages)
	for i := len(messages) - 1; i >= 0; i-- {
		used += EstimateTokens(messages[i].Content)
		if used > keepRecentTokens && i < len(messages)-1 {
			cut = i + 1
			break
		}
	}
	for cut < len(messages) && messages[cut].Role == "tool" {
		cut++
	}
	if cut <= 0 || cut >= len(messages) {
		if len(messages) > 2 {
			cut = len(messages) - 1
		} else {
			return nil, messages
		}
	}
	return messages[:cut], messages[cut:]
}

func ExtractiveSummary(older []BrainMessage) string {
	var b strings.Builder
	b.WriteString("Goal: Keep operating New API within granted hosting tools.\n")
	b.WriteString("Progress:\n")
	for _, msg := range older {
		if msg.Role == "assistant" && msg.Content != "" {
			b.WriteString("- ")
			b.WriteString(truncateRunes(strings.ReplaceAll(msg.Content, "\n", " "), 240))
			b.WriteString("\n")
		}
		if msg.Role == "tool" {
			b.WriteString("- tool ")
			b.WriteString(msg.Name)
			b.WriteString(": ")
			b.WriteString(truncateRunes(strings.ReplaceAll(msg.Content, "\n", " "), 160))
			b.WriteString("\n")
		}
	}
	b.WriteString("Decisions: Prefer playbooks and granted channel/log tools; hand off when blocked.\n")
	b.WriteString("Next Steps: Continue from the latest hook summary.\n")
	b.WriteString("Critical Context: Channels, hooks, and incidents already acted on stay in this summary.\n")
	return b.String()
}

func CompactMessages(agent *model.HostingAgent, messages []BrainMessage, summary string) []BrainMessage {
	older, recent := SplitForCompaction(messages, agent.KeepRecentTokens)
	if len(older) == 0 {
		return messages
	}
	if strings.TrimSpace(summary) == "" {
		summary = ExtractiveSummary(older)
	}
	compacted := []BrainMessage{{
		Role:    "user",
		Content: "[COMPACTION]\n" + summary,
	}}
	return append(compacted, recent...)
}

func LooksLikeContextOverflow(err error, resp *BrainResponse) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if resp != nil {
		msg += " " + strings.ToLower(resp.RawError)
	}
	return strings.Contains(msg, "context") && (strings.Contains(msg, "length") || strings.Contains(msg, "too long") || strings.Contains(msg, "overflow") || strings.Contains(msg, "maximum"))
}

func FormatCompactRecord(summary string) string {
	return fmt.Sprintf("compacted:\n%s", summary)
}
