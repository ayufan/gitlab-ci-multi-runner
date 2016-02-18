package machine

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
