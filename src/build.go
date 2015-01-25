package src

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
)

type Build struct {
	GetBuildResponse
}

func (b *Build) projectDir() string {
	return fmt.Sprintf("project-%d", b.ProjectId)
}

func (b *Build) writeCloneCmd(w *bufio.Writer, builds_dir string) {
	w.WriteString(fmt.Sprintf("mkdir -p %s && ", builds_dir))
	w.WriteString(fmt.Sprintf("cd %s && ", builds_dir))
	w.WriteString(fmt.Sprintf("rm -rf %s && ", b.projectDir()))
	w.WriteString(fmt.Sprintf("git clone %s %s && ", b.RepoURL, b.projectDir()))
	w.WriteString(fmt.Sprintf("cd %s\n", b.projectDir()))
}

func (b *Build) writeFetchCmd(w *bufio.Writer, builds_dir string) {
	w.WriteString(fmt.Sprintf("if [[ -d %s/%s/.git ]]; then\n", builds_dir, b.projectDir()))
	w.WriteString(fmt.Sprintf("cd %s/%s && ", builds_dir, b.projectDir()))
	w.WriteString(fmt.Sprintf("git clean -fdx && "))
	w.WriteString(fmt.Sprintf("git reset --hard && "))
	w.WriteString(fmt.Sprintf("git remote set-url origin %s &&", b.RepoURL))
	w.WriteString(fmt.Sprintf("git fetch origin\n"))
	w.WriteString(fmt.Sprintf("else\n"))
	b.writeCloneCmd(w, builds_dir)
	w.WriteString(fmt.Sprintf("fi\n"))
}

func (b *Build) writeCheckoutCmd(w *bufio.Writer, builds_dir string) {
	w.WriteString(fmt.Sprintf("git checkout %s && ", b.RefName))
	w.WriteString(fmt.Sprintf("git reset --hard %s\n", b.Sha))
}

func (b *Build) Generate(builds_dir string) *string {
	file, err := ioutil.TempFile("", "build_script")
	if err != nil {
		return nil
	}

	os.Chmod(file.Name(), os.ModePerm&0700)

	w := bufio.NewWriter(file)
	defer w.Flush()

	w.WriteString("#!/usr/bin/env bash\n")
	w.WriteString("\n")
	w.WriteString("echo Using $(hostname)\n")
	w.WriteString("\n")
	w.WriteString("trap 'kill -s INT 0' EXIT\n")
	w.WriteString("set -evo pipefail\n")
	w.WriteString("\n")

	if b.AllowGitFetch {
		b.writeFetchCmd(w, builds_dir)
	} else {
		b.writeCloneCmd(w, builds_dir)
	}

	b.writeCheckoutCmd(w, builds_dir)
	w.WriteString("\n")

	w.WriteString(b.Commands)

	name := file.Name()
	return &name
}
