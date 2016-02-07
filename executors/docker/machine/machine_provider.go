package machine

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/docker"
	"io/ioutil"
	"os"
)

type machineState int

const (
	machineStateIdle machineState = iota
	machineStateAcquired
	machineStateCreating
	machineStateUsed
	machineStateRemoving
)

func (t machineState) String() string {
	switch t {
	case machineStateIdle:
		return "Idle"
	case machineStateAcquired:
		return "Acquired"
	case machineStateCreating:
		return "Creating"
	case machineStateUsed:
		return "Used"
	case machineStateRemoving:
		return "Removing"
	default:
		return "Unknown"
	}
}

func (t machineState) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

type machineDetails struct {
	Name      string
	Created   time.Time `yaml:"-"`
	Used      time.Time `yaml:"-"`
	UsedCount int
	State     machineState
	Reason    string
}

func (m *machineDetails) isUsed() bool {
	return m.State != machineStateIdle
}

func (m *machineDetails) match(machineFilter string) bool {
	var query string
	if n, _ := fmt.Sscanf(m.Name, machineFilter, &query); n != 1 {
		return false
	}
	return true
}

type machinesDetails map[string]*machineDetails

type machineProvider struct {
	machine  docker_helpers.Machine
	details  machinesDetails
	lock     sync.RWMutex
	executor string
}

func machineFormat(runner string, template string) string {
	if runner != "" {
		return "runner-" + runner + "-" + template
	}
	return template
}

func newMachineName(machineFilter string) string {
	t := time.Now().Unix()
	r := rand.Int31()
	return fmt.Sprintf(machineFilter, fmt.Sprintf("%d-%d", t, r))
}

func (m *machineProvider) filter(config *common.RunnerConfig) string {
	return machineFormat(config.ShortDescription(), config.Machine.MachineName)
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
		details.Used = time.Now()
	}

	return details
}

func (m *machineProvider) create(config *common.RunnerConfig, state machineState) (details *machineDetails, errCh chan error) {
	name := newMachineName(m.filter(config))
	details = m.machineDetails(name, true)
	details.State = machineStateCreating
	errCh = make(chan error, 1)

	// Create machine asynchronously
	go func() {
		started := time.Now()
		err := m.machine.Create(config.Machine.MachineDriver, details.Name, config.Machine.MachineOptions...)
		for i := 0; i < 3 && err != nil; i++ {
			logrus.Warningln("Machine creation failed, trying to provision", err)
			time.Sleep(time.Second)
			err = m.machine.Provision(details.Name)
		}

		if err != nil {
			m.remove(details.Name, "Failed to create")
		} else {
			details.State = state
			logrus.WithField("time", time.Since(started)).
				WithField("name", details.Name).
				Infoln("Machine created")
		}
		errCh <- err
	}()
	return
}

func (m *machineProvider) useMachine(config *common.RunnerConfig) (details *machineDetails, err error) {
	machines, err := m.loadMachines(config)
	if err != nil {
		return
	}
	details = m.findFreeMachine(machines)
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
		time.Sleep(time.Second)
	}
	return
}

func (m *machineDetails) writeDebugInformation() {
	if logrus.GetLevel() < logrus.DebugLevel {
		return
	}

	var details struct {
		machineDetails
		Time       string
		CreatedAgo time.Duration
	}
	details.machineDetails = *m
	details.Time = time.Now().String()
	details.CreatedAgo = time.Since(m.Created)
	data := helpers.ToYAML(&details)
	ioutil.WriteFile("machines/"+details.Name+".yml", []byte(data), 0600)
}

func (m *machineProvider) remove(machineName string, reason ...interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()

	details, ok := m.details[machineName]
	if ok {
		details.Reason = fmt.Sprint(reason...)
		details.State = machineStateRemoving
		logrus.WithField("name", machineName).
			WithField("created", time.Since(details.Created)).
			WithField("used", time.Since(details.Used)).
			WithField("reason", details.Reason).
			Warningln("Removing machine")
		details.Used = time.Now()
		details.writeDebugInformation()
	}

	go func() {
		m.machine.Remove(machineName)
		m.lock.Lock()
		defer m.lock.Unlock()
		delete(m.details, machineName)
	}()
}

