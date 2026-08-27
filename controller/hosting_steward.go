package controller

import (
	"strconv"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/hosting"

	"github.com/gin-gonic/gin"
)

func GetStewardStatus(c *gin.Context) {
	common.ApiSuccess(c, hosting.PublicStewardStatus(c.GetInt("role")))
}

func GetStewardSettings(c *gin.Context) {
	settings, err := hosting.LoadStewardSettings()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, settings)
}

func UpdateStewardSettings(c *gin.Context) {
	var req hosting.UpdateStewardRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	settings, err := hosting.SaveStewardSettings(req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, settings)
}

func PostStewardChat(c *gin.Context) {
	var req struct {
		Message string `json:"message"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(req.Message) > constant.MaxHostingChatMessageRunes {
		common.ApiErrorMsg(c, "message is too long")
		return
	}
	agent, err := model.GetDefaultHostingAgent()
	if err != nil {
		common.ApiErrorMsg(c, "the steward is not ready yet")
		return
	}
	result, err := hosting.RunChatTurn(agent, c.GetInt("id"), c.GetInt("role"), req.Message, c.ClientIP())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetStewardSession(c *gin.Context) {
	agent, err := model.GetDefaultHostingAgent()
	if err != nil {
		common.ApiErrorMsg(c, "the steward is not ready yet")
		return
	}
	row, entries, err := hosting.StewardChatSession(agent.Id, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	sessionId := ""
	if row != nil {
		sessionId = row.SessionId
	}
	common.ApiSuccess(c, gin.H{
		"session_id": sessionId,
		"entries":    entries,
	})
}

func RotateStewardSession(c *gin.Context) {
	agent, err := model.GetDefaultHostingAgent()
	if err != nil {
		common.ApiErrorMsg(c, "the steward is not ready yet")
		return
	}
	row, err := model.RotateHostingChatSession(agent.Id, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, row)
}

func GetStewardApprovals(c *gin.Context) {
	agent, err := model.GetDefaultHostingAgent()
	if err != nil {
		common.ApiSuccess(c, []*model.HostingApproval{})
		return
	}
	items, err := hosting.ListApprovalsForActor(agent.Id, c.GetInt("id"), c.GetInt("role"), c.Query("status"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}

func DecideStewardApproval(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid approval id")
		return
	}
	var req hosting.ApprovalDecision
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	item, err := hosting.DecideApproval(id, c.GetInt("id"), c.GetInt("role"), req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func GetStewardBriefing(c *gin.Context) {
	agent, err := model.GetDefaultHostingAgent()
	if err != nil {
		common.ApiErrorMsg(c, "the steward is not ready yet")
		return
	}
	force := c.Query("refresh") == "1" || c.Query("refresh") == "true"
	result, err := hosting.GenerateStewardBriefing(agent, c.GetInt("id"), force)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}
