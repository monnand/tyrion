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

func (self *TaskServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	self.ServeJson(w, r.Body)
}

type taskResult struct {
	Errors []string `json:"errors"`
	Envs   []*Env   `json:"envs"`
}

func (self *TaskServer) ServeJson(w io.Writer, r io.Reader) {
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
	decoder := json.NewDecoder(r)
	var taskSpec TaskSpec
	decoder.Decode(&taskSpec)
	task := taskSpec.GetWorker(self.rr)
	envs := task.Execute(errChan)
	close(errChan)
	wg.Wait()

	tr.Envs = envs
	encoder := json.NewEncoder(w)
	encoder.Encode(&tr)
}
