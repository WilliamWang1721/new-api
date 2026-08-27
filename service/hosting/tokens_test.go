package hosting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPAllowed(t *testing.T) {
	assert.True(t, ipAllowed("", "1.2.3.4"))
	assert.True(t, ipAllowed("1.2.3.4", "1.2.3.4"))
	assert.False(t, ipAllowed("1.2.3.4", "5.6.7.8"))
	assert.True(t, ipAllowed("10.0.0.0/8", "10.1.2.3"))
	assert.False(t, ipAllowed("10.0.0.0/8", "11.1.2.3"))
}
