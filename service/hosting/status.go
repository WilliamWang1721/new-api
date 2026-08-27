package hosting

import (
	"sync"

	"github.com/QuantumNous/new-api/constant"
)

type Runtime struct {
	Enabled bool   `json:"enabled"`
	State   string `json:"state"`
	Error   string `json:"error,omitempty"`
}

var (
	runtimeMu sync.RWMutex
	runtime   = Runtime{State: constant.HostingStatusDisabled}
)

func setRuntime(state, errMsg string) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	runtime.State = state
	runtime.Error = errMsg
	runtime.Enabled = state == constant.HostingStatusReady
}

func GetPublicStatus() string {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	if runtime.State == "" {
		return constant.HostingStatusDisabled
	}
	return runtime.State
}

func GetRuntime() Runtime {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	copy := runtime
	return copy
}

func IsReady() bool {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return runtime.Enabled && runtime.State == constant.HostingStatusReady
}
