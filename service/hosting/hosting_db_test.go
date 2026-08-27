package hosting

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupHostingDB(t *testing.T) *gorm.DB {
	t.Helper()
	prevDB := model.DB
	prevLog := model.LOG_DB
	prevRedis := common.RedisEnabled
	prevMain := common.MainDatabaseType()
	prevLogType := common.LogDatabaseType()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	models := []any{
		&model.User{},
		&model.Channel{},
		&model.Ability{},
		&model.Log{},
		&model.CasbinRule{},
		&model.AuthzRole{},
		&model.SystemTask{},
	}
	models = append(models, model.HostingAutoMigrateModels()...)
	require.NoError(t, db.AutoMigrate(models...))

	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)

	t.Cleanup(func() {
		model.DB = prevDB
		model.LOG_DB = prevLog
		common.RedisEnabled = prevRedis
		common.SetDatabaseTypes(prevMain, prevLogType)
		_ = sqlDB.Close()
	})
	return db
}

func seedHandoffUser(t *testing.T) *model.User {
	t.Helper()
	user := &model.User{
		Username: "ho-" + common.GetRandomString(8),
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
		Group:    "default",
		AffCode:  "af-" + common.GetRandomString(8),
	}
	require.NoError(t, model.DB.Create(user).Error)
	return user
}

func seedAgent(t *testing.T, mutate func(*model.HostingAgent)) *model.HostingAgent {
	t.Helper()
	handoff := seedHandoffUser(t)
	agentUser := &model.User{
		Username:    "ha-" + common.GetRandomString(8),
		Status:      common.UserStatusEnabled,
		Role:        common.RoleAdminUser,
		Group:       "default",
		AccountKind: constant.AccountKindAgent,
		AffCode:     "ag-" + common.GetRandomString(8),
	}
	require.NoError(t, model.DB.Create(agentUser).Error)
	agent := &model.HostingAgent{
		Name:             "test-agent",
		Enabled:          true,
		UserId:           agentUser.Id,
		HandoffUserId:    handoff.Id,
		BrainSource:      constant.HostingBrainInternal,
		BrainModel:       "gpt-test",
		DailyTokenBudget: 200000,
		ContextWindow:    128000,
		ReserveTokens:    20,
		KeepRecentTokens: 40,
		SessionId:        model.NewHostingSessionID(),
		MaxWakesPerHour:  10,
	}
	agent.Normalize()
	if mutate != nil {
		mutate(agent)
	}
	now := common.GetTimestamp()
	agent.CreatedAt = now
	agent.UpdatedAt = now
	require.NoError(t, model.DB.Create(agent).Error)
	return agent
}
