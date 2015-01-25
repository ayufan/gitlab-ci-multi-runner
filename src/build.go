package src

import (
	"bufio"
	"os"
	"io/ioutil"
)

type Build struct {
	GetBuildResponse
}

func (b *Build) Generate() *string {
	file, err := ioutil.TempFile("", "build_script")
	if err != nil {
		return nil
	}

	os.Chmod(file.Name(), os.ModePerm & 0700)

	w := bufio.NewWriter(file)
	defer w.Flush()

	w.WriteString("#!/usr/bin/env bash\n")
	w.WriteString("\n")
	w.WriteString("echo Using $(hostname)\n")
	w.WriteString("\n")
	w.WriteString("set -e\n")
	w.WriteString("trap 'kill -s INT 0' EXIT\n")
	w.WriteString("\n")
	w.WriteString(b.Commands)

	name := file.Name()
	return &name
}
