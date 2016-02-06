package helpers

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/archives"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/formatter"
)

type CacheExtractorCommand struct {
	File string `long:"file" description:"The file containing your cache artifacts"`
	URL  string `long:"url" description:"Download artifacts instead of uploading them"`
}

func (c *CacheExtractorCommand) download() error {
	os.MkdirAll(filepath.Dir(c.File), 0600)

	file, err := ioutil.TempFile(filepath.Dir(c.File), "cache")
	if err != nil {
		return err
	}
	defer file.Close()
	defer os.Remove(file.Name())

	resp, err := http.Get(c.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return os.ErrNotExist
	} else if resp.StatusCode != 200 {
		return fmt.Errorf("Received: %s", resp.Status)
	}

	fi, _ := os.Lstat(c.File)
	date, _ := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
	if fi != nil && !date.After(fi.ModTime()) {
		logrus.Infoln(filepath.Base(c.File), "is up to date")
		return nil
	}

	logrus.Infoln("Downloading", filepath.Base(c.File))
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	os.Chtimes(file.Name(), time.Now(), date)

	err = os.Rename(file.Name(), c.File)
	if err != nil {
		return err
	}
	return nil
}

func (c *CacheExtractorCommand) Execute(context *cli.Context) {
	formatter.SetRunnerFormatter()

	if len(c.File) == 0 {
		logrus.Fatalln("Missing cache file")
	}

	if c.URL != "" {
		err := c.download()
		if err != nil && !os.IsNotExist(err) {
			logrus.Warningln(err)
		}
	}

	err := archives.ExtractZipFile(c.File)
	if err != nil && !os.IsNotExist(err) {
		logrus.Fatalln(err)
	}
}

func init() {
	common.RegisterCommand2("cache-extractor", "download and extract cache artifacts (internal)", &CacheExtractorCommand{})
}
