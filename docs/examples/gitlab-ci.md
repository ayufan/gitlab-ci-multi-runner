## How to configure runner for GitLab CI integration tests (uses confined Docker executor)

### 1. Run the setup

The registration token can be found at `https://ci.gitlab.com/projects/:id/runners`.
You can export it as a variable and run the command below as is:

```bash
gitlab-ci-multi-runner setup \
--non-interactive \
--url "https://ci.gitlab.com/" \
--registration-token "REGISTRATION_TOKEN" \
--description "gitlab-ci-ruby-2.1" \
--executor "docker" \
--docker-image ruby:2.1 --docker-mysql latest \
--docker-postgres latest --docker-redis latest
```

### 2. Add job to test with

#### MySQL

Paste the snippet below at the jobs page to run the GitLab CE tests with MySQL:

```bash
wget -q http://ftp.de.debian.org/debian/pool/main/p/phantomjs/phantomjs_1.9.0-1+b1_amd64.deb
dpkg -i phantomjs_1.9.0-1+b1_amd64.deb

apt-get update -qq
apt-get install -qq nodejs

bundle install --deployment --path /cache

cp config/application.yml.example config/application.yml

cp config/database.yml.mysql config/database.yml
sed -i 's/username:.*/username: root/g' config/database.yml
sed -i 's/password:.*/password:/g' config/database.yml
sed -i 's/# socket:.*/host: mysql/g' config/database.yml

cp config/resque.yml.example config/resque.yml
sed -i 's/localhost/redis/g' config/resque.yml

bundle exec rake db:create
bundle exec rake db:setup
bundle exec rake spec
```

#### PostgreSQL

Paste the snippet below at the jobs page to run the GitLab CE tests with PostgresSQL:

```bash
wget -q http://ftp.de.debian.org/debian/pool/main/p/phantomjs/phantomjs_1.9.0-1+b1_amd64.deb
dpkg -i phantomjs_1.9.0-1+b1_amd64.deb

apt-get update -qq
apt-get install -qq nodejs

bundle install --deployment --path /cache

cp config/application.yml.example config/application.yml

cp config/database.yml.postgresql config/database.yml
sed -i 's/username:.*/username: postgres/g' config/database.yml
sed -i 's/password:.*/password:/g' config/database.yml
sed -i 's/# socket:.*/host: postgres/g' config/database.yml

cp config/resque.yml.example config/resque.yml
sed -i 's/localhost/redis/g' config/resque.yml

bundle exec rake db:create
bundle exec rake db:setup
bundle exec rake spec
```

----

You now have GitLab CI integration testing instance with bundle caching.
Push some commits to test it.

For [advanced setup](../configuration/advanced_setup.md), look into
`/home/gitlab_ci_multi_runner/config.toml` and tune it.
