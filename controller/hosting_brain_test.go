package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestHostingBrainContextPinsChannel(t *testing.T) {
	assert.Equal(t, "17", HostingBrainContextChannelID(&model.HostingAgent{
		BrainSource:    constant.HostingBrainInternal,
		BrainChannelId: 17,
	}))
	assert.Empty(t, HostingBrainContextChannelID(&model.HostingAgent{BrainSource: constant.HostingBrainDedicated, BrainChannelId: 17}))
	assert.Empty(t, HostingBrainContextChannelID(&model.HostingAgent{BrainSource: constant.HostingBrainInternal}))
}
