### How to configure runner for GitLab CI integration tests (uses confined Docker executor)

1. Run setup
    ```bash
    $ gitlab-ci-multi-runner setup \
      --non-interactive \
      --url "https://ci.gitlab.com/" \
      --registration-token "REGISTRATION_TOKEN" \
      --description "gitlab-ci-ruby-2.1" \
      --executor "docker" \
      --docker-image ruby:2.1 --docker-mysql latest \
      --docker-postgres latest --docker-redis latest
    ```

1. Add job to test with MySQL
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

1. Add job to test with PostgreSQL
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

1. You now have GitLab CI integration testing instance with bundle caching.
   Push some commits to test it.

1. For [advanced setup](../configuration/advanced_setup.md), look into
  `/home/gitlab_ci_multi_runner/config.toml` and tune it.
