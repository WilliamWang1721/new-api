package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

type HostingApproval struct {
	Id              int    `json:"id"`
	AgentId         int    `json:"agent_id" gorm:"index"`
	RequesterUserId int    `json:"requester_user_id" gorm:"index"`
	TargetUserId    int    `json:"target_user_id"`
	Kind            string `json:"kind" gorm:"type:varchar(32);index"`
	Status          string `json:"status" gorm:"type:varchar(32);index"`
	ToolName        string `json:"tool_name" gorm:"type:varchar(64)"`
	Risk            string `json:"risk" gorm:"type:varchar(16)"`
	Resource        string `json:"resource" gorm:"type:varchar(64)"`
	Action          string `json:"action" gorm:"type:varchar(64)"`
	Summary         string `json:"summary" gorm:"type:varchar(512)"`
	PayloadJSON     string `json:"payload_json" gorm:"type:text"`
	Reason          string `json:"reason" gorm:"type:text"`
	ReviewNote      string `json:"review_note" gorm:"type:text"`
	AutoReviewRule  string `json:"auto_review_rule" gorm:"type:varchar(64)"`
	ReviewerUserId  int    `json:"reviewer_user_id"`
	CreatedAt       int64  `json:"created_at"`
	DecidedAt       int64  `json:"decided_at"`
	ExecutedAt      int64  `json:"executed_at"`
}

type HostingChatSession struct {
	Id               int    `json:"id"`
	AgentId          int    `json:"agent_id" gorm:"uniqueIndex:idx_hosting_chat_actor"`
	UserId           int    `json:"user_id" gorm:"uniqueIndex:idx_hosting_chat_actor"`
	SessionId        string `json:"session_id" gorm:"type:varchar(64);index"`
	LastBriefingAt   int64  `json:"last_briefing_at"`
	LastBriefingText string `json:"last_briefing_text" gorm:"type:text"`
	UpdatedAt        int64  `json:"updated_at"`
}

func (a *HostingApproval) Insert() error {
	if a.Status == "" {
		a.Status = constant.HostingApprovalPending
	}
	now := common.GetTimestamp()
	a.CreatedAt = now
	return DB.Create(a).Error
}

func GetHostingApprovalById(id int) (*HostingApproval, error) {
	if id <= 0 {
		return nil, errors.New("approval id is empty")
	}
	var item HostingApproval
	if err := DB.First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func ListHostingApprovals(agentId, requesterId int, status string, limit int) ([]*HostingApproval, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := DB.Order("id desc").Limit(limit)
	if agentId > 0 {
		q = q.Where("agent_id = ?", agentId)
	}
	if requesterId > 0 {
		q = q.Where("requester_user_id = ?", requesterId)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var items []*HostingApproval
	err := q.Find(&items).Error
	return items, err
}

func CountPendingHostingApprovals(agentId int) (int64, error) {
	q := DB.Model(&HostingApproval{}).Where("status = ?", constant.HostingApprovalPending)
	if agentId > 0 {
		q = q.Where("agent_id = ?", agentId)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func GetOrCreateHostingChatSession(agentId, userId int) (*HostingChatSession, error) {
	if agentId <= 0 || userId <= 0 {
		return nil, fmt.Errorf("agent and user are required")
	}
	var row HostingChatSession
	err := DB.Where("agent_id = ? AND user_id = ?", agentId, userId).First(&row).Error
	if err == nil {
		if row.SessionId == "" {
			row.SessionId = NewHostingChatSessionID(agentId, userId)
			row.UpdatedAt = common.GetTimestamp()
			if err := DB.Model(&row).Updates(map[string]any{
				"session_id": row.SessionId,
				"updated_at": row.UpdatedAt,
			}).Error; err != nil {
				return nil, err
			}
		}
		return &row, nil
	}
	row = HostingChatSession{
		AgentId:   agentId,
		UserId:    userId,
		SessionId: NewHostingChatSessionID(agentId, userId),
		UpdatedAt: common.GetTimestamp(),
	}
	if err := DB.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func RotateHostingChatSession(agentId, userId int) (*HostingChatSession, error) {
	row, err := GetOrCreateHostingChatSession(agentId, userId)
	if err != nil {
		return nil, err
	}
	row.SessionId = NewHostingChatSessionID(agentId, userId)
	row.UpdatedAt = common.GetTimestamp()
	if err := DB.Model(row).Updates(map[string]any{
		"session_id": row.SessionId,
		"updated_at": row.UpdatedAt,
	}).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func NewHostingChatSessionID(agentId, userId int) string {
	return fmt.Sprintf("chat-%d-%d-%d", agentId, userId, common.GetTimestamp())
}

func SaveHostingChatBriefing(agentId, userId int, text string) error {
	row, err := GetOrCreateHostingChatSession(agentId, userId)
	if err != nil {
		return err
	}
	now := common.GetTimestamp()
	return DB.Model(row).Updates(map[string]any{
		"last_briefing_at":   now,
		"last_briefing_text": text,
		"updated_at":         now,
	}).Error
}
