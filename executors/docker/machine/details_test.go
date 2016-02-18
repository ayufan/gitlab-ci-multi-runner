package machine

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMachineDetailsUsed(t *testing.T) {
	d := machineDetails{}
	d.State = machineStateIdle
	assert.False(t, d.isUsed())
	d.State = machineStateAcquired
	assert.True(t, d.isUsed())
	d.State = machineStateCreating
	assert.True(t, d.isUsed())
	d.State = machineStateUsed
	assert.True(t, d.isUsed())
	d.State = machineStateRemoving
	assert.True(t, d.isUsed())
}

func TestMachineDetailsMatcher(t *testing.T) {
	d := machineDetails{Name: newMachineName("machine-template-%s")}
	assert.True(t, d.match("machine-template-%s"))
	assert.False(t, d.match("machine-other-template-%s"))
}
