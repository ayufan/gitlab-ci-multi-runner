package src

type RunnerConfig struct {
	Name     string `toml:"name",omitempty`
	URL      string `toml:"url"`
	Token    string `toml:"token"`
	Limit    int    `toml:"limit",omitempty`
	Executor string `toml:"executor",omitempty`
}

type Config struct {
	Concurrent int             `toml:"concurrent"`
	Runners    []*RunnerConfig `toml:"runners"`
}

func (c RunnerConfig) ShortDescription() string {
	return c.Token[0:8]
}
