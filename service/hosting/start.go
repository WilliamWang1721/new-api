package hosting

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/authz"
)

type hookEvalHandler struct{}

func (hookEvalHandler) Type() string { return model.SystemTaskTypeHostingHooks }

func (hookEvalHandler) Enabled() bool { return IsReady() }

func (hookEvalHandler) Interval() time.Duration { return hookEvalInterval }

func (hookEvalHandler) NewPayload() any { return nil }

func (hookEvalHandler) Run(_ context.Context, task *model.SystemTask, runnerID string) {
	defer func() {
		if r := recover(); r != nil {
			logger.LogWarn(nil, fmt.Sprintf("hosting hook evaluation panic: %v", r))
		}
	}()
	if IsReady() {
		EvaluateHooks()
	}
	_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, map[string]any{"ok": true}, "")
}

const hookEvalInterval = 15 * time.Second

var (
	startOnce sync.Once
	migrated  bool
)

// EnvEnabled is the hard kill switch from HOSTING_ENABLED.
func EnvEnabled() bool {
	return common.GetEnvOrDefaultBool("HOSTING_ENABLED", true)
}

// PanelEnabled is the in-process option HostingEnabled.
func PanelEnabled() bool {
	return common.HostingEnabled
}

func SubsystemEnabled() bool {
	return EnvEnabled() && PanelEnabled()
}

func maybeStartPiSidecar() {
	if strings.TrimSpace(os.Getenv("HOSTING_RUNTIME")) != "pi" {
		return
	}
	common.SysLog("HOSTING_RUNTIME=pi is reserved and not enabled in this release; using the built-in Go runner")
}

func notifyStartFailure(msg string) {
	if model.DB == nil {
		return
	}
	defer func() { _ = recover() }()
	service.NotifyRootUser(dto.NotifyTypeHostingDegraded, "Intelligent hosting failed to start", msg)
}

// ApplyPanelSwitch reacts to HostingEnabled option changes after boot.
func ApplyPanelSwitch() {
	if !EnvEnabled() {
		setRuntime(constant.HostingStatusDisabled, "")
		return
	}
	if !PanelEnabled() {
		setRuntime(constant.HostingStatusDisabled, "")
		common.SysLog("intelligent hosting disabled by panel switch")
		return
	}
	if migrated && GetPublicStatus() != constant.HostingStatusError {
		setRuntime(constant.HostingStatusReady, "")
		return
	}
	Start()
}

// Start migrates hosting tables and launches hook evaluation after the API is
// already listening. Any failure disables hosting only.
func Start() {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("hosting start panic: %v", r)
			common.SysError(msg)
			setRuntime(constant.HostingStatusError, msg)
			notifyStartFailure(msg)
		}
	}()

	common.NotifyHostingEnabledChanged = ApplyPanelSwitch
	maybeStartPiSidecar()

	if !EnvEnabled() {
		setRuntime(constant.HostingStatusDisabled, "")
		common.SysLog("intelligent hosting disabled by HOSTING_ENABLED")
		return
	}
	if !PanelEnabled() {
		setRuntime(constant.HostingStatusDisabled, "")
		common.SysLog("intelligent hosting disabled by panel switch")
		return
	}
	if model.DB == nil {
		msg := "database is not initialized"
		setRuntime(constant.HostingStatusError, msg)
		notifyStartFailure(msg)
		return
	}

	if err := model.DB.AutoMigrate(model.HostingAutoMigrateModels()...); err != nil {
		msg := "hosting migrate failed: " + err.Error()
		common.SysError(msg)
		setRuntime(constant.HostingStatusError, msg)
		notifyStartFailure(msg)
		return
	}
	migrated = true

	if err := authz.LoadHostingAgentUserIDs(model.DB); err != nil {
		common.SysError("hosting failed to load agent users: " + err.Error())
	}

	if err := seedSystemHooks(); err != nil {
		msg := "hosting failed to seed system hooks: " + err.Error()
		common.SysError(msg)
		setRuntime(constant.HostingStatusError, msg)
		notifyStartFailure(msg)
		return
	}

	if err := EnsureDefaultSteward(); err != nil {
		common.SysError("hosting failed to ensure default steward: " + err.Error())
	}

	service.SetHostingEventSink(func(event service.HostingEvent) {
		defer func() { _ = recover() }()
		HandleHostingEvent(event)
	})

	startOnce.Do(func() {
		service.RegisterSystemTaskHandler(hookEvalHandler{})
	})

	setRuntime(constant.HostingStatusReady, "")
	common.SysLog("intelligent hosting ready")
}
