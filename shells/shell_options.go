package shells

type archivingOptions struct {
	Untracked bool     `json:"untracked"`
	Paths     []string `json:"paths"`
	Name      string   `json:"name"`
	Key       string   `json:"key"`
}

type dependencies []string

func (m *dependencies) IsDependent(name string) bool {
	if m == nil {
		return true
	}
	for _, other := range *m {
		if other == name {
			return true
		}
	}
	return false
}

type shellOptions struct {
	Dependencies *dependencies     `json:"dependencies"`
	Cache        *archivingOptions `json:"cache"`
	Artifacts    *archivingOptions `json:"artifacts"`
}
