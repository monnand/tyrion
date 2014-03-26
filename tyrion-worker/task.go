package main

import (
	"fmt"
	"sync"
)

type TaskSpec struct {
	Actions [][]*ActionSpec `json:"actions"`
	InitEnv *Env            `json:"env"`
}

type Task struct {
	subTaskChan chan<- *subTask
	spec        *TaskSpec
	rr          ResponseReader
}

type subTaskResult struct {
	err     error
	updates []*Env
}

type subTask struct {
	action  *Action
	env     *Env
	resChan chan<- *subTaskResult
}

func subTaskExecutor(taskChan <-chan *subTask) {
	for st := range taskChan {
		updates, err := st.action.Perform(st.env)
		res := new(subTaskResult)
		res.updates = updates
		res.err = err
		st.resChan <- res
	}
}

func (self *Task) Execute(errChan chan<- error) {
	envs := make([]*Env, 1)
	envs[0] = self.spec.InitEnv

	for _, concurrentActions := range self.spec.Actions {
		nrActions := len(concurrentActions)
		if nrActions == 0 {
			continue
		}
		updates := make([]*Env, 0, nrActions*2)
		resChan := make(chan *subTaskResult)
		var wg sync.WaitGroup
		wg.Add(1)
		// reaper function
		go func(n int) {
			defer wg.Done()
			for i := 0; i < n; i++ {
				res := <-resChan
				if res.err != nil {
					errChan <- res.err
					continue
				}
				updates = append(updates, res.updates...)
			}
		}(nrActions)
		for _, env := range envs {
			for _, spec := range concurrentActions {
				action, err := spec.GetAction(self.rr)
				if err != nil {
					errChan <- fmt.Errorf("Action %v is invalid: %v", action.Tag, err)
					continue
				}
				st := new(subTask)
				st.action = action
				st.env = env
				st.resChan = resChan
				self.subTaskChan <- st
			}
		}
		wg.Wait()
		forks := make([]*Env, 0, len(envs)*len(updates))
		for _, env := range envs {
			f := env.Fork(updates...)
			forks = append(forks, f...)
		}
		envs = forks
	}
}
