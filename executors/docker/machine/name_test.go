package machine

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMachineNewName(t *testing.T) {
	a := newMachineName("machine-template-%s")
	b := newMachineName("machine-template-%s")
	assert.NotEqual(t, a, b)
}
