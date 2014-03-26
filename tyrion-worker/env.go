package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type Env struct {
	NameValuePairs map[string]string `json:"vars"`
}

func (self *Env) Fork(deltas ...*Env) []*Env {
	merged := mergeSameEnvs(deltas...)
	if len(merged) == 0 {
		return nil
	}
	ret := make([]*Env, len(merged))
	for i, e := range merged {
		n := self.Clone()
		n.Update(e)
		ret[i] = n
	}
	return ret
}

func (self *Env) Clone() *Env {
	if self == nil {
		return nil
	}
	if len(self.NameValuePairs) == 0 {
		return nil
	}
	ret := new(Env)
	ret.NameValuePairs = make(map[string]string, len(self.NameValuePairs))
	for k, v := range self.NameValuePairs {
		ret.NameValuePairs[k] = v
	}
	return ret
}

func (self *Env) Equals(env *Env) bool {
	if (self == nil || len(self.NameValuePairs) == 0) &&
		(env == nil || len(env.NameValuePairs) == 0) {
		return true
	}
	if len(self.NameValuePairs) != len(env.NameValuePairs) {
		return false
	}
	for k, v := range self.NameValuePairs {
		if nv, ok := env.NameValuePairs[k]; ok {
			if nv != v {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func mergeSameEnvs(envs ...*Env) []*Env {
	var set map[string]struct{}
	ret := make([]*Env, 0, len(envs))
	for _, e := range envs {
		if e == nil {
			continue
		}
		data, err := json.Marshal(e)
		if err != nil {
			continue
		}
		hash := sha256.New()
		hash.Write(data)
		sig := fmt.Sprintf("%x", hash.Sum(nil))
		if _, exist := set[sig]; !exist {
			var v struct{}
			set[sig] = v
			ret = append(ret, e)
		}
	}
	return ret
}

func (self *Env) Update(envs ...*Env) {
	merged := mergeSameEnvs(envs...)
	for _, env := range merged {
		for k, v := range env.NameValuePairs {
			self.NameValuePairs[k] = v
		}
	}
}
