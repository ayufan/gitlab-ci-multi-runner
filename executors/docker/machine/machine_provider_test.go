package machine

import (
	"errors"
	"strings"
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/docker"
)

var machineDefaultConfig = &common.RunnerConfig{
	RunnerSettings: common.RunnerSettings{
		Machine: &common.DockerMachine{
			MachineName: "test-machine-%s",
		},
	},
}

var machineCreateFail = &common.RunnerConfig{
	RunnerSettings: common.RunnerSettings{
		Machine: &common.DockerMachine{
			MachineName: "create-fail-%s",
		},
	},
}

func createMachineConfig(idleCount int, idleTime int) *common.RunnerConfig {
	return &common.RunnerConfig{
		RunnerSettings: common.RunnerSettings{
			Machine: &common.DockerMachine{
				MachineName: "test-machine-%s",
				IdleCount:   idleCount,
				IdleTime:    idleTime,
			},
		},
	}
}

type testMachine struct {
	machines []string
}

func (m *testMachine) Create(driver, name string, opts ...string) error {
	if strings.Contains(name, "create-fail") {
		return errors.New("Failed to create")
	}
	m.machines = append(m.machines, name)
	return nil
}

func (m *testMachine) Remove(name string) error {
	var machines []string
	for _, machine := range m.machines {
		if machine != name {
			machines = append(machines, machine)
		}
	}
	m.machines = machines
	return nil
}

func (m *testMachine) List(nodeFilter string) (machines []string, err error) {
	return m.machines, nil
}

func (m *testMachine) CanConnect(name string) bool {
	if name == "no-can-connect" {
		return false
	}
	return true
}

func (m *testMachine) Credentials(name string) (dc docker_helpers.DockerCredentials, err error) {
	if name == "no-connect" {
		err = errors.New("Failed to connect")
	}
	return
}

func testMachineProvider(machine ...string) *machineProvider {
	return &machineProvider{
		details: make(machinesDetails),
		machine: &testMachine{
			machines: machine,
		},
	}
}

func TestMachineDetailsUsed(t *testing.T) {
	d := machineDetails{}
	d.State = machineStateIdle
	assert.False(t, d.isUsed())
	d.State = machineStateCreating
	assert.True(t, d.isUsed())
	d.State = machineStateUsed
	assert.True(t, d.isUsed())
	d.State = machineStateRemoving
	assert.True(t, d.isUsed())
}

func TestMachineNewName(t *testing.T) {
	a := newMachineName("machine-template-%s")
	b := newMachineName("machine-template-%s")
	assert.NotEqual(t, a, b)
}

func TestMachineDetailsMatcher(t *testing.T) {
	d := machineDetails{Name: newMachineName("machine-template-%s")}
	assert.True(t, d.match("machine-template-%s"))
	assert.False(t, d.match("machine-other-template-%s"))
}

func TestMachineFindNew(t *testing.T) {
	p := testMachineProvider()
	details := p.findNew("%s")
	assert.Nil(t, details, "no machines to return")

	p = testMachineProvider("no-can-connect")
	details = p.findNew("%s")
	assert.Nil(t, details, "the CanConnect should fail")

	p = testMachineProvider("machine1", "machine2")
	details = p.findNew("%s")
	assert.NotNil(t, details, "to return machine from list")
	assert.Equal(t, machineStateUsed, details.State, "machine should be used, the exclusive access should be taken by findNew")

	details2 := p.findNew("%s")
	assert.NotNil(t, details2, "to return next machine from list")
	assert.NotEqual(t, details2, details, "to return different machine")

	details3 := p.findNew("%s")
	assert.Nil(t, details3, "no more machines on list")

	details.State = machineStateIdle
	details4 := p.findNew("%s")
	assert.NotNil(t, details4, "to return machine from list")
	assert.Equal(t, details4, details, "to return first machine from list")
	assert.NotEqual(t, details.Used, details.Created, "to mark machine as used")
}

func TestMachineCreationAndRemoval(t *testing.T) {
	p := testMachineProvider()
	details, errCh := p.create(machineDefaultConfig, machineStateUsed)
	assert.NotNil(t, details)
	assert.NoError(t, <-errCh)
	assert.Equal(t, machineStateUsed, details.State)
	assert.NotNil(t, p.details[details.Name])

	details2, errCh := p.create(machineCreateFail, machineStateUsed)
	assert.NotNil(t, details2)
	assert.Error(t, <-errCh)
	assert.Equal(t, machineStateRemoving, details2.State)

	p.remove(details.Name)
	assert.Equal(t, machineStateRemoving, details.State)
}

