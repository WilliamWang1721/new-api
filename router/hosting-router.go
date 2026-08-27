package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func registerHostingRoutes(apiRouter *gin.RouterGroup) {
	admin := apiRouter.Group("/hosting/admin")
	admin.Use(middleware.RootAuth())
	{
		admin.GET("/status", controller.GetHostingStatus)
		admin.GET("/agents", controller.GetHostingAgents)
		admin.GET("/agents/:id", controller.GetHostingAgent)
		admin.POST("/agents", controller.CreateHostingAgent)
		admin.PUT("/agents/:id", controller.UpdateHostingAgent)
		admin.DELETE("/agents/:id", controller.DeleteHostingAgent)
		admin.POST("/agents/:id/session/rotate", controller.RotateHostingAgentSession)
		admin.GET("/agents/:id/permissions", controller.GetHostingAgentPermissions)
		admin.PUT("/agents/:id/permissions", controller.UpdateHostingAgentPermissions)
		admin.GET("/agents/:id/tokens", controller.ListHostingAgentTokens)
		admin.POST("/agents/:id/tokens", controller.CreateHostingAgentToken)
		admin.POST("/agents/:id/tokens/:token_id/rotate", controller.RotateHostingAgentToken)
		admin.DELETE("/agents/:id/tokens/:token_id", controller.RevokeHostingAgentToken)
		admin.POST("/agents/:id/brain/test", controller.TestHostingBrain)
		admin.POST("/brain/test", controller.TestHostingBrain)
		admin.GET("/agents/:id/session", controller.GetHostingSession)
		admin.GET("/agents/:id/cost", controller.GetHostingCost)
		admin.GET("/incidents", controller.GetHostingIncidents)
		admin.PUT("/incidents/:id", controller.UpdateHostingIncident)
		admin.GET("/hooks", controller.GetHostingHooks)
		admin.PUT("/hooks/:id", controller.UpdateHostingHook)
		admin.DELETE("/hooks/:id", controller.DeleteHostingHook)
	}

	agent := apiRouter.Group("/hosting")
	agent.Use(middleware.AgentAuth())
	{
		agent.GET("/tools", controller.GetHostingTools)
		agent.POST("/tools/:name", controller.ExecuteHostingTool)
		agent.POST("/mcp", controller.HostingMCP)
	}
}
