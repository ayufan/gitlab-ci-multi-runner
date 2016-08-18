// Copyright 2015 Daniel Theophanes.
// Use of this source code is governed by a zlib-style
// license that can be found in the LICENSE file.

package service

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"
)

type sysv struct {
	i Interface
	*Config
}

func newSystemVService(i Interface, c *Config) (Service, error) {
	s := &sysv{
		i:      i,
		Config: c,
	}

	return s, nil
}

func isDebianSysv() bool {
	if _, err := os.Stat("/lib/lsb/init-functions"); err != nil {
		return false
	}
	if _, err := os.Stat("/sbin/start-stop-daemon"); err != nil {
		return false
	}
	return true
}

func isRedhatSysv() bool {
	if _, err := os.Stat("/etc/rc.d/init.d/functions"); err != nil {
		return false
	}
	return true
}

func (s *sysv) String() string {
	if len(s.DisplayName) > 0 {
		return s.DisplayName
	}
	return s.Name
}

var errNoUserServiceSystemV = errors.New("User services are not supported on SystemV.")

func (s *sysv) configPath() (cp string, err error) {
	if s.Option.bool(optionUserService, optionUserServiceDefault) {
		err = errNoUserServiceSystemV
		return
	}
	cp = "/etc/init.d/" + s.Config.Name
	return
}

func (s *sysv) template() (*template.Template, error) {
	script := sysvScript
	if isDebianSysv() {
		script = sysvDebianScript
	} else if isRedhatSysv() {
		script = sysvRedhatScript
	} else {
		return nil, errors.New("Not supported system")
	}
	return template.Must(template.New("").Funcs(tf).Parse(script)), nil
}

func (s *sysv) Install() error {
	confPath, err := s.configPath()
	if err != nil {
		return err
	}
	_, err = os.Stat(confPath)
	if err == nil {
		return fmt.Errorf("Init already exists: %s", confPath)
	}

	f, err := os.Create(confPath)
	if err != nil {
		return err
	}
	defer f.Close()

	path, err := s.execPath()
	if err != nil {
		return err
	}

	var to = &struct {
		*Config
		Path string
	}{
		s.Config,
		path,
	}

	template, err := s.template()
	if err != nil {
		return err
	}
	err = template.Execute(f, to)
	if err != nil {
		return err
	}

	if err = os.Chmod(confPath, 0755); err != nil {
		return err
	}
	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err = os.Symlink(confPath, "/etc/rc"+i+".d/S50"+s.Name); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err = os.Symlink(confPath, "/etc/rc"+i+".d/K02"+s.Name); err != nil {
			continue
		}
	}

	return nil
}

func (s *sysv) Uninstall() error {
	cp, err := s.configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(cp); err != nil {
		return err
	}
	return nil
}

func (s *sysv) Logger(errs chan<- error) (Logger, error) {
	if system.Interactive() {
		return ConsoleLogger, nil
	}
	return s.SystemLogger(errs)
}
func (s *sysv) SystemLogger(errs chan<- error) (Logger, error) {
	return newSysLogger(s.Name, errs)
}

func (s *sysv) Run() (err error) {
	err = s.i.Start(s)
	if err != nil {
		return err
	}

	s.Option.funcSingle(optionRunWait, func() {
		var sigChan = make(chan os.Signal, 3)
		signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
		<-sigChan
	})()

	return s.i.Stop(s)
}

func (s *sysv) Start() error {
	return run("service", s.Name, "start")
}

func (s *sysv) Stop() error {
	return run("service", s.Name, "stop")
}

func (s *sysv) Status() error {
	return checkStatus("service", []string{s.Name, "status"}, "is running", "unrecognized service")
}

func (s *sysv) Restart() error {
	err := s.Stop()
	if err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)
	return s.Start()
}

const sysvScript = `#!/bin/sh
# For RedHat and cousins:
# chkconfig: - 99 01
# description: {{.Description}}
# processname: {{.Path}}

### BEGIN INIT INFO
# Provides:          {{.Path}}
# Required-Start:    $local_fs $remote_fs $network $syslog
# Required-Stop:     $local_fs $remote_fs $network $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: {{.DisplayName}}
# Description:       {{.Description}}
### END INIT INFO

cmd="{{.Path}}{{range .Arguments}} {{.|cmd}}{{end}}"

name="{{.Name}}"
pid_file="/var/run/$name.pid"
stdout_log="/var/log/$name.log"
stderr_log="/var/log/$name.err"

get_pid() {
    cat "$pid_file"
}

is_running() {
    [ -f "$pid_file" ] && ps $(get_pid) > /dev/null 2>&1
}

case "$1" in
    start)
        if is_running; then
            echo "Already started"
        else
            echo "Starting $name"
            {{if .WorkingDirectory}}cd '{{.WorkingDirectory}}'{{end}}
            $cmd >> "$stdout_log" 2>> "$stderr_log" &
            echo $! > "$pid_file"
            if ! is_running; then
                echo "Unable to start, see $stdout_log and $stderr_log"
                exit 1
            fi
        fi
    ;;
    stop)
        if is_running; then
            echo -n "Stopping $name.."
            kill $(get_pid)
            for i in {1..10}
            do
                if ! is_running; then
                    break
                fi
                echo -n "."
                sleep 1
            done
            echo
            if is_running; then
                echo "Not stopped; may still be shutting down or shutdown may have failed"
                exit 1
            else
                echo "Stopped"
                if [ -f "$pid_file" ]; then
                    rm "$pid_file"
                fi
            fi
        else
            echo "Not running"
        fi
    ;;
    restart)
        $0 stop
        if is_running; then
            echo "Unable to stop, will not attempt to start"
            exit 1
        fi
        $0 start
    ;;
    status)
        if is_running; then
            echo "Running"
        else
            echo "Stopped"
            exit 1
        fi
    ;;
    *)
    echo "Usage: $0 {start|stop|restart|status}"
    exit 1
    ;;
esac
exit 0
`

