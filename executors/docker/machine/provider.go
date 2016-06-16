package machine

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/docker"
)

type machineProvider struct {
	machine     docker_helpers.Machine
	details     machinesDetails
	lock        sync.RWMutex
	acquireLock sync.Mutex
	// provider stores a real executor that is used to start run the builds
	provider common.ExecutorProvider
}

func (m *machineProvider) machineDetails(name string, acquire bool) *machineDetails {
	m.lock.Lock()
	defer m.lock.Unlock()

	details, ok := m.details[name]
	if !ok {
		details = &machineDetails{
			Name:    name,
			Created: time.Now(),
			Used:    time.Now(),
			State:   machineStateIdle,
		}
		m.details[name] = details
	}

	if acquire {
		if details.isUsed() {
			return nil
		}
		details.State = machineStateAcquired
	}

	return details
}

func (m *machineProvider) create(config *common.RunnerConfig, state machineState) (details *machineDetails, errCh chan error) {
	name := newMachineName(machineFilter(config))
	details = m.machineDetails(name, true)
	details.State = machineStateCreating
	errCh = make(chan error, 1)

	// Create machine asynchronously
	go func() {
		started := time.Now()
		err := m.machine.Create(config.Machine.MachineDriver, details.Name, config.Machine.MachineOptions...)
		for i := 0; i < 3 && err != nil; i++ {
			logrus.WithField("name", details.Name).
				Warningln("Machine creation failed, trying to provision", err)
			time.Sleep(provisionRetryInterval)
			err = m.machine.Provision(details.Name)
		}

		if err != nil {
			m.remove(details.Name, "Failed to create")
		} else {
			details.State = state
			details.Used = time.Now()
			logrus.WithField("time", time.Since(started)).
				WithField("name", details.Name).
				Infoln("Machine created")
		}
		errCh <- err
	}()
	return
}

func (m *machineProvider) findFreeMachine(machines ...string) (details *machineDetails) {
	// Enumerate all machines
	for _, name := range machines {
		details := m.machineDetails(name, true)
		if details == nil {
			continue
		}

		// Check if node is running
		canConnect := m.machine.CanConnect(name)
		if !canConnect {
			m.remove(name, "machine is unavailable")
			continue
		}
		return details
	}

	return nil
}

func (m *machineProvider) useMachine(config *common.RunnerConfig) (details *machineDetails, err error) {
	machines, err := m.loadMachines(config)
	if err != nil {
		return
	}
	details = m.findFreeMachine(machines...)
	if details == nil {
		var errCh chan error
		details, errCh = m.create(config, machineStateAcquired)
		err = <-errCh
	}
	return
}

func (m *machineProvider) retryUseMachine(config *common.RunnerConfig) (details *machineDetails, err error) {
	// Try to find a machine
	for i := 0; i < 3; i++ {
		details, err = m.useMachine(config)
		if err == nil {
			break
		}
		time.Sleep(provisionRetryInterval)
	}
	return
}

func (m *machineProvider) finalizeRemoval(details *machineDetails) {
	for {
		if !m.machine.Exist(details.Name) {
			logrus.WithField("name", details.Name).
				WithField("created", time.Since(details.Created)).
				WithField("used", time.Since(details.Used)).
				WithField("reason", details.Reason).
				Warningln("Skipping machine removal, because it doesn't exist")
			break
		}

		err := m.machine.Remove(details.Name)
		if err == nil {
			break
		}
		time.Sleep(30 * time.Second)
		logrus.WithField("name", details.Name).
			WithField("created", time.Since(details.Created)).
			WithField("used", time.Since(details.Used)).
			WithField("reason", details.Reason).
			Warningln("Retrying removal")
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.details, details.Name)
}

func (m *machineProvider) remove(machineName string, reason ...interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()

	details, _ := m.details[machineName]
	if details == nil {
		return
	}

	details.Reason = fmt.Sprint(reason...)
	details.State = machineStateRemoving
	logrus.WithField("name", machineName).
		WithField("created", time.Since(details.Created)).
		WithField("used", time.Since(details.Used)).
		WithField("reason", details.Reason).
		Warningln("Removing machine")
	details.Used = time.Now()
	details.writeDebugInformation()

	go m.finalizeRemoval(details)
}

