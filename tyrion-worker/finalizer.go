package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

func init() {
	RegisterTaskFinalizer(&mergeFinalizerFactory{})
	RegisterTaskFinalizer(&initEnvReplacerFactory{})
	RegisterTaskFinalizer(&taskSpecWriterFactory{})
}

type TaskFinalizerSpec struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"parameters,omitempty"`
}

func (self *TaskFinalizerSpec) GetTaskFinalizer(factory TaskFinalizerFactory, rest TaskFinalizer) (tf TaskFinalizer, err error) {
	if self.Name != factory.String() {
		err = fmt.Errorf("Unmached factory: %v is not %v", factory.String(), self.Name)
		return
	}
	tf, err = factory.NewFinalizer(self.Params, rest)
	return
}

type TaskFinalizer interface {
	FinalizeTask(spec *TaskSpec, envs []*Env) error
}

type TaskFinalizerFactory interface {
	String() string
	NewFinalizer(params map[string]string, rest TaskFinalizer) (tf TaskFinalizer, err error)
}

type TaskFinalizerManager struct {
	nameMap map[string]TaskFinalizerFactory
	lock    sync.RWMutex
}

func (self *TaskFinalizerManager) Register(f TaskFinalizerFactory) {
	if f == nil {
		return
	}
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.nameMap == nil {
		self.nameMap = make(map[string]TaskFinalizerFactory, 10)
	}
	self.nameMap[f.String()] = f
}

func (self *TaskFinalizerManager) NewTaskFinalizerChain(specs []*TaskFinalizerSpec) (tf TaskFinalizer, err error) {
	var ret TaskFinalizer
	self.lock.RLock()
	defer self.lock.RLock()

	for n := len(specs) - 1; n >= 0; n-- {
		spec := specs[n]
		if f, ok := self.nameMap[spec.Name]; ok {
			ret, err = spec.GetTaskFinalizer(f, ret)
		} else {
			err = fmt.Errorf("Unknown finalizer: %v", spec.Name)
			return
		}
	}
	tf = ret
	return
}

var globalTfm TaskFinalizerManager

func RegisterTaskFinalizer(f TaskFinalizerFactory) {
	globalTfm.Register(f)
}

func NewTaskFinalizerChain(specs []*TaskFinalizerSpec) (tf TaskFinalizer, err error) {
	return globalTfm.NewTaskFinalizerChain(specs)
}

type mergeFinalizerFactory struct {
}

func (self *mergeFinalizerFactory) String() string {
	return "merge"
}

func (self *mergeFinalizerFactory) NewFinalizer(params map[string]string, rest TaskFinalizer) (tf TaskFinalizer, err error) {
	var keys []string
	if ks, ok := params["keys"]; ok {
		keys = strings.Split(ks, ",")
	}
	if len(keys) == 0 {
		err = fmt.Errorf("has to specify at least one merge key")
		return
	}
	ret := new(mergeFinalizer)
	ret.mergeKeys = keys
	ret.rest = rest
	tf = ret
	return
}

type mergeFinalizer struct {
	mergeKeys []string
	rest      TaskFinalizer
}

func (self *mergeFinalizer) FinalizeTask(spec *TaskSpec, envs []*Env) error {
	env := EmptyEnv()
	for _, key := range self.mergeKeys {
		value := ""
		for _, e := range envs {
			fmt.Printf("merging env: %+v\n", e)
			if v, ok := e.NameValuePairs[key]; ok {
				if len(value) > 0 {
					if v != value {
						err := fmt.Errorf("cannot merge key %v, which has two values: %v and %v",
							key, v, value)
						return err
					}
				} else {
					value = v
				}
			}
		}
		env.NameValuePairs[key] = value
	}
	fmt.Printf("merged env: %+v\n", env)
	if self.rest != nil {
		return self.rest.FinalizeTask(spec, []*Env{env})
	}
	return nil
}

type initEnvReplacer struct {
	rest TaskFinalizer
}

func (self *initEnvReplacer) FinalizeTask(spec *TaskSpec, envs []*Env) error {
	if len(envs) != 1 {
		return fmt.Errorf("replace-init-env: %v environments, not 1", len(envs))
	}

	if self.rest != nil {
		spec.InitEnv = envs[0]
		return self.rest.FinalizeTask(spec, nil)
	}
	return nil
}

type initEnvReplacerFactory struct {
}

func (self *initEnvReplacerFactory) String() string {
	return "replace-init-env"
}

func (self *initEnvReplacerFactory) NewFinalizer(params map[string]string, rest TaskFinalizer) (tf TaskFinalizer, err error) {
	ret := new(initEnvReplacer)
	ret.rest = rest
	tf = ret
	return
}

type taskSpecWriter struct {
	w         io.WriteCloser
	notPretty bool
	rest      TaskFinalizer
}

func (self *taskSpecWriter) FinalizeTask(spec *TaskSpec, envs []*Env) error {
	defer self.w.Close()
	var buf []byte
	var err error
	if self.notPretty {
		buf, err = json.Marshal(spec)
	} else {
		buf, err = json.MarshalIndent(spec, "", "    ")
	}
	if err != nil {
		return fmt.Errorf("unable to marshal the task spec: %v", err)
	}
	_, err = self.w.Write(buf)
	if err != nil {
		return fmt.Errorf("unable to write the task spec: %v", err)
	}
	if self.rest != nil {
		return self.rest.FinalizeTask(spec, envs)
	}
	return nil
}

type taskSpecWriterFactory struct {
}

func (self *taskSpecWriterFactory) String() string {
	return "write-spec"
}

func (self *taskSpecWriterFactory) NewFinalizer(params map[string]string, rest TaskFinalizer) (tf TaskFinalizer, err error) {
	ret := new(taskSpecWriter)
	ret.rest = rest
	if filename, ok := params["file"]; ok {
		ret.w, err = os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return
		}
	}
	if ret.w == nil {
		err = fmt.Errorf("write-spec: cannot find output")
		return
	}
	tf = ret
	return
}
