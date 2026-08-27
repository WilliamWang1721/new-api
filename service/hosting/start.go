package hosting

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/bytedance/gopkg/util/gopool"
)

const hookTickInterval = 15 * time.Second

// Start migrates hosting tables and launches hook evaluation after the API is
// already listening. Any failure disables hosting only.
func Start() {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("hosting start panic: %v", r)
			common.SysError(msg)
			setRuntime(constant.HostingStatusError, msg)
		}
	}()

	if !common.GetEnvOrDefaultBool("HOSTING_ENABLED", true) {
		setRuntime(constant.HostingStatusDisabled, "")
		common.SysLog("intelligent hosting disabled by HOSTING_ENABLED")
		return
	}
	if model.DB == nil {
		setRuntime(constant.HostingStatusError, "database is not initialized")
		return
	}

	if err := model.DB.AutoMigrate(model.HostingAutoMigrateModels()...); err != nil {
		msg := "hosting migrate failed: " + err.Error()
		common.SysError(msg)
		setRuntime(constant.HostingStatusError, msg)
		return
	}

	if err := authz.LoadHostingAgentUserIDs(model.DB); err != nil {
		common.SysError("hosting failed to load agent users: " + err.Error())
	}

	if err := seedSystemHooks(); err != nil {
		msg := "hosting failed to seed system hooks: " + err.Error()
		common.SysError(msg)
		setRuntime(constant.HostingStatusError, msg)
		return
	}

	service.SetHostingEventSink(func(event service.HostingEvent) {
		defer func() { _ = recover() }()
		HandleHostingEvent(event)
	})

	gopool.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				common.SysError(fmt.Sprintf("hosting hook ticker panic: %v", r))
			}
		}()
		ticker := time.NewTicker(hookTickInterval)
		defer ticker.Stop()
		for range ticker.C {
			if !IsReady() {
				continue
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.LogWarn(nil, fmt.Sprintf("hosting hook evaluation panic: %v", r))
					}
				}()
				EvaluateHooks()
			}()
		}
	})

	setRuntime(constant.HostingStatusReady, "")
	common.SysLog("intelligent hosting ready")
}
