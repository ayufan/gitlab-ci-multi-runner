package ssh

type Config struct {
	User         *string `toml:"user" json:"user"`
	Password     *string `toml:"password" json:"password"`
	Host         *string `toml:"host" json:"host"`
	Port         *string `toml:"port" json:"port"`
	IdentityFile *string `toml:"identity_file" json:"identity_file"`
}
