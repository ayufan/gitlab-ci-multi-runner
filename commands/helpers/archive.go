package commands_helpers

import (
	"bufio"
	"bytes"
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ArchiveCommand struct {
	Paths     []string `long:"path" description:"Add paths to archive"`
	Untracked bool     `long:"untracked" description:"Add git untracked files"`
	Output    string   `long:"output" description:"The filepath to output file"`
	Silent    bool     `long:"silent" description:"Suppress archiving ouput"`
	List      bool     `long:"list" description:"List files to archive"`

	wd    string
	files map[string]os.FileInfo
}

func (c *ArchiveCommand) isChanged(modTime time.Time) bool {
	for _, info := range c.files {
		if modTime.Before(info.ModTime()) {
			return true
		}
	}
	return false
}

func (c *ArchiveCommand) sortedFiles() []string {
	files := make([]string, len(c.files))

	i := 0
	for file := range c.files {
		files[i] = file
		i++
	}

	sort.Strings(files)
	return files
}

func (c *ArchiveCommand) add(path string, info os.FileInfo) (err error) {
	if info == nil {
		info, err = os.Lstat(path)
	}
	if err == nil {
		c.files[path] = info
	} else if os.IsNotExist(err) {
		logrus.Warningln("File", path, "doesn't exist")
		err = nil
	}
	return
}

func (c *ArchiveCommand) process(match string) error {
	absolute, err := filepath.Abs(match)
	if err != nil {
		return err
	}

	relative, err := filepath.Rel(c.wd, absolute)
	if err != nil {
		return err
	}

	// store relative path if points to current working directory
	if strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return c.add(absolute, nil)
	} else {
		return c.add(relative, nil)
	}
}

func (c *ArchiveCommand) processPaths() {
	for _, path := range c.Paths {
		matches, err := filepath.Glob(path)
		if err != nil {
			logrus.Warningln(err)
			continue
		}

		for _, match := range matches {
			err := filepath.Walk(match, func(path string, info os.FileInfo, err error) error {
				return c.process(path)
			})
			if err != nil {
				logrus.Warningln("Walking", match, err)
			}
		}
	}
}

func (c *ArchiveCommand) processUntracked() {
	if !c.Untracked {
		return
	}

	var output bytes.Buffer
	cmd := exec.Command("git", "ls-files", "-o")
	cmd.Env = os.Environ()
	cmd.Stdout = &output
	cmd.Stderr = os.Stderr
	logrus.Debugln("Executing command:", strings.Join(cmd.Args, " "))
	err := cmd.Run()
	if err == nil {
		reader := bufio.NewReader(&output)
		for {
			line, _, err := reader.ReadLine()
			if err == io.EOF {
				break
			} else if err != nil {
				logrus.Warningln(err)
				break
			}
			c.process(string(line))
		}
	} else {
		logrus.Warningln(err)
	}
}

func (c *ArchiveCommand) listFiles() {
	if len(c.files) == 0 {
		logrus.Infoln("No files to archive.")
		return
	}

	for _, file := range c.sortedFiles() {
		println(string(file))
	}
}

func (c *ArchiveCommand) archive() {
	if len(c.files) == 0 {
		logrus.Infoln("No files to archive.")
		return
	}

	logrus.Infoln("Creating archive", filepath.Base(c.Output), "...")
	var files bytes.Buffer
	for _, file := range c.sortedFiles() {
		files.WriteString(string(file) + "\n")
	}

	// create directories to store archive
	os.MkdirAll(filepath.Dir(c.Output), 0700)

	tempFile, err := ioutil.TempFile(filepath.Dir(c.Output), "archive_")
	if err != nil {
		logrus.Fatalln("Failed to create temporary archive", err)
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	logrus.Debugln("Temporary file:", tempFile.Name())

	flags := "-zcPv"
	if c.Silent {
		flags = "-zcP"
	}

	cmd := exec.Command("tar", flags, "-T", "-", "--no-recursion", "-f", tempFile.Name())
	cmd.Env = os.Environ()
	cmd.Stdin = &files
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logrus.Debugln("Executing command:", strings.Join(cmd.Args, " "))
	err = cmd.Run()
	if err != nil {
		logrus.Fatalln("Failed to create archive:", err)
	}

	err = os.Rename(tempFile.Name(), c.Output)
	if err != nil {
		logrus.Warningln("Failed to rename archive:", err)
	}

	logrus.Infoln("Done!")
}

func (c *ArchiveCommand) Execute(context *cli.Context) {
	logrus.SetFormatter(
		&logrus.TextFormatter{
			ForceColors:      true,
			DisableTimestamp: false,
		},
	)

	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatalln("Failed to get current working directory:", err)
	}
	if c.Output == "" && !c.List {
		logrus.Fatalln("Missing archive file name!")
	}

	c.wd = wd
	c.files = make(map[string]os.FileInfo)

	c.processPaths()
	c.processUntracked()

	ai, err := os.Stat(c.Output)
	if err != nil && !os.IsNotExist(err) {
		logrus.Fatalln("Failed to verify archive:", c.Output, err)
	}
	if ai != nil {
		if !c.isChanged(ai.ModTime()) {
			logrus.Infoln("Archive is up to date!")
			return
		}
	}

	if c.List {
		c.listFiles()
	} else {
		c.archive()
	}
}

func init() {
	common.RegisterCommand2("archive", "find and archive files (internal)", &ArchiveCommand{})
}
