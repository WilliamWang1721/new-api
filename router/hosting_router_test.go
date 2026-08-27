package router

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/QuantumNous/new-api/service/hosting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestHostingAgentTokenCallsRuntimeSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevDB := model.DB
	prevLog := model.LOG_DB
	prevRedis := common.RedisEnabled
	prevMaster := common.IsMasterNode
	prevHosting := common.HostingEnabled
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
	common.IsMasterNode = true
	common.HostingEnabled = true
	t.Setenv("HOSTING_ENABLED", "true")
	t.Cleanup(func() {
		model.DB = prevDB
		model.LOG_DB = prevLog
		common.RedisEnabled = prevRedis
		common.IsMasterNode = prevMaster
		common.HostingEnabled = prevHosting
		_ = sqlDB.Close()
	})

	require.NoError(t, authz.Init(db))
	hosting.Start()
	require.True(t, hosting.IsReady())

	created, err := hosting.CreateAgent(hosting.CreateAgentRequest{
		Name:                  "snapshot-agent",
		BrainModel:            "gpt-test",
		ApplyRecommendedPerms: true,
		IssueToken:            true,
		TokenName:             "ops",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.Secret)
	assert.True(t, strings.HasPrefix(created.Secret, constant.HostingTokenPrefix))

	engine := gin.New()
	api := engine.Group("/api")
	registerHostingRoutes(api)

	req := httptest.NewRequest(http.MethodPost, "/api/hosting/tools/get_runtime_snapshot", bytes.NewReader([]byte("{}")))
	req.Header.Set("Authorization", "Bearer "+created.Secret)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var payload map[string]any
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &payload))
	require.Equal(t, true, payload["success"], w.Body.String())
	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, data, "auto_disabled_count")
	assert.Contains(t, data, "host_resources")
	assert.Equal(t, constant.HostingStatusReady, data["hosting"])
}
