package hosting

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
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

func TestBrainCompletionPayloadDisablesStoreForCompaction(t *testing.T) {
	on := BrainCompletionPayload(BrainRequest{
		Model:          "gpt-test",
		Messages:       []BrainMessage{{Role: "user", Content: "sum"}},
		PromptCacheOff: true,
	})
	assert.Equal(t, false, on["store"])
	off := BrainCompletionPayload(BrainRequest{Model: "gpt-test", Messages: []BrainMessage{{Role: "user", Content: "hi"}}})
	assert.NotContains(t, off, "store")
}

func TestDedicatedBrainConnectivityAndPromptCacheOff(t *testing.T) {
	var sawStore any
	var sawAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = common.Unmarshal(raw, &body)
		sawStore = body["store"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"pong"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	t.Cleanup(srv.Close)

	ok, message := TestDedicatedBrain(srv.URL, "sk-test", "gpt-test", 5, nil)
	require.True(t, ok, message)
	assert.Equal(t, "Bearer sk-test", sawAuth)

	resp, err := postChatCompletions(srv.URL, "sk-test", nil, 5, BrainRequest{
		Model:          "gpt-test",
		Messages:       []BrainMessage{{Role: "user", Content: "ping"}},
		PromptCacheOff: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "pong", resp.Content)
	assert.Equal(t, false, sawStore)

	failOK, failMsg := TestDedicatedBrain("http://127.0.0.1:1", "sk-test", "gpt-test", 1, nil)
	assert.False(t, failOK)
	assert.NotEmpty(t, failMsg)
}
