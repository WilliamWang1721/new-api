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
		if err == nil && strings.TrimSpace(channel.Group) != "" {
			ok, message := ProbeChannel(channel)
			actions = append(actions, "probed channel: "+message)
			if ok {
				if model.UpdateChannelStatus(channel.Id, "", common.ChannelStatusEnabled, "hosting playbook probe succeeded") {
					actions = append(actions, fmt.Sprintf("re-enabled channel #%d", channel.Id))
					return true, actions
				}
			}
		}
	}
	if len(actions) > 0 && event.Name != "channel.auto_disabled" {
		return true, actions
	}
	return false, actions
}

func constantDefaultGroups(agent *model.HostingAgent) string {
	if agent != nil && strings.TrimSpace(agent.DefaultChannelGroups) != "" {
		return agent.DefaultChannelGroups
	}
	return "default"
}
