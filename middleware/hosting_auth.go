package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/QuantumNous/new-api/service/hosting"
	"github.com/gin-gonic/gin"
)

func AgentAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if raw == "" {
			raw = c.Query("token")
		}
		agent, token, err := hosting.AuthenticateAgentToken(raw, c.ClientIP())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		authz.MarkHostingAgentUser(agent.UserId)
		c.Set("id", agent.UserId)
		c.Set("role", common.RoleAdminUser)
		c.Set("username", agent.Name)
		c.Set("group", "default")
		c.Set("user_group", "default")
		c.Set("auth_method", constant.HostingAuthMethod)
		c.Set("hosting_agent_id", agent.Id)
		c.Set("hosting_token_id", token.Id)
		c.Set("hosting_agent", agent)
		c.Set("hosting_token", token)
		c.Next()
	}
}

func HostingToolContext(c *gin.Context) *hosting.ToolContext {
	raw, ok := c.Get("hosting_agent")
	if !ok {
		return nil
	}
	agent, ok := raw.(*model.HostingAgent)
	if !ok || agent == nil {
		return nil
	}
	var token *model.HostingAgentToken
	if tokenRaw, exists := c.Get("hosting_token"); exists {
		token, _ = tokenRaw.(*model.HostingAgentToken)
	}
	return hosting.ToolContextFromAgent(agent, token, c.ClientIP())
}
