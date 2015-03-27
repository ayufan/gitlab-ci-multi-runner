package commands

import (
	"github.com/codegangsta/cli"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/ayufan/gitlab-ci-multi-runner/common"
	"net/http"
)

func serverHelloWorld(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func runServer(addr string) error {
	if len(addr) == 0 {
		return nil
	}

	http.HandleFunc("/", serverHelloWorld)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func runHerokuURL(addr string) error {
	if len(addr) == 0 {
		return nil
	}

	for {
		resp, err := http.Get(addr)
		if err == nil {
			log.Infoln("HEROKU_URL acked!")
			defer resp.Body.Close()
		} else {
			log.Infoln("HEROKU_URL error: ", err)
		}
		time.Sleep(5 * time.Minute)
	}
}

func runSingle(c *cli.Context) {
	runner := common.RunnerConfig{
		URL:       c.String("url"),
		Token:     c.String("token"),
		Executor:  c.String("executor"),
		BuildsDir: c.String("builds-dir"),
		Shell:     c.String("shell"),
	}

	if len(runner.URL) == 0 {
		log.Fatalln("Missing URL")
	}
	if len(runner.Token) == 0 {
		log.Fatalln("Missing Token")
	}
	if len(runner.Executor) == 0 {
		log.Fatalln("Missing Executor")
	}

	go runServer(c.String("addr"))
	go runHerokuURL(c.String("heroku-url"))

	log.Println("Starting runner for", runner.URL, "with token", runner.ShortDescription(), "...")

	for {
		buildData, healthy := common.GetBuild(runner)
		if !healthy {
			log.Println("Runner died, beacuse it's not healthy!")
			os.Exit(1)
		}
		if buildData == nil {
			time.Sleep(common.CheckInterval * time.Second)
			continue
		}

		newBuild := common.Build{
			GetBuildResponse: *buildData,
			Runner:           &runner,
		}
		newBuild.Prepare([]*common.Build{})
		newBuild.Run()
	}
}

func init() {
	common.RegisterCommand(cli.Command{
		Name:   "run-single",
		Usage:  "start single runner",
		Action: runSingle,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "token",
				Value:  "",
				Usage:  "Runner token",
				EnvVar: "RUNNER_TOKEN",
			},
			cli.StringFlag{
				Name:   "url",
				Value:  "",
				Usage:  "Runner URL",
				EnvVar: "CI_SERVER_URL",
			},
			cli.StringFlag{
				Name:   "executor",
				Value:  "shell",
				Usage:  "Executor",
				EnvVar: "RUNNER_EXECUTOR",
			},
			cli.StringFlag{
				Name:   "shell",
				Value:  common.GetDefaultShell(),
				Usage:  "Shell to use for run the script",
				EnvVar: "RUNNER_SHELL",
			},
			cli.StringFlag{
				Name:   "addr",
				Value:  "",
				Usage:  "Hello World Server",
				EnvVar: "",
			},
			cli.StringFlag{
				Name:   "heroku-url",
				Value:  "",
				Usage:  "Current application address",
				EnvVar: "HEROKU_URL",
			},
			cli.StringFlag{
				Name:   "builds-dir",
				Value:  "",
				Usage:  "Custom builds directory",
				EnvVar: "RUNNER_BUILDS_DIR",
			},
		},
	})
}
