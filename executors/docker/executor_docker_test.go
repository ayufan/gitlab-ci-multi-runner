package docker

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseDeviceStringOne(t *testing.T) {
	e := executor{}

	device, err := e.parseDeviceString("/dev/kvm")

	assert.NoError(t, err)
	assert.Equal(t, device.PathOnHost, "/dev/kvm")
	assert.Equal(t, device.PathInContainer, "/dev/kvm")
	assert.Equal(t, device.CgroupPermissions, "rwm")
}

func TestParseDeviceStringTwo(t *testing.T) {
	e := executor{}

	device, err := e.parseDeviceString("/dev/kvm:/devices/kvm")

	assert.NoError(t, err)
	assert.Equal(t, device.PathOnHost, "/dev/kvm")
	assert.Equal(t, device.PathInContainer, "/devices/kvm")
	assert.Equal(t, device.CgroupPermissions, "rwm")
}

func TestParseDeviceStringThree(t *testing.T) {
	e := executor{}

	device, err := e.parseDeviceString("/dev/kvm:/devices/kvm:r")

	assert.NoError(t, err)
	assert.Equal(t, device.PathOnHost, "/dev/kvm")
	assert.Equal(t, device.PathInContainer, "/devices/kvm")
	assert.Equal(t, device.CgroupPermissions, "r")
}

func TestParseDeviceStringFour(t *testing.T) {
	e := executor{}

	_, err := e.parseDeviceString("/dev/kvm:/devices/kvm:r:oops")

	assert.Error(t, err)
}

func TestSplitService(t *testing.T) {
	e := executor{}

	tests := []struct {
		description string
		service     string
		version     string
		alias       string
		alternative string
	}{
		{"service", "service", "latest", "service", ""},
		{"service:version", "service", "version", "service", ""},
		{"namespace/service", "namespace/service", "latest", "namespace__service", "namespace-service"},
		{"namespace/service:version", "namespace/service", "version", "namespace__service", "namespace-service"},
	}

	for _, test := range tests {
		service, version, linkNames := e.splitServiceAndVersion(test.description)

		assert.Equal(t, test.service, service, "for", test.description)
		assert.Equal(t, test.version, version, "for", test.description)
		assert.Equal(t, test.alias, linkNames[0], "for", test.description)
		if test.alternative != "" {
			assert.Len(t, linkNames, 2, "for", test.description)
			assert.Equal(t, test.alternative, linkNames[1], "for", test.description)
		} else {
			assert.Len(t, linkNames, 1, "for", test.description)
		}
	}
}
