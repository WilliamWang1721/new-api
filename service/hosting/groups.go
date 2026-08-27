package hosting

import (
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// ExistingGroupNames returns group keys that exist in both ratio settings and
// the usable-group catalog. "default" is always eligible when present in either.
func ExistingGroupNames() map[string]struct{} {
	out := make(map[string]struct{})
	for name := range ratio_setting.GetGroupRatioCopy() {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = struct{}{}
		}
	}
	for name := range setting.GetUserUsableGroupsCopy() {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = struct{}{}
		}
	}
	if len(out) == 0 {
		out[constant.DefaultHostingChannelGroups] = struct{}{}
	}
	return out
}

// ResolveChannelGroups fills an empty Channel.Group with preferred groups
// intersected with groups that actually exist on this instance.
func ResolveChannelGroups(group string, preferred string) string {
	if strings.TrimSpace(group) != "" {
		return strings.TrimSpace(group)
	}
	if strings.TrimSpace(preferred) == "" {
		preferred = constant.DefaultHostingChannelGroups
	}
	existing := ExistingGroupNames()
	var kept []string
	seen := make(map[string]struct{})
	for _, part := range strings.Split(preferred, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, ok := existing[name]; !ok {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		kept = append(kept, name)
	}
	if len(kept) == 0 {
		if _, ok := existing[constant.DefaultHostingChannelGroups]; ok {
			return constant.DefaultHostingChannelGroups
		}
		for name := range existing {
			return name
		}
		return constant.DefaultHostingChannelGroups
	}
	return strings.Join(kept, ",")
}

func SuggestChannelGroups() []map[string]string {
	existing := ExistingGroupNames()
	out := make([]map[string]string, 0, len(existing))
	for name := range existing {
		reason := "Configured routing group"
		if name == constant.DefaultHostingChannelGroups {
			reason = "Default routing group used when a channel has no groups"
		}
		out = append(out, map[string]string{
			"group":  name,
			"reason": reason,
		})
	}
	return out
}
