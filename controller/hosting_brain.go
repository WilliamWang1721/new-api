package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service/hosting"

	"github.com/gin-gonic/gin"
)

func init() {
	hosting.SetInternalBrainRelay(RelayHostingBrain)
}

// RelayHostingBrain sends an internal-channel brain request through Distribute
// and Relay so billing, retry, grouping, and consume logs stay on the normal path.
func RelayHostingBrain(agent *model.HostingAgent, req hosting.BrainRequest) (*hosting.BrainResponse, error) {
	if agent == nil {
		return nil, fmt.Errorf("agent is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		req.Model = agent.BrainModel
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, fmt.Errorf("brain model is empty")
	}

	payload, err := common.Marshal(hosting.BrainCompletionPayload(req))
	if err != nil {
		return nil, err
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(payload))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httpReq
	defer common.CleanupBodyStorage(c)

	if err := applyHostingBrainContext(c, agent); err != nil {
		return nil, err
	}

	middleware.Distribute()(c)
	if c.IsAborted() {
		return hostingCompletionError(w, "internal brain distribute failed")
	}
	Relay(c, types.RelayFormatOpenAI)
	if w.Code >= 400 {
		return hostingCompletionError(w, fmt.Sprintf("internal brain HTTP %d", w.Code))
	}
	resp, parseErr := hosting.ParseChatCompletionBody(w.Body.Bytes())
	if parseErr != nil {
		return nil, parseErr
	}
	if agent.BrainChannelId > 0 {
		resp.PinnedChannel = agent.BrainChannelId
	} else if id := c.GetInt("channel_id"); id > 0 {
		resp.PinnedChannel = id
	}
	return resp, nil
}

func applyHostingBrainContext(c *gin.Context, agent *model.HostingAgent) error {
	cache, err := model.GetUserCache(agent.UserId)
	if err != nil {
		return fmt.Errorf("hosting brain user cache: %w", err)
	}
	if cache.Status != common.UserStatusEnabled {
		return fmt.Errorf("hosting user is disabled")
	}
	cache.WriteContext(c)
	c.Set("id", agent.UserId)
	c.Set("token_name", "hosting-brain")
	c.Set("token_unlimited_quota", true)
	c.Set("token_model_limit_enabled", false)
	c.Set(string(constant.ContextKeyTokenUnlimited), true)
	requestId := common.GetTimeString()
	c.Set(common.RequestIdKey, requestId)
	c.Header(common.RequestIdKey, requestId)

	group := strings.TrimSpace(agent.BrainGroup)
	if group == "" {
		group = cache.Group
	}
	if group == "" {
		group = "default"
	}
	common.SetContextKey(c, constant.ContextKeyUsingGroup, group)
	if agent.BrainSource == constant.HostingBrainInternal && agent.BrainChannelId > 0 {
		common.SetContextKey(c, constant.ContextKeyTokenSpecificChannelId, strconv.Itoa(agent.BrainChannelId))
	}
	return nil
}

func hostingCompletionError(w *httptest.ResponseRecorder, fallback string) (*hosting.BrainResponse, error) {
	raw := w.Body.Bytes()
	msg := fallback
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := common.Unmarshal(raw, &parsed); err == nil && parsed.Error.Message != "" {
		msg = parsed.Error.Message
	}
	return &hosting.BrainResponse{RawError: string(raw), FinishReason: "error"}, fmt.Errorf("%s", msg)
}

// HostingBrainContextChannelID is exported for tests of pinned-channel setup.
func HostingBrainContextChannelID(agent *model.HostingAgent) string {
	if agent == nil || agent.BrainSource != constant.HostingBrainInternal || agent.BrainChannelId <= 0 {
		return ""
	}
	return strconv.Itoa(agent.BrainChannelId)
}
