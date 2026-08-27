package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

type HostingAgent struct {
	Id                    int    `json:"id"`
	Name                  string `json:"name" gorm:"type:varchar(128);not null"`
	Enabled               bool   `json:"enabled"`
	UserId                int    `json:"user_id" gorm:"uniqueIndex"`
	HandoffUserId         int    `json:"handoff_user_id"`
	BrainSource           string `json:"brain_source" gorm:"type:varchar(32)"`
	BrainModel            string `json:"brain_model" gorm:"type:varchar(191)"`
	BrainGroup            string `json:"brain_group" gorm:"type:varchar(64)"`
	BrainChannelId        int    `json:"brain_channel_id"`
	DedicatedBaseURL      string `json:"dedicated_base_url" gorm:"type:varchar(512)"`
	DedicatedAPIKey       string `json:"-" gorm:"type:text"`
	DedicatedAPIType      string `json:"dedicated_api_type" gorm:"type:varchar(32)"`
	DedicatedHeaders      string `json:"dedicated_headers" gorm:"type:text"`
	DedicatedTimeoutSec   int    `json:"dedicated_timeout_sec"`
	DefaultChannelGroups  string `json:"default_channel_groups" gorm:"type:varchar(255)"`
	MaxActionsPerIncident int    `json:"max_actions_per_incident"`
	DryRun                bool   `json:"dry_run"`
	DailyTokenBudget      int    `json:"daily_token_budget"`
	WakeMergeWindowSec    int    `json:"wake_merge_window_sec"`
	MaxWakesPerHour       int    `json:"max_wakes_per_hour"`
	AllowAgentHooks       bool   `json:"allow_agent_hooks"`
	MaxAgentHooks         int    `json:"max_agent_hooks"`
	MinHookIntervalSec    int    `json:"min_hook_interval_sec"`
	ContextWindow         int    `json:"context_window"`
	ReserveTokens         int    `json:"reserve_tokens"`
	KeepRecentTokens      int    `json:"keep_recent_tokens"`
	SessionId             string `json:"session_id" gorm:"type:varchar(64);index"`
	LastCompactSummary    string `json:"last_compact_summary" gorm:"type:text"`
	Remark                string `json:"remark" gorm:"type:varchar(255)"`
	CreatedAt             int64  `json:"created_at"`
	UpdatedAt             int64  `json:"updated_at"`
}

func (a *HostingAgent) Normalize() {
	a.Name = strings.TrimSpace(a.Name)
	if a.BrainSource != constant.HostingBrainDedicated {
		a.BrainSource = constant.HostingBrainInternal
	}
	if a.DedicatedAPIType == "" {
		a.DedicatedAPIType = "openai"
	}
	if a.DefaultChannelGroups == "" {
		a.DefaultChannelGroups = constant.DefaultHostingChannelGroups
	}
	if a.MaxActionsPerIncident <= 0 {
		a.MaxActionsPerIncident = constant.DefaultHostingMaxActions
	}
	if a.DailyTokenBudget <= 0 {
		a.DailyTokenBudget = constant.DefaultHostingDailyBudget
	}
	if a.WakeMergeWindowSec <= 0 {
		a.WakeMergeWindowSec = constant.DefaultHostingMergeWindow
	}
	if a.MaxWakesPerHour <= 0 {
		a.MaxWakesPerHour = constant.DefaultHostingMaxWakesHour
	}
	if a.MaxAgentHooks <= 0 {
		a.MaxAgentHooks = constant.DefaultHostingMaxAgentHooks
	}
	if a.MinHookIntervalSec <= 0 {
		a.MinHookIntervalSec = constant.DefaultHostingMinHookSec
	}
	if a.ContextWindow <= 0 {
		a.ContextWindow = constant.DefaultHostingContextWindow
	}
	if a.ReserveTokens <= 0 {
		a.ReserveTokens = constant.DefaultHostingReserveTokens
	}
	if a.KeepRecentTokens <= 0 {
		a.KeepRecentTokens = constant.DefaultHostingKeepRecent
	}
	if a.DedicatedTimeoutSec <= 0 {
		a.DedicatedTimeoutSec = 60
	}
}

