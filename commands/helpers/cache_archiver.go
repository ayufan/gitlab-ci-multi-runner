package helpers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/archives"
)

type CacheArchiverCommand struct {
	fileArchiver
	File string `long:"file" description:"The path to file"`
	URL  string `long:"url" description:"Download artifacts instead of uploading them"`
}

func (c *CacheArchiverCommand) upload() error {
	logrus.Infoln("Uploading", filepath.Base(c.File))

	file, err := os.Open(c.File)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", c.URL, file)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Last-Modified", fi.ModTime().Format(http.TimeFormat))
	req.ContentLength = fi.Size()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Received: %s", resp.Status)
	}

	return nil
}

func (c *CacheArchiverCommand) Execute(*cli.Context) {
	if c.File == "" {
		logrus.Fatalln("Missing --file")
	}

	// Enumerate files
	err := c.enumerate()
	if err != nil {
		logrus.Fatalln(err)
	}

	// Check if list of files changed
	if !c.isFileChanged(c.File) {
		logrus.Infoln("Archive is up to date!")
		return
	}

	// Create archive
	err = archives.CreateZipFile(c.File, c.sortedFiles())
	if err != nil {
		logrus.Fatalln(err)
	}

	// Upload archive if needed
	if c.URL != "" {
		for i := 0; i < 3; i++ {
			err := c.upload()
			if err == nil {
				break
			}
			logrus.Warningln(err)
			time.Sleep(time.Second)
		}
	}
}

func init() {
	common.RegisterCommand2("cache-archiver", "create and upload cache artifacts (internal)", &CacheArchiverCommand{})
}
