package helpers

import (
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
	u, err := user.Current()
	if err == nil {
		return u.HomeDir
	}

	// alternate methods
	homeDir := os.Getenv("HOME") // *nix
	return homeDir
}
