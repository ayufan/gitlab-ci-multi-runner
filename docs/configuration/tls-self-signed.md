## The self-signed certificates or custom Certification Authorities

Since version 0.7.0 the GitLab Runner have allows to configure certificates that are used to verify TLS peer when connecting GitLab server.

**This allows to solve the `x509: certificate signed by unknown authority` problem when registering runner.**

The GitLab Runner provides these options:

1. **Default**: GitLab Runner reads system certificate store and verifies the GitLab server against the CA's stored in system.

2. GitLab Runner reads the PEM (**DER format is not supported**) certificate from predefined file:

        - `/etc/gitlab-runner/certs/hostname.crt` on *nix systems when gitlab-runner is executed as root.
        - `~/.gitlab-runner/certs/hostname.crt` on *nix systems when gitlab-runner is executed as non-root,
        - `./certs/hostname.crt` on other systems.
            
        If address of your server is: `https://my.gitlab.server.com:8443/`.
        Create the certificate file at: `/etc/gitlab-runner/certs/my.gitlab.server.com`. 

3. GitLab Runner exposes `tls-ca-file` option during registration and in [`config.toml`](advanced-configuration.md)
which allows you to specify custom file with certificates. This file will be read everytime when runner tries to
access the GitLab server.

4. GitLab Runner exposes `tls-skip-verify` option during registration and [`config.toml`](advanced-configuration.md)
which allows you to skip TLS verification when connecting to server.
**This approach is INSECURE! Use at your own risk!**
Anyone can eavesdrop your connection:

        - see the runner token which is used to authenticate against GitLab,
        - see tokens which are used to clone GitLab projects,
        - see the secure variables that are passed to runner.

### Git cloning

Currently the certificates are only used to verify connections between GitLab Runner and GitLab server.
This doesn't affect git commands (ie. `git clone`).
You still will see build errors due to TLS validation failure.

To have maximum security for your builds you should configure `GIT_SSL_CAINFO` variable.
It allows you to define the file containing the certificates to verify the peer with when fetching or pushing over HTTPS

The other options is to disable TLS verification, but as said before this approach is insecure. 
Add to your `.gitlab-ci.yml` this lines:
```
variables:
  GIT_SSL_NO_VERIFY: "1"
```
