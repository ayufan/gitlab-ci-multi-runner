// Copyright 2015 Daniel Theophanes.
// Use of this source code is governed by a zlib-style
// license that can be found in the LICENSE file.

package service

import (
	"log"
	"os"
	"runtime"
	"testing"
)

const runAsServiceArg = "RunThisAsService"

var sc = &Config{
	Name:      "go_service_test",
	Arguments: []string{runAsServiceArg},
	Option: KeyValue{
		"UserService": userService(),
	},
}

func userService() bool {
	if runtime.GOOS == "darwin" {
		return true
	} else {
		return false
	}
}

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == runAsServiceArg {
		runService()
		return
	}
	os.Exit(m.Run())
}

func TestInstallRunRestartStopRemove(t *testing.T) {
	p := &program{}
	s, err := New(p, sc)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Uninstall()

	err = s.Status()
	if err != ErrServiceIsNotInstalled {
		t.Fatal("status", err)
	}

	err = s.Install()
	if err != nil {
		t.Fatal("install", err)
	}
	defer s.Uninstall()

	err = s.Status()
	if err != ErrServiceIsNotRunning {
		t.Fatal("status", err)
	}

	err = s.Start()
	if err != nil {
		t.Fatal("start", err)
	}
	err = s.Restart()
	if err != nil {
		t.Fatal("restart", err)
	}
	err = s.Stop()
	if err != nil {
		t.Fatal("stop", err)
	}
	err = s.Status()
	if err != ErrServiceIsNotRunning {
		t.Fatal("status", err)
	}
	err = s.Uninstall()
	if err != nil {
		t.Fatal("uninstall", err)
	}
	err = s.Status()
	if err != ErrServiceIsNotInstalled {
		t.Fatal("status", err)
	}
}

func runService() {
	p := &program{}
	s, err := New(p, sc)
	if err != nil {
		log.Fatal(err)
	}
	err = s.Run()
	if err != nil {
		log.Fatal(err)
	}
}

type program struct{}

func (p *program) Start(s Service) error {
	go p.run()
	return nil
}
func (p *program) run() {
	// Do work here
}
func (p *program) Stop(s Service) error {
	return nil
}
