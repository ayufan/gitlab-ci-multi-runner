## How to configure runner for GitLab CE integration tests (uses confined Docker executor)

### 1. Register the runner

The registration token can be found at `https://ci.gitlab.com/projects/:id/runners`.
You can export it as a variable and run the command below as is:

```bash
gitlab-ci-multi-runner register \
--non-interactive \
--url "https://ci.gitlab.com/" \
--registration-token "REGISTRATION_TOKEN" \
--description "gitlab-ce-ruby-2.1" \
--executor "docker" \
--docker-image ruby:2.1 --docker-mysql latest \
--docker-postgres latest --docker-redis latest
```

### 2. Add a job to test with

#### MySQL

Paste the snippet below at the jobs page to run the GitLab CE tests with MySQL:

```bash
wget -q http://ftp.de.debian.org/debian/pool/main/p/phantomjs/phantomjs_1.9.0-1+b1_amd64.deb
dpkg -i phantomjs_1.9.0-1+b1_amd64.deb

apt-get update -qq
apt-get install -y -qq libicu-dev libkrb5-dev cmake nodejs

bundle install --deployment --path /cache

cp config/gitlab.yml.example config/gitlab.yml

cp config/database.yml.mysql config/database.yml
sed -i 's/username:.*/username: root/g' config/database.yml
sed -i 's/password:.*/password:/g' config/database.yml
sed -i 's/# socket:.*/host: mysql/g' config/database.yml

cp config/resque.yml.example config/resque.yml
sed -i 's/localhost/redis/g' config/resque.yml

bundle exec rake db:create

bundle exec rake test_ci
```

#### PostgreSQL

Paste the snippet below at the jobs page to run the GitLab CE tests with PostgreSQL:

```bash
wget -q http://ftp.de.debian.org/debian/pool/main/p/phantomjs/phantomjs_1.9.0-1+b1_amd64.deb
dpkg -i phantomjs_1.9.0-1+b1_amd64.deb

apt-get update -qq
apt-get install -y -qq libicu-dev libkrb5-dev cmake nodejs

bundle install --deployment --path /cache

cp config/gitlab.yml.example config/gitlab.yml

cp config/database.yml.postgresql config/database.yml
sed -i 's/username:.*/username: postgres/g' config/database.yml
sed -i 's/password:.*/password:/g' config/database.yml
sed -i 's/pool:.*/&\n  host: postgres/g' config/database.yml

cp config/resque.yml.example config/resque.yml
sed -i 's/localhost/redis/g' config/resque.yml

bundle exec rake db:create

bundle exec rake test_ci
```

----

You now have a GitLab CE integration testing instance with bundle caching.
Push some commits to test it.

For [advanced setup](../configuration/advanced_setup.md), look into
`/home/gitlab_ci_multi_runner/config.toml` and tune it.
