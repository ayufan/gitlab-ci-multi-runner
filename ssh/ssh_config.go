package ssh

type Config struct {
	User         *string `toml:"user" json:"user" long:"user" env:"SSH_USER" description:"User name"`
	Password     *string `toml:"password" json:"password" long:"password" env:"SSH_PASSWORD" description:"User password"`
	Host         *string `toml:"host" json:"host" long:"host" env:"SSH_HOST" description:"Remote host"`
	Port         *string `toml:"port" json:"port" long:"port" env:"SSH_PORT" description:"Remote host port"`
	IdentityFile *string `toml:"identity_file" json:"identity_file" long:"identity-file" env:"SSH_IDENTITY_FILE" description:"Identity file to be used"`
}
