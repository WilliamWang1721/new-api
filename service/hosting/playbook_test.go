package hosting

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
)

func TestUnhealthyKeyIndexes(t *testing.T) {
	indexes := UnhealthyKeyIndexes([]string{"good", "bad", "also-good"}, func(key string) bool {
		return key != "bad"
	})
	assert.Equal(t, []int{1}, indexes)
}

func TestPlaybookEmptyEventDoesNotCallBrain(t *testing.T) {
	called := false
	SetInternalBrainRelay(func(agent *model.HostingAgent, req BrainRequest) (*BrainResponse, error) {
		called = true
		return &BrainResponse{}, nil
	})
	t.Cleanup(func() { SetInternalBrainRelay(nil) })
	ok, actions := RunPlaybook(nil, nil, service.HostingEvent{Name: "other"})
	assert.False(t, ok)
	assert.Empty(t, actions)
	assert.False(t, called)
}

func TestSkillForChannelDisable(t *testing.T) {
	skill := SkillForMessages([]BrainMessage{{Role: "user", Content: "hook=x event=channel.auto_disabled"}})
	assert.Contains(t, skill, "playbook")
}
