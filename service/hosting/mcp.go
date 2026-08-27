package hosting

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func HandleMCP(c *gin.Context, ctx *ToolContext) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	var req mcpRequest
	if err := common.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusOK, mcpError(req.ID, -32700, "parse error"))
		return
	}
	switch req.Method {
	case "initialize":
		c.JSON(http.StatusOK, mcpResult(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "new-api-hosting", "version": common.Version},
		}))
	case "notifications/initialized", "initialized":
		c.Status(http.StatusAccepted)
	case "tools/list":
		c.JSON(http.StatusOK, mcpResult(req.ID, map[string]any{
			"tools": mcpTools(ctx),
		}))
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := common.Unmarshal(req.Params, &params); err != nil {
			c.JSON(http.StatusOK, mcpError(req.ID, -32602, "invalid params"))
			return
		}
		result, callErr := ExecuteTool(ctx, params.Name, params.Arguments)
		if callErr != nil {
			c.JSON(http.StatusOK, mcpResult(req.ID, map[string]any{
				"content": []map[string]any{{"type": "text", "text": callErr.Error()}},
				"isError": true,
			}))
			return
		}
		raw, _ := common.Marshal(result)
		c.JSON(http.StatusOK, mcpResult(req.ID, map[string]any{
			"content": []map[string]any{{"type": "text", "text": string(raw)}},
		}))
	default:
		c.JSON(http.StatusOK, mcpError(req.ID, -32601, fmt.Sprintf("method %s not found", req.Method)))
	}
}

func mcpTools(ctx *ToolContext) []map[string]any {
	listed := ListToolsForAgent(ctx)
	out := make([]map[string]any, 0, len(listed))
	for _, item := range listed {
		out = append(out, map[string]any{
			"name":        item["name"],
			"description": item["description"],
			"inputSchema": item["input_schema"],
		})
	}
	return out
}

func mcpResult(id any, result any) gin.H {
	return gin.H{"jsonrpc": "2.0", "id": id, "result": result}
}

func mcpError(id any, code int, message string) gin.H {
	return gin.H{"jsonrpc": "2.0", "id": id, "error": gin.H{"code": code, "message": message}}
}

func ToolContextFromAgent(agent *model.HostingAgent, token *model.HostingAgentToken, ip string) *ToolContext {
	ctx := &ToolContext{
		Agent:       agent,
		UserID:      agent.UserId,
		Role:        common.RoleAdminUser,
		ClientIP:    ip,
		ActorUserID: agent.UserId,
		ActorRole:   common.RoleAdminUser,
	}
	if token != nil {
		ctx.TokenID = token.Id
	}
	return ctx
}
