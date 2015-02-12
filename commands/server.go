package commands

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"sync"
)

type MultiRunnerServer struct {
	*MultiRunner
	listenAddresses []string
}

func (mr *MultiRunnerServer) getConfig(w http.ResponseWriter, req *http.Request) {
	config := mr.config
	if config == nil {
		http.Error(w, "Missing config", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (mr *MultiRunnerServer) getAllBuilds(w http.ResponseWriter, req *http.Request) {
	data, err := json.Marshal(mr.allBuilds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (mr *MultiRunnerServer) getCurrentBuilds(w http.ResponseWriter, req *http.Request) {
	data, err := json.Marshal(mr.builds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (mr *MultiRunnerServer) Run() error {
	http.HandleFunc("/api/v1/config.json", mr.getConfig)
	http.HandleFunc("/api/v1/builds.json", mr.getAllBuilds)
	http.HandleFunc("/api/v1/current.json", mr.getCurrentBuilds)

	var returnError error
	wg := sync.WaitGroup{}

	for _, listenAddr := range mr.listenAddresses {
		log.Infoln("Starting API server on", listenAddr, "...")

		wg.Add(1)
		go func() {
			err := http.ListenAndServe(listenAddr, nil)
			if err != nil {
				log.Fatal(err)
				returnError = err
			}
			wg.Done()
		}()
	}

	wg.Wait()

	return returnError
}
