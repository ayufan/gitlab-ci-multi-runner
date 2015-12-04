package commands

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/EMSSConsulting/Thargo"
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type ArchiveCommand struct {
	Paths     []string `long:"path" description:"Glob based filters used to include files"`
	Untracked bool     `long:"untracked" description:"Add git untracked files"`
	Output    string   `long:"output" description:"The filepath to output file"`
	Silent    bool     `long:"silent" description:"Suppress archiving ouput"`
	List      bool     `long:"list" description:"List files to archive"`

	files map[string]bool
	wd    string
}

func (c *ArchiveCommand) archive() {
	logrus.Infoln("Creating archive", filepath.Base(c.Output), "...")

	// create directories to store archive
	os.MkdirAll(filepath.Dir(c.Output), 0700)

	tempFile, err := ioutil.TempFile(filepath.Dir(c.Output), "archive_")
	if err != nil {
		logrus.Fatalln("Failed to create temporary archive", err)
	}
	tempFile.Close()

	defer os.Remove(tempFile.Name())

	archive, err := thargo.NewArchiveFile(tempFile.Name(), nil)
	if err != nil {
		logrus.Fatalln("Failed to open archive for writing: ", err)
	}

	logrus.Debugln("Temporary file:", tempFile.Name())

	filter := thargo.Filter{
		Entry: func(entry thargo.Entry) bool {
			header, err := entry.Header()
			if err != nil {
				return false
			}

			// Fix up the header name if the path is outside of the cwd
			if filepath.HasPrefix(header.Name, "../") {
				absPath, err := filepath.Abs(header.Name)
				if err != nil {
					return false
				}

				header.Name = absPath
			}

			// Don't include duplicate files
			if _, exists := c.files[header.Name]; exists {
				return false
			}

			if c.List {
				println(header.Name)
			}

			c.files[header.Name] = true
			return true
		},
		Entries: func(entries []thargo.Entry) bool {
			ai, err := os.Stat(c.Output)
			if err != nil && !os.IsNotExist(err) {
				logrus.Fatalln("Failed to verify archive:", c.Output, err)
			}

			if ai == nil {
				return true
			}

			for _, entry := range entries {
				header, err := entry.Header()
				if err != nil {
					return false
				}

				if header.ModTime.After(ai.ModTime()) {
					return true
				}
			}

			return false
		},
	}

	targets := []thargo.Target{}

	for _, filter := range c.Paths {
		if !c.Silent {
			logrus.Infof("Adding '%s' to archive", filter)
		}

		targets = append(targets, &thargo.FileSystemTarget{
			Path:    c.wd,
			Pattern: filter,
		})
	}

	if c.Untracked {
		targets = append(targets, &GitUntrackedFilesTarget{
			Path: c.wd,
		})
	}

	added, err := archive.AddFiltered(targets, filter)
	if err != nil {
		logrus.Fatalln("Failed to add entries to archive: ", err)
	}

	if err := archive.Close(); err != nil {
		logrus.Warningln("Failed to close temp archive: ", err)
	}

	if added == 0 {
		logrus.Infoln("Archive up to date!")
	} else {
		err = os.Rename(tempFile.Name(), c.Output)
		if err != nil {
			logrus.Warningln("Failed to rename archive:", err)
		}
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
	c.files = make(map[string]bool)

	if len(c.Paths) == 0 && !c.Untracked {
		logrus.Infoln("No inclusion filters specified.")
		return
	}

	c.archive()

	if !c.Silent {
		for file := range c.files {
			logrus.Println(" - ", file)
		}
	}
}

func init() {
	common.RegisterCommand2("archive", "find and archive files (internal)", &ArchiveCommand{})
}

// GitUntrackedFilesTarget is a Thargo target which provides a list of untracked
// git files for inclusion in an archive.
type GitUntrackedFilesTarget struct {
	Path string
}

func (t *GitUntrackedFilesTarget) Entries() ([]thargo.Entry, error) {
	entries := []thargo.Entry{}

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

			entry, err := t.processFile(string(line))
			if err != nil {
				logrus.Warningln("Failed to include untracked file", line, err)
			} else if entry == nil {
				continue
			}

			entries = append(entries, entry)
		}
	} else {
		logrus.Warningln(err)
	}

	return entries, nil
}

func (t *GitUntrackedFilesTarget) processFile(path string) (thargo.Entry, error) {
	f, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// Skip directories
	if f.IsDir() {
		return nil, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	return &thargo.FileEntry{Name: path, Path: absPath, Info: f}, nil
}
