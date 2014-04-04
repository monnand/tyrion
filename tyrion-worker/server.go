package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type TaskServer struct {
	rr ResponseReader
}

func NewTaskServer(rr ResponseReader) *TaskServer {
	ret := new(TaskServer)
	ret.rr = rr
	return ret
}

func (self *TaskServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	self.ServeJson(w, r.Body)
}

type taskResult struct {
	Errors []string `json:"errors,omitempty"`
	Envs   []*Env   `json:"envs"`
}

func (self *TaskServer) ServeJson(w io.Writer, r io.Reader) {
	decoder := json.NewDecoder(r)
	var taskSpec TaskSpec
	err := decoder.Decode(&taskSpec)

	if err != nil {
		fmt.Fprintf(w, `{"errors": "%v"}`, err)
		return
	}
	errChan := make(chan error)
	var tr taskResult
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range errChan {
			if err != nil {
				es := fmt.Sprintf("Error: %v\n", err)
				tr.Errors = append(tr.Errors, es)
			}
		}
	}()
	task := taskSpec.GetWorker(self.rr)
	envs := task.Execute(errChan)
	close(errChan)
	wg.Wait()

	tr.Envs = envs
	encoder := json.NewEncoder(w)
	encoder.Encode(&tr)
}
