package authz

const (
	ResourceUser         = "user"
	ResourceToken        = "token"
	ResourceLog          = "log"
	ResourceRedemption   = "redemption"
	ResourceGroup        = "group"
	ResourceVendor       = "vendor"
	ResourceModelMeta    = "model_meta"
	ResourceSubscription = "subscription"
	ResourceSystemTask   = "system_task"
	ResourceOption       = "option"
	ResourceHosting      = "hosting"
)

var (
	UserRead  = Permission{Resource: ResourceUser, Action: ActionRead}
	UserWrite = Permission{Resource: ResourceUser, Action: ActionWrite}

	TokenRead  = Permission{Resource: ResourceToken, Action: ActionRead}
	TokenWrite = Permission{Resource: ResourceToken, Action: ActionWrite}

	LogRead = Permission{Resource: ResourceLog, Action: ActionRead}

	RedemptionRead  = Permission{Resource: ResourceRedemption, Action: ActionRead}
	RedemptionWrite = Permission{Resource: ResourceRedemption, Action: ActionWrite}

	GroupRead  = Permission{Resource: ResourceGroup, Action: ActionRead}
	GroupWrite = Permission{Resource: ResourceGroup, Action: ActionWrite}

	VendorRead  = Permission{Resource: ResourceVendor, Action: ActionRead}
	VendorWrite = Permission{Resource: ResourceVendor, Action: ActionWrite}

	ModelMetaRead  = Permission{Resource: ResourceModelMeta, Action: ActionRead}
	ModelMetaWrite = Permission{Resource: ResourceModelMeta, Action: ActionWrite}

	SubscriptionRead  = Permission{Resource: ResourceSubscription, Action: ActionRead}
	SubscriptionWrite = Permission{Resource: ResourceSubscription, Action: ActionWrite}

	SystemTaskRead  = Permission{Resource: ResourceSystemTask, Action: ActionRead}
	SystemTaskWrite = Permission{Resource: ResourceSystemTask, Action: ActionWrite}

	OptionRead  = Permission{Resource: ResourceOption, Action: ActionRead}
	OptionWrite = Permission{Resource: ResourceOption, Action: ActionWrite}

	HostingRead  = Permission{Resource: ResourceHosting, Action: ActionRead}
	HostingWrite = Permission{Resource: ResourceHosting, Action: ActionWrite}
)

func init() {
	admin := []string{BuiltInRoleAdmin}
	registerCRUD(ResourceUser, "Users", "View users without secrets.", "Create, update, or disable non-root users.", admin)
	registerCRUD(ResourceToken, "Tokens", "View user API tokens without secrets.", "Create or update user API tokens.", admin)
	RegisterResource(ResourceDefinition{
		Resource: ResourceLog,
		LabelKey: "Logs",
		Actions: []ActionDefinition{
			{
				Action:         ActionRead,
				LabelKey:       "Read logs",
				DescriptionKey: "Search consume and error logs.",
				DefaultRoles:   admin,
			},
		},
	})
	registerCRUD(ResourceRedemption, "Redemption codes", "View redemption codes.", "Create or update redemption codes.", admin)
	registerCRUD(ResourceGroup, "Groups", "View routing groups.", "Edit group ratios and usable groups.", admin)
	registerCRUD(ResourceVendor, "Vendors", "View vendors.", "Create or update vendors.", admin)
	registerCRUD(ResourceModelMeta, "Model metadata", "View model metadata.", "Create or update model metadata.", admin)
	registerCRUD(ResourceSubscription, "Subscriptions", "View subscription plans.", "Create or update subscription plans.", admin)
	registerCRUD(ResourceSystemTask, "System tasks", "View scheduled system tasks.", "Create or change scheduled system tasks.", nil)
	registerCRUD(ResourceOption, "System options", "View system options.", "Change system options.", nil)
	registerCRUD(ResourceHosting, "Intelligent hosting", "View hosting agents, hooks, incidents, and snapshots.", "Create or change hosting agents, hooks, and tokens.", nil)
}

func registerCRUD(resource, label, readDesc, writeDesc string, defaultRoles []string) {
	RegisterResource(ResourceDefinition{
		Resource: resource,
		LabelKey: label,
		Actions: []ActionDefinition{
			{
				Action:         ActionRead,
				LabelKey:       "Read " + resource,
				DescriptionKey: readDesc,
				DefaultRoles:   defaultRoles,
			},
			{
				Action:         ActionWrite,
				LabelKey:       "Write " + resource,
				DescriptionKey: writeDesc,
				DefaultRoles:   defaultRoles,
			},
		},
	})
}
