package hosting

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

func RunPlaybook(agent *model.HostingAgent, hook *model.HostingHook, event service.HostingEvent) (bool, []string) {
	var actions []string
	if event.ChannelId > 0 {
		channel, err := model.GetChannelById(event.ChannelId, true)
		if err == nil && strings.TrimSpace(channel.Group) == "" {
			preferred := constantDefaultGroups(agent)
			channel.Group = ResolveChannelGroups("", preferred)
			if err := channel.Update(); err == nil {
				actions = append(actions, fmt.Sprintf("filled channel #%d groups to %s", channel.Id, channel.Group))
			}
		}
	}
	if event.Name == "channel.auto_disabled" && event.ChannelId > 0 {
		channel, err := model.GetChannelById(event.ChannelId, true)
		if err != nil {
			return false, actions
		}
		if strings.TrimSpace(channel.Group) == "" {
			return false, actions
		}
		ok, extra := repairAutoDisabledChannel(channel, ProbeChannelKey)
		actions = append(actions, extra...)
		return ok, actions
	}
	if len(actions) > 0 && event.Name != "channel.auto_disabled" {
		return true, actions
	}
	return false, actions
}

type keyProbeFunc func(channel *model.Channel, key string) (bool, string)

func UnhealthyKeyIndexes(keys []string, healthy func(string) bool) []int {
	out := make([]int, 0)
	for i, key := range keys {
		if !healthy(key) {
			out = append(out, i)
		}
	}
	return out
}

func repairAutoDisabledChannel(channel *model.Channel, probe keyProbeFunc) (bool, []string) {
	var actions []string
	if channel.ChannelInfo.IsMultiKey {
		keys := channel.GetKeys()
		healthy := 0
		for i, key := range keys {
			ok, message := probe(channel, key)
			if ok {
				healthy++
				continue
			}
			reason := "hosting playbook: " + message
			if model.UpdateChannelStatus(channel.Id, key, common.ChannelStatusAutoDisabled, reason) {
				actions = append(actions, fmt.Sprintf("disabled unhealthy key #%d: %s", i, message))
			}
		}
		if healthy > 0 {
			if model.UpdateChannelStatus(channel.Id, "", common.ChannelStatusEnabled, "hosting playbook kept healthy keys") {
				actions = append(actions, fmt.Sprintf("re-enabled channel #%d with %d healthy keys", channel.Id, healthy))
				return true, actions
			}
		}
		return false, actions
	}
	ok, message := ProbeChannel(channel)
	actions = append(actions, "probed channel: "+message)
	if ok {
		if model.UpdateChannelStatus(channel.Id, "", common.ChannelStatusEnabled, "hosting playbook probe succeeded") {
			actions = append(actions, fmt.Sprintf("re-enabled channel #%d", channel.Id))
			return true, actions
		}
	}
	return false, actions
}

func constantDefaultGroups(agent *model.HostingAgent) string {
	if agent != nil && strings.TrimSpace(agent.DefaultChannelGroups) != "" {
		return agent.DefaultChannelGroups
	}
	return "default"
}
