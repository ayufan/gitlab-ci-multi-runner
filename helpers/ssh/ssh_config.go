package ssh

type Config struct {
	User         string `toml:"user,omitempty" json:"user" long:"user" env:"SSH_USER" description:"User name"`
	Password     string `toml:"password,omitempty" json:"password" long:"password" env:"SSH_PASSWORD" description:"User password"`
	Host         string `toml:"host,omitempty" json:"host" long:"host" env:"SSH_HOST" description:"Remote host"`
	Port         string `toml:"port,omitempty" json:"port" long:"port" env:"SSH_PORT" description:"Remote host port"`
	IdentityFile string `toml:"identity_file,omitempty" json:"identity_file" long:"identity-file" env:"SSH_IDENTITY_FILE" description:"Identity file to be used"`
}
