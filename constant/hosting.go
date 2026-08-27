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
	HostingEventHostResource    = "host.resource_threshold"

	HostingMaxConcurrentWakes         = 4
	HostingMaxToolCallsPerIncidentMin = 20

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

	DefaultHostingStewardName = "AI Steward"

	HostingPresetWatch   = "watch"
	HostingPresetOperate = "operate"
	HostingPresetFull    = "full"

	HostingAutoReviewOff          = "off"
	HostingAutoReviewConservative = "conservative"
	HostingAutoReviewBalanced     = "balanced"
	HostingAutoReviewAggressive   = "aggressive"

	HostingBriefingOff       = "off"
	HostingBriefingEveryOpen = "every_open"
	HostingBriefingDaily     = "daily"

	HostingApprovalPending      = "pending"
	HostingApprovalAutoApproved = "auto_approved"
	HostingApprovalApproved     = "approved"
	HostingApprovalDenied       = "denied"
	HostingApprovalExecuted     = "executed"

	HostingApprovalToolAction     = "tool_action"
	HostingApprovalUserPermission = "user_permission"

	HostingToolRiskRead   = "read"
	HostingToolRiskLow    = "low"
	HostingToolRiskMedium = "medium"
	HostingToolRiskHigh   = "high"

	MaxHostingChatMessageRunes  = 4000
	MaxHostingQuotaGrant        = 10_000_000
	MaxHostingAutoQuotaGrant    = 100_000
	MaxHostingOptionValueRunes  = 100_000
	HostingBriefingEveryOpenSec = 600
	HostingBriefingDailySec     = 86400
)
