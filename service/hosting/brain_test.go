package hosting

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatInternalUsesRegisteredRelay(t *testing.T) {
	var gotID int
	SetInternalBrainRelay(func(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error) {
		gotID = agent.BrainChannelId
		assert.Equal(t, "gpt-test", req.Model)
		return &BrainResponse{Content: "ok", PinnedChannel: agent.BrainChannelId}, nil
	})
	t.Cleanup(func() { SetInternalBrainRelay(nil) })

	resp, err := ChatWithBrain(&model.HostingAgent{
		BrainSource:    constant.HostingBrainInternal,
		BrainChannelId: 42,
		BrainModel:     "gpt-test",
	}, BrainRequest{Messages: []BrainMessage{{Role: "user", Content: "hi"}}})
	require.NoError(t, err)
	assert.Equal(t, 42, gotID)
	assert.Equal(t, 42, resp.PinnedChannel)
	assert.Equal(t, "ok", resp.Content)
}

func TestChatInternalRequiresRelay(t *testing.T) {
	SetInternalBrainRelay(nil)
	_, err := ChatWithBrain(&model.HostingAgent{BrainSource: constant.HostingBrainInternal, BrainModel: "m"}, BrainRequest{Model: "m"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestParseChatCompletionBody(t *testing.T) {
	resp, err := ParseChatCompletionBody([]byte(`{"choices":[{"message":{"content":"hi","tool_calls":[{"id":"1","type":"function","function":{"name":"sleep_until_hook","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`))
	require.NoError(t, err)
	assert.Equal(t, "hi", resp.Content)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "sleep_until_hook", resp.ToolCalls[0].Function.Name)
	assert.Equal(t, 3, resp.PromptTokens)
}

func TestParseExtraHeaders(t *testing.T) {
	jsonHeaders := ParseExtraHeaders(`{"X-Test":"1"}`)
	assert.Equal(t, "1", jsonHeaders["X-Test"])
	lineHeaders := ParseExtraHeaders("X-Foo: bar\nX-Baz: qux")
	assert.Equal(t, "bar", lineHeaders["X-Foo"])
	assert.Equal(t, "qux", lineHeaders["X-Baz"])
	assert.Nil(t, ParseExtraHeaders("   "))
}
