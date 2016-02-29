package helpers

import (
	"github.com/docker/docker/pkg/homedir"
	"os"
)

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