func TestMachineAcquireAndRelease(t *testing.T) {
	p := testMachineProvider("test-machine")

	_, machine, err := p.acquire(machineDefaultConfig)
	assert.NoError(t, err)
	assert.Equal(t, "test-machine", machine, "acquire already created machine")

	_, machine2, err := p.acquire(machineDefaultConfig)
	assert.NoError(t, err)
	assert.NotEqual(t, "test-machine", machine2, "create a new machine")

	_, _, err = p.acquire(machineCreateFail)
	assert.Error(t, err, "fail to create machine")

	p.release(machine2)

	_, machine4, err := p.acquire(machineDefaultConfig)
	assert.NoError(t, err)
	assert.Equal(t, machine2, machine4, "Acquire previously created machine")
}

func TestMachineMaxBuilds(t *testing.T) {
	p := testMachineProvider()

	details, errCh := p.create(machineDefaultConfig, machineStateIdle)
	assert.NoError(t, <-errCh, "machine creation should not fail")

	_, errCh = p.create(machineDefaultConfig, machineStateIdle)
	assert.NoError(t, <-errCh, "machine creation should not fail")

	_, machine, err := p.acquire(machineDefaultConfig)
	assert.NoError(t, err)
	assert.Equal(t, machine, details.Name, "acquire created machine")

	p.release(machine)

	config := createMachineConfig(5, 0)
	config.Machine.MaxBuilds = 1
	err = p.Update(config)
	assert.NoError(t, err, "provider should not fail since we have one idle machine")
	assert.Equal(t, machineStateRemoving, details.State, "provider should get removed due to too many builds")
	assert.Equal(t, "Too many builds", details.Reason, "provider should get removed due to too many builds")
}

func TestMachineIdleLimits(t *testing.T) {
	p := testMachineProvider()

	config := createMachineConfig(2, 1)
	details, errCh := p.create(config, machineStateIdle)
	assert.NoError(t, <-errCh, "machine creation should not fail")

	err := p.Update(config)
	assert.NoError(t, err)
	assert.Equal(t, machineStateIdle, details.State, "machine should not be removed, because is still in idle time")

	config = createMachineConfig(2, 0)
	err = p.Update(config)
	assert.NoError(t, err)
	assert.Equal(t, machineStateIdle, details.State, "machine should not be removed, because no more than two idle")

	config = createMachineConfig(0, 0)
	err = p.Update(config)
	assert.NoError(t, err)
	assert.Equal(t, machineStateRemoving, details.State, "machine should not be removed, because no more than two idle")
	assert.Equal(t, "Too many idle machines", details.Reason)
}

func TestMachineOnDemandMode(t *testing.T) {
	p := testMachineProvider()

	config := createMachineConfig(0, 1)
	err := p.Update(config)
	assert.NoError(t, err)
}

func countIdleMachines(p *machineProvider) (count int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, details := range p.details {
		if details.State == machineStateIdle {
			count++
		}
	}
	return
}

func assertIdleMachines(t *testing.T, p *machineProvider, expected int, msgAndArgs ...interface{}) bool {
	var idle int
	for i := 0; i < 10; i++ {
		idle = countIdleMachines(p)

		if expected == idle {
			return true
		}

		time.Sleep(50 * time.Microsecond)
	}

	result := fmt.Sprintf("should have %d idle, but has %d", expected, idle)
	assert.Fail(t, result, msgAndArgs...)
	return false
}

func TestMachinePreCreateMode(t *testing.T) {
	p := testMachineProvider()

	config := createMachineConfig(1, 0)
	err := p.Update(config)
	assert.Error(t, err, "it should fail with message that currently there's no free machines")
	assertIdleMachines(t, p, 1, "it should contain exactly one machine")

	err = p.Update(config)
	assert.NoError(t, err, "it should be ready to process builds")
	assertIdleMachines(t, p, 1)

	config = createMachineConfig(2, 0)
	err = p.Update(config)
	assert.NoError(t, err)
	assertIdleMachines(t, p, 2, "it should start creating a second machine")

	config = createMachineConfig(1, 0)
	err = p.Update(config)
	assert.NoError(t, err)
	assertIdleMachines(t, p, 1, "it should leave single machine")

	_, _, err = p.acquire(config)
	assert.NoError(t, err, "we should acquire single machine")

	err = p.Update(config)
	assert.Error(t, err, "it should fail with message that currently there's no free machines")
	assertIdleMachines(t, p, 1, "it should leave one idle")
}

func TestMachineList(t *testing.T) {
	p := testMachineProvider("machine1", "machine2")
	config := createMachineConfig(1, 0)
	err := p.Update(config)
	assert.NoError(t, err, "it should have some machines")
}
