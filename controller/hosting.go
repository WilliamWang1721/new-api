package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/QuantumNous/new-api/service/hosting"

	"github.com/gin-gonic/gin"
)

func GetHostingStatus(c *gin.Context) {
	rt := hosting.GetRuntime()
	if c.GetInt("role") < common.RoleAdminUser {
		rt.Error = ""
	}
	common.ApiSuccess(c, gin.H{
		"status":   rt,
		"snapshot": hosting.RuntimeSnapshot(),
		"template": authz.RecommendedHostingAgentPermissions(),
	})
}

func GetHostingAgents(c *gin.Context) {
	agents, err := model.ListHostingAgents()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	out := make([]*hosting.AgentView, 0, len(agents))
	for _, agent := range agents {
		out = append(out, hosting.PublicAgent(agent))
	}
	common.ApiSuccess(c, out)
}

func GetHostingAgent(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	agent, err := model.GetHostingAgentById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, hosting.PublicAgent(agent))
}

func CreateHostingAgent(c *gin.Context) {
	var req hosting.CreateAgentRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	result, err := hosting.CreateAgent(req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func UpdateHostingAgent(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req hosting.CreateAgentRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	agent, err := hosting.UpdateAgent(id, req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, agent)
}

func DeleteHostingAgent(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := hosting.DeleteAgent(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func RotateHostingAgentSession(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	agent, err := hosting.RotateAgentSession(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, agent)
}

func GetHostingAgentPermissions(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	agent, err := model.GetHostingAgentById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, authz.Capabilities(agent.UserId, common.RoleAdminUser))
}

func UpdateHostingAgentPermissions(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	agent, err := model.GetHostingAgentById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var perms authz.PermissionsMap
	if err := common.DecodeJson(c.Request.Body, &perms); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := authz.SetHostingAgentPermissions(agent.UserId, perms); err != nil {
		common.ApiError(c, err)
		return
	}
	_ = authz.ReloadPolicy()
	common.ApiSuccess(c, authz.Capabilities(agent.UserId, common.RoleAdminUser))
}

func CreateHostingAgentToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Name     string `json:"name"`
		AllowIPs string `json:"allow_ips"`
	}
	_ = common.DecodeJson(c.Request.Body, &req)
	issued, err := hosting.IssueAgentToken(id, req.Name, req.AllowIPs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"token":  issued.Token,
		"secret": issued.Secret,
	})
}

func ListHostingAgentTokens(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	tokens, err := model.ListHostingTokens(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, tokens)
}

func RotateHostingAgentToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	tokenId, _ := strconv.Atoi(c.Param("token_id"))
	issued, err := hosting.RotateAgentToken(id, tokenId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"token": issued.Token, "secret": issued.Secret})
}

func RevokeHostingAgentToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	tokenId, _ := strconv.Atoi(c.Param("token_id"))
	if err := model.RevokeHostingToken(tokenId, id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func TestHostingBrain(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Model   string `json:"model"`
		Timeout int    `json:"timeout_sec"`
	}
	_ = common.DecodeJson(c.Request.Body, &req)
	if req.APIKey == "" && id > 0 {
		agent, err := model.GetHostingAgentById(id)
		if err == nil && agent.DedicatedAPIKey != "" {
			if key, decErr := common.DecryptSecret(agent.DedicatedAPIKey); decErr == nil {
				req.APIKey = key
			}
			if req.BaseURL == "" {
				req.BaseURL = agent.DedicatedBaseURL
			}
			if req.Model == "" {
				req.Model = agent.BrainModel
			}
		}
	}
	ok, message := hosting.TestDedicatedBrain(req.BaseURL, req.APIKey, req.Model, req.Timeout)
	common.ApiSuccess(c, gin.H{"ok": ok, "message": message})
}

func GetHostingIncidents(c *gin.Context) {
	agentId, _ := strconv.Atoi(c.Query("agent_id"))
	status := c.Query("status")
	items, err := model.ListHostingIncidents(agentId, status, 100)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}

func UpdateHostingIncident(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status string `json:"status"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	item, err := hosting.ResolveIncident(id, req.Status)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func GetHostingHooks(c *gin.Context) {
	agentId, _ := strconv.Atoi(c.Query("agent_id"))
	hooks, err := model.ListHostingHooks(agentId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, hooks)
}

func UpdateHostingHook(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Enabled  *bool  `json:"enabled"`
		WakeMode string `json:"wake_mode"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	hook, err := hosting.AdminUpdateHook(id, req.Enabled, req.WakeMode)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, hook)
}

func DeleteHostingHook(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := hosting.AdminDeleteHook(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetHostingSession(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	entries, err := hosting.SessionTimeline(id, c.Query("session_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, entries)
}

func GetHostingCost(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	common.ApiSuccess(c, hosting.CostSnapshot(id))
}

func GetHostingTools(c *gin.Context) {
	ctx := middleware.HostingToolContext(c)
	if ctx == nil {
		common.ApiErrorMsg(c, "missing agent context")
		return
	}
	common.ApiSuccess(c, hosting.ListToolsForAgent(ctx))
}

func ExecuteHostingTool(c *gin.Context) {
	ctx := middleware.HostingToolContext(c)
	if ctx == nil {
		common.ApiErrorMsg(c, "missing agent context")
		return
	}
	name := c.Param("name")
	var args map[string]any
	_ = common.DecodeJson(c.Request.Body, &args)
	result, err := hosting.ExecuteTool(ctx, name, args)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func HostingMCP(c *gin.Context) {
	ctx := middleware.HostingToolContext(c)
	if ctx == nil {
		common.ApiErrorMsg(c, "missing agent context")
		return
	}
	hosting.HandleMCP(c, ctx)
}
