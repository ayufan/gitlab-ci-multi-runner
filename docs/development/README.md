# Development environment

## 1. Install dependencies and Go runtime

### For Debian/Ubuntu
```bash
apt-get install -y mercurial git-core wget make
wget https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go*-*.tar.gz
```

### For OSX using binary package
```bash
wget https://storage.googleapis.com/golang/go1.4.2.darwin-amd64-osx10.8.tar.gz
sudo tar -C /usr/local -xzf go*-*.tar.gz
```

### For OSX if you have brew.sh
```
brew install go
```

### For OSX using installation package
```
wget https://storage.googleapis.com/golang/go1.4.2.darwin-amd64-osx10.8.pkg
open go*-*.pkg
```

### For FreeBSD
```
pkg install go-1.4.2 gmake git mercurial
```

## 2. Configure Go

Add to `.profile` or `.bash_profile`:

```bash
export GOPATH=$HOME/Go
export PATH=$PATH:$GOPATH/bin:/usr/local/go/bin
```

Create new terminal session and create $GOPATH directory:

```
mkdir -p $GOPATH
```

## 3. Download runner sources

```
go get gitlab.com/gitlab-org/gitlab-ci-multi-runner
cd $GOPATH/src/gitlab.com/gitlab-org/gitlab-ci-multi-runner/
```

## 4. Install runner dependencies

This will download and restore all dependencies required to build runner:

```
make deps
```

**For FreeBSD use `gmake deps`**

## 5. Run runner

Normally you would use `gitlab-runner`, in order to compile and run Go source use go toolchain:

```
go run main.go
```

You can run runner in debug-mode:

```
go run --debug main.go
```

## 6. Compile and install runner binary

```
go build
go install
```

## 7. Congratulations!

You can start hacking GitLab-Runner code. If you are interested you can use Intellij IDEA Community Edition with [go-lang-idea-plugin](https://github.com/go-lang-plugin-org/go-lang-idea-plugin) to edit and debug code.