type HostingAgentToken struct {
	Id          int    `json:"id"`
	AgentId     int    `json:"agent_id" gorm:"index"`
	Name        string `json:"name" gorm:"type:varchar(128)"`
	TokenHash   string `json:"-" gorm:"type:varchar(128);uniqueIndex"`
	TokenPrefix string `json:"token_prefix" gorm:"type:varchar(32)"`
	AllowIPs    string `json:"allow_ips" gorm:"type:varchar(512)"`
	Status      int    `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	LastUsedAt  int64  `json:"last_used_at"`
}

type HostingHook struct {
	Id          int    `json:"id"`
	AgentId     int    `json:"agent_id" gorm:"index"`
	Owner       string `json:"owner" gorm:"type:varchar(16);index"`
	Name        string `json:"name" gorm:"type:varchar(128)"`
	Enabled     bool   `json:"enabled"`
	Kind        string `json:"kind" gorm:"type:varchar(16);index"`
	EventName   string `json:"event_name" gorm:"type:varchar(64);index"`
	Cron        string `json:"cron" gorm:"type:varchar(64)"`
	FireAt      int64  `json:"fire_at"`
	NextFireAt  int64  `json:"next_fire_at" gorm:"index"`
	Condition   string `json:"condition" gorm:"type:varchar(64)"`
	ConditionN  int    `json:"condition_n"`
	ChannelId   int    `json:"channel_id"`
	WakeMode    string `json:"wake_mode" gorm:"type:varchar(32)"`
	CooldownSec int    `json:"cooldown_sec"`
	MaxFires    int    `json:"max_fires"`
	FireCount   int    `json:"fire_count"`
	MergeKey    string `json:"merge_key" gorm:"type:varchar(128)"`
	PromptHint  string `json:"prompt_hint" gorm:"type:text"`
	LastFiredAt int64  `json:"last_fired_at"`
	SystemKey   string `json:"system_key" gorm:"type:varchar(64);index"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type HostingIncident struct {
	Id             int    `json:"id"`
	AgentId        int    `json:"agent_id" gorm:"index"`
	Status         string `json:"status" gorm:"type:varchar(32);index"`
	SourceHookId   int    `json:"source_hook_id"`
	SourceEvent    string `json:"source_event" gorm:"type:varchar(64)"`
	Summary        string `json:"summary" gorm:"type:text"`
	ContextJSON    string `json:"context_json" gorm:"type:text"`
	ActionsJSON    string `json:"actions_json" gorm:"type:text"`
	HandoffReason  string `json:"handoff_reason" gorm:"type:text"`
	AssigneeUserId int    `json:"assignee_user_id"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type HostingSessionEntry struct {
	Id         int    `json:"id"`
	AgentId    int    `json:"agent_id" gorm:"index"`
	SessionId  string `json:"session_id" gorm:"type:varchar(64);index"`
	Seq        int64  `json:"seq"`
	Role       string `json:"role" gorm:"type:varchar(32)"`
	Name       string `json:"name" gorm:"type:varchar(64)"`
	Content    string `json:"content" gorm:"type:text"`
	ToolCallID string `json:"tool_call_id" gorm:"type:varchar(64)"`
	TokenCount int    `json:"token_count"`
	CreatedAt  int64  `json:"created_at"`
}

type HostingBrainUsage struct {
	Id           int    `json:"id"`
	AgentId      int    `json:"agent_id" gorm:"index"`
	Day          string `json:"day" gorm:"type:varchar(16);index"`
	PromptTokens int    `json:"prompt_tokens"`
	OutputTokens int    `json:"output_tokens"`
	WakeCount    int    `json:"wake_count"`
	ErrorCount   int    `json:"error_count"`
	UpdatedAt    int64  `json:"updated_at"`
}

func HostingAutoMigrateModels() []any {
	return []any{
		&HostingAgent{},
		&HostingAgentToken{},
		&HostingHook{},
		&HostingIncident{},
		&HostingSessionEntry{},
		&HostingBrainUsage{},
	}
}

func GetHostingAgentById(id int) (*HostingAgent, error) {
	if id <= 0 {
		return nil, errors.New("agent id is empty")
	}
	var agent HostingAgent
	if err := DB.First(&agent, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

func GetHostingAgentByUserId(userId int) (*HostingAgent, error) {
	var agent HostingAgent
	if err := DB.First(&agent, "user_id = ?", userId).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

func ListHostingAgents() ([]*HostingAgent, error) {
	var agents []*HostingAgent
	err := DB.Order("id desc").Find(&agents).Error
	return agents, err
}

func (a *HostingAgent) Insert() error {
	now := common.GetTimestamp()
	a.CreatedAt = now
	a.UpdatedAt = now
	a.Normalize()
	return DB.Create(a).Error
}

func (a *HostingAgent) Update() error {
	a.UpdatedAt = common.GetTimestamp()
	a.Normalize()
	return DB.Model(a).Updates(a).Error
}

func DeleteHostingAgent(id int) error {
	return DB.Delete(&HostingAgent{}, "id = ?", id).Error
}

func GetHostingTokenByHash(hash string) (*HostingAgentToken, error) {
	var token HostingAgentToken
	if err := DB.First(&token, "token_hash = ? AND status = ?", hash, common.UserStatusEnabled).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func ListHostingTokens(agentId int) ([]*HostingAgentToken, error) {
	var tokens []*HostingAgentToken
	err := DB.Where("agent_id = ?", agentId).Order("id desc").Find(&tokens).Error
	return tokens, err
}

func (t *HostingAgentToken) Insert() error {
	t.CreatedAt = common.GetTimestamp()
	if t.Status == 0 {
		t.Status = common.UserStatusEnabled
	}
	return DB.Create(t).Error
}

func RevokeHostingToken(id int, agentId int) error {
	return DB.Model(&HostingAgentToken{}).Where("id = ? AND agent_id = ?", id, agentId).Update("status", common.UserStatusDisabled).Error
}

func TouchHostingToken(id int) {
	_ = DB.Model(&HostingAgentToken{}).Where("id = ?", id).Update("last_used_at", common.GetTimestamp()).Error
}

func ListHostingHooks(agentId int) ([]*HostingHook, error) {
	var hooks []*HostingHook
	q := DB.Order("id asc")
	if agentId > 0 {
		q = q.Where("agent_id = ? OR owner = ?", agentId, constant.HostingHookOwnerSystem)
	}
	err := q.Find(&hooks).Error
	return hooks, err
}

func ListEnabledHostingHooks() ([]*HostingHook, error) {
	var hooks []*HostingHook
	err := DB.Where("enabled = ?", true).Find(&hooks).Error
	return hooks, err
}

func GetHostingHookById(id int) (*HostingHook, error) {
	var hook HostingHook
	if err := DB.First(&hook, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &hook, nil
}

func CountAgentHooks(agentId int) (int64, error) {
	var n int64
	err := DB.Model(&HostingHook{}).Where("agent_id = ? AND owner = ?", agentId, constant.HostingHookOwnerAgent).Count(&n).Error
	return n, err
}

func (h *HostingHook) Insert() error {
	now := common.GetTimestamp()
	h.CreatedAt = now
	h.UpdatedAt = now
	return DB.Create(h).Error
}

func (h *HostingHook) Update() error {
	h.UpdatedAt = common.GetTimestamp()
	return DB.Model(h).Updates(h).Error
}

func DeleteHostingHook(id int) error {
	return DB.Delete(&HostingHook{}, "id = ?", id).Error
}

func GetSystemHookByKey(key string) (*HostingHook, error) {
	var hook HostingHook
	if err := DB.First(&hook, "system_key = ?", key).Error; err != nil {
		return nil, err
	}
	return &hook, nil
}

func ListHostingIncidents(agentId int, status string, limit int) ([]*HostingIncident, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var items []*HostingIncident
	q := DB.Order("id desc").Limit(limit)
	if agentId > 0 {
		q = q.Where("agent_id = ?", agentId)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Find(&items).Error
	return items, err
}

func GetHostingIncidentById(id int) (*HostingIncident, error) {
	var item HostingIncident
	if err := DB.First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (i *HostingIncident) Insert() error {
	now := common.GetTimestamp()
	i.CreatedAt = now
	i.UpdatedAt = now
	if i.Status == "" {
		i.Status = constant.HostingIncidentOpen
	}
	return DB.Create(i).Error
}

func (i *HostingIncident) Update() error {
	i.UpdatedAt = common.GetTimestamp()
	return DB.Model(i).Updates(i).Error
}

func AppendHostingSessionEntry(entry *HostingSessionEntry) error {
	if entry.CreatedAt == 0 {
		entry.CreatedAt = common.GetTimestamp()
	}
	var maxSeq int64
	_ = DB.Model(&HostingSessionEntry{}).Where("agent_id = ? AND session_id = ?", entry.AgentId, entry.SessionId).Select("COALESCE(MAX(seq), 0)").Scan(&maxSeq).Error
	entry.Seq = maxSeq + 1
	return DB.Create(entry).Error
}

func ListHostingSessionEntries(agentId int, sessionId string, afterSeq int64, limit int) ([]*HostingSessionEntry, error) {
	if limit <= 0 {
		limit = 500
	}
	var entries []*HostingSessionEntry
	q := DB.Where("agent_id = ? AND session_id = ?", agentId, sessionId).Order("seq asc").Limit(limit)
	if afterSeq > 0 {
		q = q.Where("seq > ?", afterSeq)
	}
	err := q.Find(&entries).Error
	return entries, err
}

func CountHostingSessionTokens(agentId int, sessionId string) (int, error) {
	var sum int
	err := DB.Model(&HostingSessionEntry{}).Where("agent_id = ? AND session_id = ?", agentId, sessionId).Select("COALESCE(SUM(token_count), 0)").Scan(&sum).Error
	return sum, err
}

func GetHostingBrainUsage(agentId int, day string) (*HostingBrainUsage, error) {
	var row HostingBrainUsage
	err := DB.Where("agent_id = ? AND day = ?", agentId, day).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func AddHostingBrainUsage(agentId int, promptTokens, outputTokens, wakes, errs int) error {
	day := time.Now().Format("2006-01-02")
	now := common.GetTimestamp()
	var row HostingBrainUsage
	err := DB.Where("agent_id = ? AND day = ?", agentId, day).First(&row).Error
	if err != nil {
		row = HostingBrainUsage{
			AgentId:      agentId,
			Day:          day,
			PromptTokens: promptTokens,
			OutputTokens: outputTokens,
			WakeCount:    wakes,
			ErrorCount:   errs,
			UpdatedAt:    now,
		}
		return DB.Create(&row).Error
	}
	return DB.Model(&row).Updates(map[string]any{
		"prompt_tokens": row.PromptTokens + promptTokens,
		"output_tokens": row.OutputTokens + outputTokens,
		"wake_count":    row.WakeCount + wakes,
		"error_count":   row.ErrorCount + errs,
		"updated_at":    now,
	}).Error
}

func HostingUsageDayTotal(agentId int, day string) int {
	row, err := GetHostingBrainUsage(agentId, day)
	if err != nil {
		return 0
	}
	return row.PromptTokens + row.OutputTokens
}

func NewHostingSessionID() string {
	return fmt.Sprintf("hs-%s", strings.ReplaceAll(common.GetUUID(), "-", "")[:16])
}