func (m *machineProvider) updateMachine(config *common.RunnerConfig, data *machinesData, details *machineDetails) error {
	if details.State != machineStateIdle {
		return nil
	}

	if config.Machine.MaxBuilds > 0 && details.UsedCount >= config.Machine.MaxBuilds {
		// Limit number of builds
		return errors.New("Too many builds")
	}

	if data.Total() >= config.Limit && config.Limit > 0 {
		// Limit maximum number of machines
		return errors.New("Too many machines")
	}

	if time.Since(details.Used) > time.Second*time.Duration(config.Machine.IdleTime) {
		if data.Idle >= config.Machine.IdleCount {
			// Remove machine that are way over the idle time
			return errors.New("Too many idle machines")
		}
	}
	return nil
}

func (m *machineProvider) updateMachines(machines []string, config *common.RunnerConfig) (data machinesData) {
	data.Runner = config.ShortDescription()

	for _, name := range machines {
		details := m.machineDetails(name, false)
		err := m.updateMachine(config, &data, details)
		if err != nil {
			m.remove(details.Name, err)
		}

		data.Add(details.State)
	}
	return
}

func (m *machineProvider) createMachines(config *common.RunnerConfig, data *machinesData) {
	// Create a new machines and mark them as Idle
	for {
		if data.Available() >= config.Machine.IdleCount {
			// Limit maximum number of idle machines
			break
		}
		if data.Total() >= config.Limit && config.Limit > 0 {
			// Limit maximum number of machines
			break
		}
		m.create(config, machineStateIdle)
		data.Creating++
	}
}

func (m *machineProvider) loadMachines(config *common.RunnerConfig) ([]string, error) {
	// Find a new machine
	return m.machine.List(machineFilter(config))
}

func (m *machineProvider) Acquire(config *common.RunnerConfig) (data common.ExecutorData, err error) {
	if config.Machine == nil || config.Machine.MachineName == "" {
		err = fmt.Errorf("Missing Machine options")
		return
	}

	machines, err := m.loadMachines(config)
	if err != nil {
		return
	}

	// Lock updating machines, because two Acquires can be run at the same time
	m.acquireLock.Lock()

	// Update a list of currently configured machines
	machinesData := m.updateMachines(machines, config)

	// Pre-create machines
	m.createMachines(config, &machinesData)

	m.acquireLock.Unlock()

	logrus.WithFields(machinesData.Fields()).
		WithField("runner", config.ShortDescription()).
		WithField("minIdleCount", config.Machine.IdleCount).
		WithField("maxMachines", config.Limit).
		WithField("time", time.Now()).
		Debugln("Docker Machine Details")
	machinesData.writeDebugInformation()

	// Try to find a free machine
	details := m.findFreeMachine(machines...)
	if details != nil {
		data = details
		return
	}

	// If we have a free machines we can process a build
	if config.Machine.IdleCount != 0 && machinesData.Idle == 0 {
		err = errors.New("No free machines that can process builds")
	}
	return
}

func (m *machineProvider) Use(config *common.RunnerConfig, data common.ExecutorData) (newConfig common.RunnerConfig, newData common.ExecutorData, err error) {
	// Find a new machine
	details, _ := data.(*machineDetails)
	if details == nil {
		details, err = m.retryUseMachine(config)
		if err != nil {
			return
		}

		// Return details only if this is a new instance
		newData = details
	}

	// Get machine credentials
	dc, err := m.machine.Credentials(details.Name)
	if err != nil {
		if newData != nil {
			m.Release(config, newData)
		}
		return
	}

	// Create shallow copy of config and store in it docker credentials
	newConfig = *config
	newConfig.Docker = &common.DockerConfig{}
	if config.Docker != nil {
		*newConfig.Docker = *config.Docker
	}
	newConfig.Docker.DockerCredentials = dc

	// Mark machine as used
	details.State = machineStateUsed
	return
}

func (m *machineProvider) Release(config *common.RunnerConfig, data common.ExecutorData) error {
	// Release machine
	details, ok := data.(*machineDetails)
	if ok {
		// Mark last used time when is Used
		if details.State == machineStateUsed {
			details.Used = time.Now()
			details.UsedCount++
		}
		details.State = machineStateIdle
	}
	return nil
}

func (m *machineProvider) CanCreate() bool {
	return m.provider.CanCreate()
}

func (m *machineProvider) GetFeatures(features *common.FeaturesInfo) {
	m.provider.GetFeatures(features)
}

func (m *machineProvider) Create() common.Executor {
	return &machineExecutor{
		provider: m,
	}
}

func newMachineProvider(executor string) *machineProvider {
	provider := common.GetExecutor(executor)
	if provider == nil {
		logrus.Panicln("Missing", executor)
	}

	return &machineProvider{
		details:  make(machinesDetails),
		machine:  docker_helpers.NewMachineCommand(),
		provider: provider,
	}
}
