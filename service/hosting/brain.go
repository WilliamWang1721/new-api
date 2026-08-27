package hosting

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

type BrainMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type BrainRequest struct {
	Model          string         `json:"model"`
	Messages       []BrainMessage `json:"messages"`
	Tools          []BrainTool    `json:"tools,omitempty"`
	MaxTokens      *int           `json:"max_tokens,omitempty"`
	PromptCacheOff bool           `json:"-"`
}

type BrainTool struct {
	Type     string      `json:"type"`
	Function BrainToolFn `json:"function"`
}

type BrainToolFn struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type BrainResponse struct {
	Content       string
	ToolCalls     []ToolCall
	PromptTokens  int
	OutputTokens  int
	FinishReason  string
	RawError      string
	PinnedChannel int
}

// InternalBrainRelayFunc runs an internal-channel brain call through the full
// relay path (distribute, retry, billing, consume logs).
type InternalBrainRelayFunc func(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error)

var (
	internalBrainRelay   InternalBrainRelayFunc
	internalBrainRelayMu sync.RWMutex
)

func SetInternalBrainRelay(fn InternalBrainRelayFunc) {
	internalBrainRelayMu.Lock()
	defer internalBrainRelayMu.Unlock()
	internalBrainRelay = fn
}

func getInternalBrainRelay() InternalBrainRelayFunc {
	internalBrainRelayMu.RLock()
	defer internalBrainRelayMu.RUnlock()
	return internalBrainRelay
}

func brainHTTPClient(timeoutSec int) *http.Client {
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	return &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
}

func ChatWithBrain(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error) {
	if agent == nil {
		return nil, fmt.Errorf("agent is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		req.Model = agent.BrainModel
	}
	switch agent.BrainSource {
	case constant.HostingBrainDedicated:
		return chatDedicated(agent, req)
	default:
		return chatInternal(agent, req)
	}
}

func chatInternal(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error) {
	fn := getInternalBrainRelay()
	if fn == nil {
		return nil, fmt.Errorf("internal brain relay is not registered")
	}
	return fn(agent, req)
}

func chatDedicated(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error) {
	key, err := common.DecryptSecret(agent.DedicatedAPIKey)
	if err != nil {
		return nil, fmt.Errorf("dedicated brain key is invalid")
	}
	base := strings.TrimRight(agent.DedicatedBaseURL, "/")
	headers := ParseExtraHeaders(agent.DedicatedHeaders)
	return postChatCompletions(base, key, headers, agent.DedicatedTimeoutSec, req)
}

func ParseExtraHeaders(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]string{}
	if err := common.UnmarshalJsonStr(raw, &out); err == nil {
		return out
	}
	for _, line := range strings.Split(raw, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return out
}

func postChatCompletions(baseURL, apiKey string, extra map[string]string, timeoutSec int, req BrainRequest) (*BrainResponse, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("brain base URL is empty")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("brain API key is empty")
	}
	if req.Model == "" {
		return nil, fmt.Errorf("brain model is empty")
	}
	body := map[string]any{
		"model":    req.Model,
		"messages": req.Messages,
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}
	if req.MaxTokens != nil {
		body["max_tokens"] = *req.MaxTokens
	}
	payload, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	for k, v := range extra {
		httpReq.Header.Set(k, v)
	}
	resp, err := brainHTTPClient(timeoutSec).Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		return &BrainResponse{RawError: string(raw), FinishReason: "error"}, fmt.Errorf("brain HTTP %d", resp.StatusCode)
	}
	return ParseChatCompletionBody(raw)
}

func ParseChatCompletionBody(raw []byte) (*BrainResponse, error) {
	var parsed struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := common.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if parsed.Error.Message != "" {
		return &BrainResponse{RawError: parsed.Error.Message, FinishReason: "error"}, fmt.Errorf("%s", parsed.Error.Message)
	}
	out := &BrainResponse{
		PromptTokens: parsed.Usage.PromptTokens,
		OutputTokens: parsed.Usage.CompletionTokens,
	}
	if len(parsed.Choices) > 0 {
		out.Content = parsed.Choices[0].Message.Content
		out.ToolCalls = parsed.Choices[0].Message.ToolCalls
		out.FinishReason = parsed.Choices[0].FinishReason
	}
	return out, nil
}

func ProbeChannel(channel *model.Channel) (bool, string) {
	if channel == nil {
		return false, "channel not found"
	}
	key, _, err := channel.GetNextEnabledKey()
	if err != nil {
		return false, "no enabled key"
	}
	return ProbeChannelKey(channel, key)
}

func ProbeChannelKey(channel *model.Channel, key string) (bool, string) {
	if channel == nil {
		return false, "channel not found"
	}
	if strings.TrimSpace(key) == "" {
		return false, "empty key"
	}
	base := strings.TrimRight(channel.GetBaseURL(), "/")
	if base == "" {
		return false, "empty base URL"
	}
	req, reqErr := http.NewRequest(http.MethodGet, base+"/v1/models", nil)
	if reqErr != nil {
		return false, reqErr.Error()
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, doErr := brainHTTPClient(15).Do(req)
	if doErr != nil {
		return false, doErr.Error()
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return true, "ok"
}

func FetchUpstreamModelNames(channel *model.Channel) ([]string, string) {
	ok, message := ProbeChannel(channel)
	if !ok {
		return nil, message
	}
	key, _, err := channel.GetNextEnabledKey()
	if err != nil {
		return nil, "no enabled key"
	}
	base := strings.TrimRight(channel.GetBaseURL(), "/")
	req, reqErr := http.NewRequest(http.MethodGet, base+"/v1/models", nil)
	if reqErr != nil {
		return nil, reqErr.Error()
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, doErr := brainHTTPClient(15).Do(req)
	if doErr != nil {
		return nil, doErr.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := common.Unmarshal(raw, &parsed); err != nil {
		return nil, err.Error()
	}
	names := make([]string, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		if item.ID != "" {
			names = append(names, item.ID)
		}
	}
	return names, "ok"
}

func TestDedicatedBrain(baseURL, apiKey, modelName string, timeoutSec int, extra map[string]string) (bool, string) {
	req := BrainRequest{
		Model: modelName,
		Messages: []BrainMessage{
			{Role: "user", Content: "ping"},
		},
	}
	maxTokens := 8
	req.MaxTokens = &maxTokens
	resp, err := postChatCompletions(strings.TrimRight(baseURL, "/"), apiKey, extra, timeoutSec, req)
	if err != nil {
		msg := err.Error()
		if resp != nil && resp.RawError != "" {
			msg = common.LocalLogPreview(resp.RawError)
		}
		return false, msg
	}
	return true, "ok"
}
