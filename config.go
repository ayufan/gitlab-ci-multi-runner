package main

type RunnerConfig struct {
	Name			string			`json:"name"`
	URL				string			`json:"url"`
	Token			string			`json:"token"`
	Limit			int				`json:"limit"`
}

type Config struct {
	Concurrent		int				`json:"concurrent"`
	Runners			[]RunnerConfig	`json:"runners"`
}
