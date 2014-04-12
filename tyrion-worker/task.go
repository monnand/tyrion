package main

import (
	"fmt"
	"io"
	"sync"
)

var subTaskChan chan *subTask

func init() {
	subTaskChan = make(chan *subTask)
}

type TaskExecutor interface {
	Execute(errChan chan<- error) []*Env
}

type ConcurrentActions struct {
	Actions             []*ActionSpec `json:"concurrent-actions"`
	ProceedWhenNoUpdate bool          `json:"proceed-when-no-update,omitempty"`
}

type TaskSpec struct {
	ConcurrentActions []*ConcurrentActions `json:"action-seq"`
	InitEnv           *Env                 `json:"env,omitempty"`
	Plugins           []*PluginSpec        `json:"plugins,omitempty"`
	Finalizers        []*TaskFinalizerSpec `json:"finally,omitempty"`
}

func (self *TaskSpec) GetWorker(rr ResponseReader) (exec TaskExecutor, err error) {
	ret := new(worker)

	if rr == nil {
		plugins := self.Plugins
		if len(plugins) == 0 {
			plugins = []*PluginSpec{
				&PluginSpec{
					Name:   "http",
					Params: nil,
				},
			}
		}
		rr, err = NewPluginChain(plugins)
		if err != nil {
			return
		}
		ret.closer = rr
	}

	ret.rr = rr
	ret.spec = self
	ret.subTaskChan = subTaskChan
	exec = ret
	return
}

func StartWorkers(n int) {
	if n <= 0 {
		n = 2
	}
	for i := 0; i < n; i++ {
		go subTaskExecutor(subTaskChan)
	}
}

func StopAllWorkers() {
	close(subTaskChan)
	subTaskChan = make(chan *subTask)
}

type worker struct {
	subTaskChan chan<- *subTask
	spec        *TaskSpec
	rr          ResponseReader
	closer      io.Closer
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

func (self *worker) Execute(errChan chan<- error) []*Env {
	if self.closer != nil {
		defer self.closer.Close()
	}
	envs := make([]*Env, 1)
	envs[0] = self.spec.InitEnv
	if envs[0].IsEmpty() {
		envs[0] = EmptyEnv()
	}
	var nilEnvs [1]*Env
	nilEnvs[0] = EmptyEnv()

	for _, concurrentActions := range self.spec.ConcurrentActions {
		nrActions := len(concurrentActions.Actions)
		if nrActions == 0 {
			continue
		}
		resChan := make(chan *subTaskResult)
		updates := make([]*Env, 0, nrActions*2)
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
		}(nrActions * len(envs))

		for _, env := range envs {
			for _, spec := range concurrentActions.Actions {
				action, err := spec.GetAction(self.rr)
				if err != nil {
					res := new(subTaskResult)
					res.err = fmt.Errorf("Action %v is invalid: %v", spec.Tag, err)
					resChan <- res
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
		if len(updates) == 0 && !concurrentActions.ProceedWhenNoUpdate {
			break
		}
		forks := make([]*Env, 0, len(envs)*len(updates))
		for _, env := range envs {
			f := env.Fork(updates...)
			forks = append(forks, f...)
		}
		envs = uniqEnvs(forks...)
		if len(envs) == 0 {
			envs = nilEnvs[:]
		}
	}
	return envs
}
