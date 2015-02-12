package ssh

type SshConfig struct {
	User     string `toml:"user" json:"user"`
	Password string `toml:"password" json:"password"`
	Host     string `toml:"host" json:"host"`
	Port     string `toml:"port" json:"port"`
}
