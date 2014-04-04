package main

import (
	"fmt"
	"sync"
)

type PluginSpec struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"parameters,omitempty"`
}

type PluginFactory interface {
	String() string
	NewPlugin(params map[string]string, rest ResponseReader) (rr ResponseReader, err error)
}

func (self *PluginSpec) GetPlugin(factory PluginFactory, rest ResponseReader) (rr ResponseReader, err error) {
	if self.Name != factory.String() {
		err = fmt.Errorf("Unmatched factory: %v is not %v.", factory.String(), self.Name)
		return
	}
	rr, err = factory.NewPlugin(self.Params, rest)
	return
}

type PluginManager struct {
	nameMap map[string]PluginFactory
	lock    sync.RWMutex
}

var globalPm PluginManager

func RegisterPlugin(f PluginFactory) {
	globalPm.Register(f)
}
func NewPluginChain(specs []*PluginSpec) (rr ResponseReader, err error) {
	return globalPm.NewPluginChain(specs)
}

func (self *PluginManager) Register(f PluginFactory) {
	if f == nil {
		return
	}
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.nameMap == nil {
		self.nameMap = make(map[string]PluginFactory, 10)
	}
	self.nameMap[f.String()] = f
}

func (self *PluginManager) NewPluginChain(specs []*PluginSpec) (rr ResponseReader, err error) {
	var ret ResponseReader
	self.lock.RLock()
	defer self.lock.RUnlock()
	for n := len(specs) - 1; n >= 0; n-- {
		spec := specs[n]
		if f, ok := self.nameMap[spec.Name]; ok {
			ret, err = spec.GetPlugin(f, ret)
		} else {
			fmt.Errorf("Unknown plugin: %v", spec.Name)
		}
	}
	rr = ret
	return
}