const sysvDebianScript = `#! /bin/bash

### BEGIN INIT INFO
# Provides:          {{.Path}}
# Required-Start:    $local_fs $remote_fs $network $syslog
# Required-Stop:     $local_fs $remote_fs $network $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: {{.DisplayName}}
# Description:       {{.Description}}
### END INIT INFO

DESC="{{.Description}}"
USER="{{.UserName}}"
NAME="{{.Name}}"
PIDFILE="/var/run/$NAME.pid"

# Read configuration variable file if it is present
[ -r /etc/default/$NAME ] && . /etc/default/$NAME

# Define LSB log_* functions.
. /lib/lsb/init-functions

## Check to see if we are running as root first.
if [ "$(id -u)" != "0" ]; then
    echo "This script must be run as root"
    exit 1
fi

do_start() {
  start-stop-daemon --start \
    {{if .ChRoot}}--chroot {{.ChRoot|cmd}}{{end}} \
    {{if .WorkingDirectory}}--chdir {{.WorkingDirectory|cmd}}{{end}} \
    {{if .UserName}} --chuid {{.UserName|cmd}}{{end}} \
    --pidfile "$PIDFILE" \
    --background \
    --make-pidfile \
    --exec {{.Path}} -- {{range .Arguments}} {{.|cmd}}{{end}}
}

do_stop() {
  start-stop-daemon --stop \
    {{if .UserName}} --chuid {{.UserName|cmd}}{{end}} \
    --pidfile "$PIDFILE" \
    --quiet
}

case "$1" in
  start)
    log_daemon_msg "Starting $DESC"
    do_start
    log_end_msg $?
    ;;
  stop)
    log_daemon_msg "Stopping $DESC"
    do_stop
    log_end_msg $?
    ;;
  restart)
    $0 stop
    $0 start
    ;;
  status)
    status_of_proc -p "$PIDFILE" "$DAEMON" "$DESC"
    ;;
  *)
    echo "Usage: sudo service $0 {start|stop|restart|status}" >&2
    exit 1
    ;;
esac

exit 0
`

const sysvRedhatScript = `#!/bin/sh
# For RedHat and cousins:
# chkconfig: - 99 01
# description: {{.Description}}
# processname: {{.Path}}
 
# Source function library.
. /etc/rc.d/init.d/functions

name="{{.Name}}"
desc="{{.Description}}"
user="{{.UserName}}"
cmd={{.Path}}
args="{{range .Arguments}} {{.|cmd}}{{end}}"
lockfile=/var/lock/subsys/$name
pidfile=/var/run/$name.pid

# Source networking configuration.
[ -r /etc/sysconfig/$name ] && . /etc/sysconfig/$name
 
start() {
    echo -n $"Starting $desc: "
    daemon \
        {{if .UserName}}--user=$user{{end}} \
        {{if .WorkingDirectory}}--chdir={{.WorkingDirectory|cmd}}{{end}} \
        "$cmd $args </dev/null >/dev/null 2>/dev/null & echo \$! > $pidfile"
    retval=$?
    [ $retval -eq 0 ] && touch $lockfile
    echo
    return $retval
}
 
stop() {
    echo -n $"Stopping $desc: "
    killproc -p $pidfile $cmd -TERM
    retval=$?
    [ $retval -eq 0 ] && rm -f $lockfile
    rm -f $pidfile
    echo
    return $retval
}
 
restart() {
    stop
    start
}
 
reload() {
    echo -n $"Reloading $desc: "
    killproc -p $pidfile $cmd -HUP
    RETVAL=$?
    echo
}
 
force_reload() {
    restart
}
 
rh_status() {
    status -p $pidfile $cmd
}
 
rh_status_q() {
    rh_status >/dev/null 2>&1
}
 
case "$1" in
    start)
        rh_status_q && exit 0
        $1
        ;;
    stop)
        rh_status_q || exit 0
        $1
        ;;
    restart)
        $1
        ;;
    reload)
        rh_status_q || exit 7
        $1
        ;;
    force-reload)
        force_reload
        ;;
    status)
        rh_status
        ;;
    condrestart|try-restart)
        rh_status_q || exit 0
        ;;
    *)
        echo $"Usage: $0 {start|stop|status|restart|condrestart|try-restart|reload|force-reload}"
        exit 2
esac
`
