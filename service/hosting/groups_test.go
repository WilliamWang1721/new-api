package hosting

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestResolveChannelGroupsFillsEmpty(t *testing.T) {
	got := ResolveChannelGroups("", "default")
	assert.Equal(t, constant.DefaultHostingChannelGroups, got)
}

func TestResolveChannelGroupsKeepsExplicit(t *testing.T) {
	got := ResolveChannelGroups("vip", "default")
	assert.Equal(t, "vip", got)
}

func TestResolveChannelGroupsDropsUnknown(t *testing.T) {
	got := ResolveChannelGroups("", "not-a-real-group,default")
	assert.Equal(t, "default", got)
}
