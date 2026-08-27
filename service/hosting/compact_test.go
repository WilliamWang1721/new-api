package hosting

import (
	"errors"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateTokens(t *testing.T) {
	assert.Greater(t, EstimateTokens("hello world"), 0)
	assert.Equal(t, 0, EstimateTokens(""))
}

func TestTransformContextTruncatesToolResults(t *testing.T) {
	big := strings.Repeat("x", maxToolResultChars+50)
	out := TransformContext([]BrainMessage{{Role: "tool", Content: big}})
	require.Len(t, out, 1)
	assert.Less(t, len([]rune(out[0].Content)), maxToolResultChars+40)
	assert.Contains(t, out[0].Content, "truncated")
}

func TestSplitForCompactionKeepsRecentAndSkipsToolCut(t *testing.T) {
	messages := []BrainMessage{
		{Role: "user", Content: strings.Repeat("old ", 200)},
		{Role: "assistant", Content: "worked on channel 1"},
		{Role: "tool", Name: "list_channels", Content: "result"},
		{Role: "user", Content: "latest hook"},
	}
	older, recent := SplitForCompaction(messages, 8)
	assert.NotEmpty(t, recent)
	if len(recent) > 0 {
		assert.NotEqual(t, "tool", recent[0].Role)
	}
	summary := ExtractiveSummary(older)
	assert.Contains(t, summary, "Goal")
	assert.Contains(t, summary, "Progress")
}

func TestCompactMessagesInsertsSummary(t *testing.T) {
	agent := &model.HostingAgent{KeepRecentTokens: 8, ContextWindow: 100, ReserveTokens: 10}
	messages := []BrainMessage{
		{Role: "user", Content: strings.Repeat("history ", 80)},
		{Role: "assistant", Content: "did work"},
		{Role: "user", Content: "now"},
	}
	require.True(t, NeedsCompaction(messages, agent))
	out := CompactMessages(agent, messages, "Goal: keep hosting")
	require.NotEmpty(t, out)
	assert.Contains(t, out[0].Content, "COMPACTION")
}

func TestTransformContextInjectsSnapshotAndSkill(t *testing.T) {
	out := TransformContext([]BrainMessage{{Role: "user", Content: "hello"}}, ContextExtra{
		Snapshot: `{"auto_disabled_count":2}`,
		Skill:    "Skill: channel auto-disable playbook.",
	})
	require.GreaterOrEqual(t, len(out), 3)
	assert.Contains(t, out[0].Content, snapshotMarker)
	assert.Contains(t, out[1].Content, skillMarker)
}

func TestLooksLikeContextOverflowAfterCompactStillDetected(t *testing.T) {
	assert.True(t, LooksLikeContextOverflow(errors.New("maximum context length"), &BrainResponse{RawError: "too long"}))
	assert.False(t, LooksLikeContextOverflow(errors.New("upstream timeout"), nil))
}

func TestValidateAgentCronRejectsAlwaysOn(t *testing.T) {
	require.Error(t, ValidateAgentCron("* * * * *", 300))
	require.Error(t, ValidateAgentCron("*/1 * * * *", 300))
	require.NoError(t, ValidateAgentCron("0 2 * * *", 300))
}
