package authz

import (
	"fmt"
	"sort"
	"sync"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var hostingAgentUsers sync.Map

func MarkHostingAgentUser(userID int) {
	if userID > 0 {
		hostingAgentUsers.Store(userID, struct{}{})
	}
}

func UnmarkHostingAgentUser(userID int) {
	hostingAgentUsers.Delete(userID)
}

func IsHostingAgentUser(userID int) bool {
	_, ok := hostingAgentUsers.Load(userID)
	return ok
}

func LoadHostingAgentUserIDs(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	var ids []int
	if err := db.Model(&model.User{}).Where("account_kind = ?", constant.AccountKindAgent).Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		MarkHostingAgentUser(id)
	}
	return nil
}

// RecommendedHostingAgentPermissions is the ops template applied when creating
// an agent: channel read/operate/write, log read, hosting read. It never
// includes sensitive_write or secret_view. Option read/write is granted so the
// steward can walk an operator through system settings without exposing secrets.
func RecommendedHostingAgentPermissions() PermissionsMap {
	return HostingPresetPermissions(constant.HostingPresetOperate)
}

func HostingPresetPermissions(preset string) PermissionsMap {
	switch preset {
	case constant.HostingPresetWatch:
		return PermissionsMap{
			ResourceChannel: {ActionRead: true},
			ResourceLog:     {ActionRead: true},
			ResourceHosting: {ActionRead: true},
			ResourceOption:  {ActionRead: true},
		}
	case constant.HostingPresetFull:
		return PermissionsMap{
			ResourceChannel: {
				ActionRead:    true,
				ActionOperate: true,
				ActionWrite:   true,
			},
			ResourceLog: {
				ActionRead: true,
			},
			ResourceHosting: {
				ActionRead:  true,
				ActionWrite: true,
			},
			ResourceOption: {
				ActionRead:  true,
				ActionWrite: true,
			},
			ResourceUser: {
				ActionRead:  true,
				ActionWrite: true,
			},
			ResourceToken: {
				ActionRead:  true,
				ActionWrite: true,
			},
			ResourceGroup: {
				ActionRead:  true,
				ActionWrite: true,
			},
			ResourceRedemption: {
				ActionRead:  true,
				ActionWrite: true,
			},
			ResourceSubscription: {
				ActionRead: true,
			},
			ResourceVendor: {
				ActionRead: true,
			},
			ResourceModelMeta: {
				ActionRead: true,
			},
			ResourceSystemTask: {
				ActionRead: true,
			},
		}
	default:
		return PermissionsMap{
			ResourceChannel: {
				ActionRead:    true,
				ActionOperate: true,
				ActionWrite:   true,
			},
			ResourceLog: {
				ActionRead: true,
			},
			ResourceHosting: {
				ActionRead: true,
			},
			ResourceOption: {
				ActionRead:  true,
				ActionWrite: true,
			},
		}
	}
}

// SetHostingAgentPermissions writes every granted action as an explicit allow.
// Unlike SetUserPermissions, matching the admin baseline is not omitted — agents
// have an empty role matrix, so omitted allows would become denials.
func SetHostingAgentPermissions(userID int, permissions PermissionsMap) error {
	e := currentEnforcer()
	if e == nil {
		return fmt.Errorf("authz enforcer is not initialized")
	}
	if err := ClearUserPermissions(userID); err != nil {
		return err
	}
	for _, policy := range hostingAgentAllowPolicies(permissions) {
		if _, err := e.AddPolicy(UserSubject(userID), policy.Resource, policy.Action, policy.Effect); err != nil {
			return err
		}
	}
	return nil
}

func SetHostingAgentPermissionsInTx(tx *gorm.DB, userID int, permissions PermissionsMap) error {
	if err := ClearUserPermissionsInTx(tx, userID); err != nil {
		return err
	}
	policies := hostingAgentAllowPolicies(permissions)
	if len(policies) == 0 {
		return nil
	}
	rules := make([]model.CasbinRule, 0, len(policies))
	for _, policy := range policies {
		rules = append(rules, newRule("p", []string{UserSubject(userID), policy.Resource, policy.Action, policy.Effect}))
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rules).Error
}

func hostingAgentAllowPolicies(permissions PermissionsMap) []overridePolicy {
	policies := make([]overridePolicy, 0)
	for resource, actions := range permissions {
		if !isKnownResource(resource) {
			continue
		}
		for _, action := range catalogActions(resource) {
			if action.Action == ActionSecretView {
				continue
			}
			if !actions[action.Action] {
				continue
			}
			policies = append(policies, overridePolicy{
				Resource: resource,
				Action:   action.Action,
				Effect:   EffectAllow,
			})
		}
	}
	sort.Slice(policies, func(i, j int) bool {
		if policies[i].Resource == policies[j].Resource {
			return policies[i].Action < policies[j].Action
		}
		return policies[i].Resource < policies[j].Resource
	})
	return policies
}
