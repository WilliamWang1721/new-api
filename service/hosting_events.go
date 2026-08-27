package service

import (
	"sync"
)

// HostingEvent is a non-AI signal delivered to the hosting hook engine.
// Emission must never block relay or channel-disable paths.
type HostingEvent struct {
	Name      string
	ChannelId int
	Reason    string
	Payload   map[string]any
}

type HostingEventSink func(event HostingEvent)

var (
	hostingEventSink   HostingEventSink
	hostingEventSinkMu sync.RWMutex
)

func SetHostingEventSink(sink HostingEventSink) {
	hostingEventSinkMu.Lock()
	defer hostingEventSinkMu.Unlock()
	hostingEventSink = sink
}

// EmitHostingEvent delivers a hosting signal without blocking the caller.
// Failures are dropped on purpose so channel disable / relay stay fail-open.
func EmitHostingEvent(event HostingEvent) {
	hostingEventSinkMu.RLock()
	sink := hostingEventSink
	hostingEventSinkMu.RUnlock()
	if sink == nil {
		return
	}
	go func() {
		defer func() { _ = recover() }()
		sink(event)
	}()
}
