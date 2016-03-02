package machine

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"os"
	"time"
)

type machinesData struct {
	Runner   string
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
	return d.Acquired + d.Creating + d.Idle + d.Used + d.Removing
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
		"runner":   d.Runner,
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
	fmt.Fprintln(file,
		"time", time.Now(),
		"runner", d.Runner,
		"acquired", d.Acquired,
		"creating", d.Creating,
		"idle", d.Idle,
		"used", d.Used,
		"removing", d.Removing)
}
