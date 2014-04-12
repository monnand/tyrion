package main

import (
	"fmt"
	"regexp"
	"sync"
)

type PluginSpec struct {
	Name        string            `json:"name"`
	TagPatterns []string          `json:"tags,omitempty"`
	Params      map[string]string `json:"parameters,omitempty"`
}

type PluginFactory interface {
	String() string
	NewPlugin(params map[string]string, rest ResponseReader) (rr ResponseReader, err error)
}

type pluginTagFilter struct {
	tags   []*regexp.Regexp
	plugin ResponseReader
	rest   ResponseReader
}

func (self *pluginTagFilter) ReadResponse(req *Request, env *Env) (resp *Response, updates *Env, err error) {
	if len(self.tags) == 0 {
		return self.plugin.ReadResponse(req, env)
	}
	matched := false
	for _, t := range self.tags {
		m := t.FindString(req.Tag)
		if len(m) > 0 {
			matched = true
			break
		}
	}
	if matched {
		return self.plugin.ReadResponse(req, env)
	}
	return self.rest.ReadResponse(req, env)
}

func (self *pluginTagFilter) Close() error {
	return self.plugin.Close()
}

func (self *PluginSpec) GetPlugin(factory PluginFactory, rest ResponseReader) (rr ResponseReader, err error) {
	if self.Name != factory.String() {
		err = fmt.Errorf("Unmatched factory: %v is not %v.", factory.String(), self.Name)
		return
	}
	tagfilter := new(pluginTagFilter)

	for _, t := range self.TagPatterns {
		var p *regexp.Regexp
		p, err = regexp.Compile(t)
		if err != nil {
			err = fmt.Errorf("tag %v is not a regular expression: %v", t, err)
			return
		}
		tagfilter.tags = append(tagfilter.tags, p)
	}
	plugin, err := factory.NewPlugin(self.Params, rest)
	if err != nil {
		return
	}
	if plugin == nil {
		err = fmt.Errorf("%v returned a nil plugin", factory.String())
		return
	}
	tagfilter.plugin = plugin
	tagfilter.rest = rest
	rr = tagfilter
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
			err = fmt.Errorf("Unknown plugin: %v", spec.Name)
			return
		}
	}
	rr = ret
	return
}
