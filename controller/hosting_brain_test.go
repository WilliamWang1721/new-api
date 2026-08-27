package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/hosting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestHostingBrainContextPinsChannel(t *testing.T) {
	assert.Equal(t, "17", HostingBrainContextChannelID(&model.HostingAgent{
		BrainSource:    constant.HostingBrainInternal,
		BrainChannelId: 17,
	}))
	assert.Empty(t, HostingBrainContextChannelID(&model.HostingAgent{BrainSource: constant.HostingBrainDedicated, BrainChannelId: 17}))
	assert.Empty(t, HostingBrainContextChannelID(&model.HostingAgent{BrainSource: constant.HostingBrainInternal}))
}

func TestApplyHostingBrainContextSelectsPinnedChannel(t *testing.T) {
	require.NoError(t, i18n.Init())
	gin.SetMode(gin.TestMode)
	prevDB := model.DB
	prevRedis := common.RedisEnabled
	prevCache := common.MemoryCacheEnabled
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Channel{}))
	model.DB = db
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB = prevDB
		common.RedisEnabled = prevRedis
		common.MemoryCacheEnabled = prevCache
		_ = sqlDB.Close()
	})

	user := &model.User{
		Username: "ha-brain",
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Role:     common.RoleAdminUser,
		Quota:    0,
	}
	require.NoError(t, db.Create(user).Error)

	channel := &model.Channel{
		Type:   constant.ChannelTypeOpenAI,
		Key:    "sk-pinned",
		Status: common.ChannelStatusEnabled,
		Name:   "pinned-brain",
		Models: "gpt-test",
		Group:  "default",
	}
	require.NoError(t, db.Create(channel).Error)

	agent := &model.HostingAgent{
		UserId:         user.Id,
		BrainSource:    constant.HostingBrainInternal,
		BrainModel:     "gpt-test",
		BrainGroup:     "default",
		BrainChannelId: channel.Id,
	}

	payload, err := common.Marshal(hosting.BrainCompletionPayload(hosting.BrainRequest{
		Model:    "gpt-test",
		Messages: []hosting.BrainMessage{{Role: "user", Content: "ping"}},
	}))
	require.NoError(t, err)
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(payload))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httpReq

	require.NoError(t, applyHostingBrainContext(c, agent))
	assert.Equal(t, fmt.Sprintf("%d", channel.Id), common.GetContextKeyString(c, constant.ContextKeyTokenSpecificChannelId))

	middleware.Distribute()(c)
	require.False(t, c.IsAborted(), w.Body.String())
	assert.Equal(t, channel.Id, common.GetContextKeyInt(c, constant.ContextKeyChannelId))
}
