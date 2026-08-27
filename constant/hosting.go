package constant

const (
	AccountKindHuman = ""
	AccountKindAgent = "agent"

	HostingAuthMethod = "hosting_agent"

	HostingTokenPrefix = "hat-"

	HostingBrainInternal  = "internal_channel"
	HostingBrainDedicated = "dedicated"

	HostingHookOwnerSystem = "system"
	HostingHookOwnerAgent  = "agent"

	HostingHookKindEvent     = "event"
	HostingHookKindSchedule  = "schedule"
	HostingHookKindCondition = "condition"

	HostingWakePlaybookOnly = "playbook_only"
	HostingWakeAI           = "wake_ai"
	HostingWakeNotifyOnly   = "notify_only"

	HostingIncidentOpen         = "open"
	HostingIncidentAutoResolved = "auto_resolved"
	HostingIncidentHandedOff    = "handed_off"
	HostingIncidentIgnored      = "ignored"

	HostingStatusDisabled = "disabled"
	HostingStatusReady    = "ready"
	HostingStatusError    = "error"

	NotifyTypeHostingHandoff = "hosting_handoff"

	HostingEventChannelDisabled = "channel.auto_disabled"
	HostingEventChannelEnabled  = "channel.auto_enabled"
	HostingEventChannelTest     = "channel.test_failed"
	HostingEventQuota           = "quota.exhausted"

	DefaultHostingChannelGroups = "default"
	DefaultHostingContextWindow = 128000
	DefaultHostingReserveTokens = 20000
	DefaultHostingKeepRecent    = 20000
	DefaultHostingDailyBudget   = 200000
	DefaultHostingMergeWindow   = 60
	DefaultHostingMaxWakesHour  = 10
	DefaultHostingMaxActions    = 20
	DefaultHostingMaxAgentHooks = 20
	DefaultHostingMinHookSec    = 300
)
