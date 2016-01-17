package helpers

import (
	"github.com/docker/docker/pkg/homedir"
	"os"
	"os/user"
)

func GetCurrentUserName() string {
	user, _ := user.Current()
	if user != nil {
		return user.Username
	}
	return ""
}

func GetCurrentWorkingDirectory() string {
	dir, err := os.Getwd()
	if err == nil {
		return dir
	}
	return ""
}

func GetHomeDir() string {
	return homedir.Get()
}
