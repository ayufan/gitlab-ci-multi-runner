package src

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
)

type Build struct {
	GetBuildResponse
	Name string
}

func (b *Build) isNameUnique(other_builds []*Build) bool {
	for _, other_build := range other_builds {
		if b.Name == other_build.Name {
			return false
		}
	}
	return true
}

func (b *Build) GenerateUniqueName(prefix string, other_builds []*Build) {
	for i := 0; ; i++ {
		b.Name = fmt.Sprintf("%s-project-%d-%d", prefix, b.ProjectId, i)
		if b.isNameUnique(other_builds) {
			return
		}
	}
}

func (b *Build) ProjectUniqueName() string {
	if len(b.Name) == 0 {
		return fmt.Sprintf("project-%d", b.ProjectId)
	} else {
		return b.Name
	}
}

func (b *Build) ProjectDir() string {
	return b.ProjectUniqueName()
}

func (b *Build) writeCloneCmd(w *bufio.Writer, builds_dir string) {
	w.WriteString(fmt.Sprintf("mkdir -p %s && ", builds_dir))
	w.WriteString(fmt.Sprintf("cd %s && ", builds_dir))
	w.WriteString(fmt.Sprintf("rm -rf %s && ", b.ProjectDir()))
	w.WriteString(fmt.Sprintf("git clone %s %s && ", b.RepoURL, b.ProjectDir()))
	w.WriteString(fmt.Sprintf("cd %s\n", b.ProjectDir()))
}

func (b *Build) writeFetchCmd(w *bufio.Writer, builds_dir string) {
	w.WriteString(fmt.Sprintf("if [[ -d %s/%s/.git ]]; then\n", builds_dir, b.ProjectDir()))
	w.WriteString(fmt.Sprintf("cd %s/%s && ", builds_dir, b.ProjectDir()))
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