type machinesData struct {
	Acquired int
	Creating int
	Idle     int
	Used     int
	Removing int
}

func (d *machinesData) Available() int {
	return d.Acquired + d.Creating + d.Idle
}

func (d *machinesData) Total() int {
	return d.Acquired + d.Creating + d.Idle + d.Used
}

func (d *machinesData) Add(state machineState) {
	switch state {
	case machineStateIdle:
		d.Idle++

	case machineStateCreating:
		d.Creating++

	case machineStateAcquired:
		d.Acquired++

	case machineStateUsed:
		d.Used++

	case machineStateRemoving:
		d.Removing++
	}
}

func (d *machinesData) Fields() logrus.Fields {
	return logrus.Fields{
		"used":     d.Used,
		"idle":     d.Idle,
		"total":    d.Total(),
		"creating": d.Creating,
		"removing": d.Removing,
	}
}

func (d *machinesData) writeDebugInformation() {
	if logrus.GetLevel() < logrus.DebugLevel {
		return
	}

	file, err := os.OpenFile("machines.csv", os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer file.Close()
	fmt.Fprintln(file, "time", time.Now(), "acquired", d.Acquired, "creating", d.Creating, "idle", d.Idle, "used", d.Used, "removing", d.Removing)
}

func (m *machineProvider) updateIdleMachine(config *common.RunnerConfig, data *machinesData, details *machineDetails) error {
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

func (m *machineProvider) updateMachines(machines []string, config *common.RunnerConfig) (data machinesData, err error) {
	for _, name := range machines {
		details := m.machineDetails(name, false)
		err := m.updateIdleMachine(config, &data, details)
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

func (m *machineProvider) findFreeMachine(machines []string) (details *machineDetails) {
	// Enumerate all machines
	for _, name := range machines {
		details := m.machineDetails(name, true)
		if details == nil {
			continue
		}

		// Check if node is running
		canConnect := m.machine.CanConnect(name)
		if !canConnect {
			m.remove(name)
			continue
		}
		return details
	}

	return nil
}

func (m *machineProvider) loadMachines(config *common.RunnerConfig) ([]string, error) {
	// Find a new machine{
	return m.machine.List(m.filter(config))
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

	// Update a list of currently configured machines
	machinesData, err := m.updateMachines(machines, config)
	if err != nil {
		return
	}

	// Pre-create machines
	m.createMachines(config, &machinesData)

	logrus.WithFields(machinesData.Fields()).
		WithField("runner", config.ShortDescription()).
		WithField("minIdleCount", config.Machine.IdleCount).
		WithField("maxMachines", config.Limit).
		WithField("time", time.Now()).
		Debugln("Docker Machine Details")
	machinesData.writeDebugInformation()

	// Try to find a free machine
	details := m.findFreeMachine(machines)
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
	}

	// Get machine credentials
	dc, err := m.machine.Credentials(details.Name)
	if err != nil {
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
	details.UsedCount++

	// Return details only if this is a new instance
	if data != details {
		newData = details
	}
	return
}

func (m *machineProvider) Release(config *common.RunnerConfig, data common.ExecutorData) error {
	// Release machine
	details, ok := data.(*machineDetails)
	if ok {
		details.Used = time.Now()
		details.State = machineStateIdle
	}
	return nil
}

func (m *machineProvider) CanCreate() bool {
	factory := common.GetExecutor(m.executor)
	if factory != nil {
		return factory.CanCreate()
	}
	return false
}

func (m *machineProvider) GetFeatures(features *common.FeaturesInfo) {
	factory := common.GetExecutor(m.executor)
	if factory != nil {
		factory.GetFeatures(features)
	}
}

func (m *machineProvider) Create() common.Executor {
	executor := common.NewExecutor(m.executor)
	if executor == nil {
		return nil
	}
	return &machineExecutor{
		provider:      m,
		otherExecutor: executor,
	}
}

func newMachineProvider(executor string) *machineProvider {
	return &machineProvider{
		details:  make(machinesDetails),
		machine:  docker_helpers.NewMachineCommand(),
		executor: executor,
	}
}
