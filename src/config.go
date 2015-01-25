package src

type RunnerConfig struct {
	Name			string			`json:"name"`
	URL				string			`json:"url"`
	Token			string			`json:"token"`
	Limit			int				`json:"limit"`
	Executor		string			`json:"executor"`
}

type Config struct {
	Concurrent		int				`json:"concurrent"`
	Runners			[]RunnerConfig	`json:"runners"`
}
